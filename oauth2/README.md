# OAuth2 Module

An authentication server that signs users in through an OAuth2 / OIDC provider and issues device-bound session tokens, built on the [`jwt`](../jwt/README.md) and [`database`](../database/README.md) modules.

## Quick Start

`docker-compose.yml` starts PostgreSQL, Redis, a fake OIDC provider (`test-provider/`), and the server on `http://localhost:8080`:

```bash
docker compose up --build -d
```

The compose setup uses the **managed users** mode with registration enabled, so any user of the fake provider can sign in.

## Deployment Modes

The mode is selected by two variables, `DATABASE_URL` and `DB_USER_EMAIL_REFERENCE`:

| | Stateless proxy | Managed users | External users |
|---|---|---|---|
| **Set when** | `DATABASE_URL` unset | `DATABASE_URL` set, `DB_USER_EMAIL_REFERENCE` unset | both set |
| **Users table** | none | `users`, owned by this server | yours, referenced by email |
| **Session tokens** | no, provider tokens are returned as-is | yes | yes |
| **`/refresh`, `/verify`, `/logout`** | not available | available | available |
| **`ALLOW_REGISTRATION`** | ignored | inserts unknown users into `users` | ignored (logged as a warning) |
| **`DELETE /users/me`** | not available | deletes the user and all sessions | not available |
| **`username` claim** | — | `users.username` | `DB_USER_USERNAME_COLUMN`, or empty |
| **`DB_AUTO_CREATE_SCHEMA=true` creates** | — | `users`, `auth_user_devices` | `auth_user_devices` |

### Stateless proxy

A lightweight OAuth2 client that keeps the client secret, `state`, and PKCE verifier on the server and hands the provider's tokens to the client. No database is needed; `DB_*`, `JWT_*`, `ALLOW_REGISTRATION`, and `ALLOW_UNVERIFIED_EMAIL` have no effect. Downstream services must validate the provider's tokens themselves.

### Managed users

This server owns a `users` table:

```sql
CREATE TABLE IF NOT EXISTS users (
    user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE
);
```

With `ALLOW_REGISTRATION=true`, a user who signs in for the first time is inserted with the provider's display name as `username`. Without it, only emails already in `users` can sign in. When `DB_AUTO_CREATE_SCHEMA` is not `true`, the table must already exist; this is checked on startup.

### External users

Your application owns the users table, and `DB_USER_EMAIL_REFERENCE` tells this server where it is, e.g. `public.accounts(email):citext`. The email column must be a `PRIMARY KEY` or `UNIQUE`. This server never writes to that table: only emails that already exist can sign in. Deleting a user in your application removes their sessions through the foreign key.

To put a display name in the `username` claim, set `DB_USER_USERNAME_COLUMN` to a column of the same table. This requires `SELECT` privilege on it, is checked on startup, and `NULL` becomes an empty string.

## Provider Tokens

When a session is created, `PASS_OAUTH_TOKEN` decides what happens to the provider's tokens:

- `false` (default): they are revoked and discarded right after login.
- `true`: they are returned in the `X-Forwarded-Access-Token` and `X-Forwarded-Refresh-Token` response headers of `/callback`, and the client is responsible for them.

In stateless proxy mode the provider's tokens are always returned: in the response body by default, or only in the headers above when `PASS_OAUTH_TOKEN=true`.

## Login Flow

1. `GET /login` returns `{ "url": "..." }`. The server generates a PKCE verifier and stores it under a one-time `state` (in Redis when `REDIS_URL` is set, otherwise in memory). Redirect the user to `url`.
2. The provider redirects back to `GET /callback?code=...&state=...`. The server looks up the verifier by `state`, deletes it so the `state` cannot be replayed, and exchanges the code.
3. In the session modes, the server reads the user's email from the provider, creates a device session, sets the `refresh_token` cookie, and returns both tokens.

Optional `/callback` request headers:

| Header | Purpose |
|---|---|
| `X-Device-Name` | Name stored with the device session (default `Unknown Device`) |
| `X-PKCE-Verifier` | Required when `OAUTH2_CLIENT_PKCE=true` |

With `OAUTH2_CLIENT_PKCE=true`, `/login` also returns a `verifier` field, and the client must send it back in `X-PKCE-Verifier`. Use this when the client drives PKCE itself.

## Sessions

- **Access token**: short-lived (`JWT_ACCESS_TOKEN_DURATION`). Send it as `Authorization: Bearer <token>`. Claims include `user_email`, `device_id`, and `username`.
- **Refresh token**: read from the `refresh_token` cookie, or from a JSON body `{ "refresh_token": "..." }`. The cookie is `HttpOnly`, `SameSite=Lax`, and expires with the token.
- **Rotation**: every `/refresh` issues a new refresh token and invalidates the previous one, so active sessions keep extending. Concurrent `/refresh` calls with the same token share one rotation (across instances when Redis is configured).
- **Username updates**: `username` is read from the users table on login and on every refresh, so renames show up within one access token lifetime.

## API

All JSON responses have the shape `{ "success": bool, "message": ... }`. A missing or invalid token returns `401` with a `WWW-Authenticate: Bearer` header.

| Endpoint | Available in | Input | Success |
|---|---|---|---|
| `GET /health` | all modes | — | `200` |
| `GET /login` | all modes | — | `200`, `message`: `{ "url", "verifier"? }` |
| `GET /callback` | all modes | `code`, `state` query | `200`, `message`: `{ "access_token", "refresh_token" }`. Session modes return session tokens and set the cookie; stateless proxy returns the provider's tokens |
| `POST /refresh` | session modes | refresh token | `200`, `message`: new token pair, sets cookie |
| `POST /refresh-access` | session modes | refresh token | `200`, `message`: new access token |
| `GET /verify` | session modes | Bearer access token | `204` with `X-Forwarded-User-Email`, `X-Forwarded-Device-ID`, and `X-Forwarded-Username` when not empty |
| `POST /logout` | session modes | refresh token (optional) | `200`, deletes the device session and clears the cookie |
| `DELETE /users/me` | managed users | Bearer access token | `200`, deletes all sessions and the user |

`/verify` is meant for a reverse proxy's auth request (e.g. nginx `auth_request`, Traefik `forwardAuth`): forward the returned headers to the upstream service.

## Configuration

Durations accept one or more `<integer><unit>` segments with units `d`, `h`, `m`, `s` (e.g. `30m`, `7d`, `1d12h`). Invalid values stop the server on startup.

### Server

| Variable | Default | Description |
|---|---|---|
| `HTTP_ADDRESS` | `:80` | Listen address. `unix:///path/to.sock` listens on a Unix socket |
| `HTTP_MODE` | `development` | In `development`, the `refresh_token` cookie is not marked `Secure`. Set to any other value (e.g. `production`) when serving over HTTPS |
| `REDIS_URL` | — | Stores `state`, caches device secrets, and deduplicates `/callback` and `/refresh` across instances. Without it, in-memory equivalents are used, which only work with a single instance |

### OAuth2 Provider

Built-in providers: `discord`, `github`, `google`. Any other `OAUTH2_PROVIDER` value uses the generic provider. Setting `OIDC_ISSUER_URL` switches to OIDC discovery and takes precedence over `OAUTH2_PROVIDER`.

| Variable | Description |
|---|---|
| `OAUTH2_CLIENT_ID`, `OAUTH2_CLIENT_SECRET`, `OAUTH2_REDIRECT_URL` | Client credentials and the redirect URL registered with the provider. OAuth2 is enabled only when all three are set |
| `OAUTH2_PROVIDER` | `discord`, `github`, `google`, or any other name for the generic provider |
| `OAUTH2_SCOPES` | Comma-separated scopes |
| `OAUTH2_AUTH_URL`, `OAUTH2_TOKEN_URL`, `OAUTH2_USERINFO_URL` | Required for the generic provider |
| `OAUTH2_REVOKE_URL` | Optional, used to revoke provider tokens |
| `OIDC_ISSUER_URL` | Reads the endpoints from `{issuer}/.well-known/openid-configuration` |
| `OAUTH2_CLIENT_PKCE` | `true` to let the client hold the PKCE verifier (see [Login Flow](#login-flow)) |

### Database

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | — | PostgreSQL connection string. Unset selects the stateless proxy mode |
| `DB_USER_EMAIL_REFERENCE` | — | Your users table, `[<schema>.]<table>(<column>)[:<type>]` (see [database → User Email Reference](../database/README.md#user-email-reference)). Unset selects the managed users mode |
| `DB_USER_USERNAME_COLUMN` | — | Column of that table used as the `username` claim. Requires `DB_USER_EMAIL_REFERENCE` |
| `DB_AUTO_CREATE_SCHEMA` | `false` | `true` to create the tables on startup (see [Deployment Modes](#deployment-modes)). Requires DDL privileges |
| `DB_CLEANUP_INTERVAL` | `24h` | How often device sessions not refreshed for 7 days are deleted. A bare integer is read as hours |
| `DB_MAX_OPEN_CONNS` | `20` | Maximum open connections |
| `DB_MAX_IDLE_CONNS` | `15` | Maximum idle connections |
| `DB_CONN_MAX_LIFETIME` | `5m` | Maximum connection lifetime. A bare integer is read as minutes |
| `DB_CONN_MAX_IDLE_TIME` | `2m` | Maximum connection idle time. A bare integer is read as minutes |

### Tokens

| Variable | Default | Description |
|---|---|---|
| `JWT_ACCESS_TOKEN_DURATION` | `15m` | Access token lifetime. A unit is required |
| `JWT_REFRESH_TOKEN_DURATION` | `7d` | Refresh token and cookie lifetime. A unit is required |
| `JWT_ISSUER` | — | `iss` claim set on issued tokens and required on verification |
| `JWT_AUDIENCE` | — | `aud` claim set on issued tokens and required on verification |

### Sign-in Policy

| Variable | Default | Description |
|---|---|---|
| `ALLOW_REGISTRATION` | `false` | Insert first-time users into `users` (managed users mode only) |
| `ALLOW_UNVERIFIED_EMAIL` | `false` | Accept emails the provider has not verified. By default `email_verified` must be true |
| `PASS_OAUTH_TOKEN` | `false` | Return the provider's tokens to the client (see [Provider Tokens](#provider-tokens)) |

## Testing

```bash
go test ./internal/... ./tests/
```

Unit and integration tests run without external services. The other suites need their environment:

| Suite | Requirement | Command |
|---|---|---|
| Redis tests in `internal/cache/device` and `internal/flight` | `TEST_REDIS_URL` | `go test ./internal/...` (skipped when unset) |
| End-to-end flow | the compose setup running | `go test ./tests/e2e/` |
