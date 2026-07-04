# OAuth2 Module

This module provides an authentication and authorization server, implementing OAuth2 flows and managing JWT-based sessions.

## Built-in Providers

This module has built-in support for the following OAuth2 providers:

- **Discord**
- **Google**
- **GitHub**
- **Generic OIDC** (any OpenID Connect provider, auto-configured via discovery — see `OIDC_ISSUER_URL`)
- **Generic** (supports any standard OAuth2 provider via custom endpoint configuration)

## Environment Variables

| Variable | Description |
| :--- | :--- |
| `DATABASE_URL` | PostgreSQL connection string |
| `HTTP_ADDRESS` | Server address (default: `:80`) |
| `OAUTH2_PROVIDER` | Provider name (`discord`, `github`, `google`, or other for `generic`) |
| `OAUTH2_CLIENT_ID` | OAuth2 Client ID |
| `OAUTH2_CLIENT_SECRET` | OAuth2 Client Secret |
| `OAUTH2_REDIRECT_URL` | Redirect URL for the OAuth2 flow |
| `OAUTH2_SCOPES` | Comma-separated OAuth2 scopes |
| `OAUTH2_AUTH_URL` | Required for `generic` provider |
| `OAUTH2_TOKEN_URL` | Required for `generic` provider |
| `OAUTH2_USERINFO_URL` | Required for `generic` provider |
| `OAUTH2_REVOKE_URL` | Optional |
| `OIDC_ISSUER_URL` | Enables OIDC discovery mode: auto-configures the auth/token/userinfo/revocation endpoints from `{issuer}/.well-known/openid-configuration`. Requires `OAUTH2_CLIENT_ID`, `OAUTH2_CLIENT_SECRET`, and `OAUTH2_REDIRECT_URL`; takes precedence over `OAUTH2_PROVIDER`. |
| `OAUTH2_CLIENT_PKCE` | Set to `true` to have the client hold the PKCE verifier itself (returned by `/login`, must be sent back via the `X-PKCE-Verifier` header on `/callback`). Defaults to `false`, meaning the server manages the verifier internally — see Login Flow below. |
| `REDIS_URL` | Redis connection string (used for refresh-token caching and OAuth `state` storage; falls back to an in-process cache if unset — not suitable for multi-instance deployments) |
| `REFRESH_TOKEN_TTL` | TTL for refresh tokens (e.g., `24h`) |
| `HTTP_MODE` | Set to `development` for dev features |
| `ALLOW_REGISTRATION` | Set to `true` to enable user registration |
| `PASS_OAUTH_TOKEN` | Set to `true` to pass OAuth provider tokens to the client (see Operational Scenarios). |

## Login Flow (State & PKCE)

`GET /login` starts the flow and returns `{ "url": "..." }`. The server generates the PKCE verifier itself and stores it keyed by a `state` value (in Redis if `REDIS_URL` is set, otherwise in an in-process cache). The `state` is embedded in the returned authorization `url`.

`GET /callback` is invoked by the provider with `code` and `state` query parameters — no client-side PKCE handling is required by default. The server looks up the verifier by `state`, exchanges the code, and deletes the `state` entry (a replayed `state` is rejected).

If you need the client itself to hold the verifier (e.g. a fully client-driven PKCE flow), set `OAUTH2_CLIENT_PKCE=true`: `/login`'s response will then also include a `verifier` field, which the client must send back as the `X-PKCE-Verifier` header when calling `/callback`.

## Operational Scenarios

This module behaves differently based on whether a database is configured and whether token forwarding is enabled:

### 1. Stateful (Pass)
*   **Conditions**: `DATABASE_URL` is set, `PASS_OAUTH_TOKEN=true`.
*   **Behavior**: The module establishes an internal JWT session. The external OAuth2 provider's access/refresh tokens are **also passed to the client** via response headers (`X-Forwarded-Refresh-Token`, `X-Forwarded-Access-Token`).
*   **Responsibility**: The client/downstream service is responsible for securely storing and managing both the internal JWT session and the passed external OAuth tokens.

### 2. Stateful (Revoke)
*   **Conditions**: `DATABASE_URL` is set, `PASS_OAUTH_TOKEN=false` (or unset).
*   **Behavior**: The module establishes an internal JWT session. External OAuth2 tokens are **immediately revoked and discarded** after the internal session is created.
*   **Responsibility**: The module manages all session lifecycle aspects via the internal JWT. External tokens are not accessible to downstream services.

### 3. Stateless (Proxy)
*   **Conditions**: `DATABASE_URL` is unset.
*   **Behavior**: The module acts as a lightweight OAuth2 proxy. No internal session is created. The external OAuth2 tokens are returned directly to the client.
*   **Responsibility**: The client is fully responsible for storing and managing the raw OAuth2 tokens provided by the OAuth2 provider.


## API Endpoints

- `GET /health`: Server health check.
- `POST /refresh`: Refresh session token.
- `POST /refresh-access`: Exchange refresh token for a new access token.
- `GET /verify`: Verify an access token.
- `POST /logout`: Logout and invalidate the session.
- `GET /login`: Start the OAuth2 login flow (only if OAuth2 is enabled).
- `GET /callback`: OAuth2 provider callback endpoint.

## End-to-End Testing

`docker-compose.yml` spins up Postgres, Redis, a local fake OIDC provider (`test-provider/`), and the middleware wired together via `OIDC_ISSUER_URL`:

```
docker compose up --build -d
go test -v ./tests/e2e/
```
