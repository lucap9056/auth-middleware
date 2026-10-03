# Configuration

Durations accept one or more `<integer><unit>` segments with units `d`, `h`, `m`, `s` (e.g. `30m`, `7d`, `1d12h`). Invalid values stop the server on startup.

## Server

| Variable | Default | Description |
|---|---|---|
| `HTTP_ADDRESS` | `:80` | Listen address. `unix:///path/to.sock` listens on a Unix socket |
| `HTTP_MODE` | `production` | Set to `development` to stop marking the `refresh_token` cookie `Secure`, for local testing over plain HTTP |
| `REDIS_URL` | — | Stores `state`, caches device secrets, and deduplicates `/callback` and `/refresh` across instances. Without it, in-memory equivalents are used, which only work with a single instance |

## OAuth2 Provider

Built-in providers: `discord`, `github`, `google`. Any other `OAUTH2_PROVIDER` value uses the generic provider. Setting `OIDC_ISSUER_URL` switches to OIDC discovery and takes precedence over `OAUTH2_PROVIDER`.

| Variable | Description |
|---|---|
| `OAUTH2_CLIENT_ID`, `OAUTH2_CLIENT_SECRET`, `OAUTH2_REDIRECT_URL` | Client credentials and the redirect URL registered with the provider. OAuth2 is enabled only when all three are set |
| `OAUTH2_PROVIDER` | `discord`, `github`, `google`, or any other name for the generic provider |
| `OAUTH2_SCOPES` | Comma-separated scopes |
| `OAUTH2_AUTH_URL`, `OAUTH2_TOKEN_URL`, `OAUTH2_USERINFO_URL` | Required for the generic provider |
| `OAUTH2_REVOKE_URL` | Optional, used to revoke provider tokens |
| `OIDC_ISSUER_URL` | Reads the endpoints from `{issuer}/.well-known/openid-configuration` |
| `OAUTH2_CLIENT_PKCE` | `true` to let the client hold the PKCE verifier (see [Login Flow](sessions.md#login-flow)) |

## Database

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | — | PostgreSQL connection string. Unset selects the stateless proxy mode |
| `DB_USER_EMAIL_REFERENCE` | — | Your users table, `[<schema>.]<table>(<column>)[:<type>]` (see [database → User Email Reference](../../database/README.md#user-email-reference)). Unset selects the managed users mode |
| `DB_USER_USERNAME_COLUMN` | — | Column of that table used as the `username` claim. Requires `DB_USER_EMAIL_REFERENCE` |
| `DB_AUTO_CREATE_SCHEMA` | `false` | `true` to create the tables on startup (see [Deployment Modes](../README.md#deployment-modes)). Requires DDL privileges |
| `DB_CLEANUP_INTERVAL` | `24h` | How often device sessions not refreshed for 7 days are deleted. A bare integer is read as hours |
| `DB_MAX_OPEN_CONNS` | `20` | Maximum open connections |
| `DB_MAX_IDLE_CONNS` | `15` | Maximum idle connections |
| `DB_CONN_MAX_LIFETIME` | `5m` | Maximum connection lifetime. A bare integer is read as minutes |
| `DB_CONN_MAX_IDLE_TIME` | `2m` | Maximum connection idle time. A bare integer is read as minutes |

## Tokens

| Variable | Default | Description |
|---|---|---|
| `JWT_ACCESS_TOKEN_DURATION` | `15m` | Access token lifetime. A unit is required |
| `JWT_REFRESH_TOKEN_DURATION` | `7d` | Refresh token and cookie lifetime. A unit is required |
| `JWT_ISSUER` | — | `iss` claim set on issued tokens and required on verification |
| `JWT_AUDIENCE` | — | `aud` claim set on issued tokens and required on verification |
| `IDENTITY_JWT_SECRET` | — | At least 32 bytes. When set, `/verify` returns a signed identity token instead of plain user headers (see [Identity Token](api.md#identity-token)) |

## Sign-in Policy

| Variable | Default | Description |
|---|---|---|
| `ALLOW_REGISTRATION` | `false` | Insert first-time users into `users` (managed users mode only) |
| `ALLOW_UNVERIFIED_EMAIL` | `false` | Accept emails the provider has not verified. By default `email_verified` must be true |
| `PASS_OAUTH_TOKEN` | `false` | Return the provider's tokens to the client (see [Provider Tokens](sessions.md#provider-tokens)) |
