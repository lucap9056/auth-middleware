# OAuth2 Module

This module provides an authentication and authorization server, implementing OAuth2 flows and managing JWT-based sessions.

## Built-in Providers

This module has built-in support for the following OAuth2 providers:

- **Discord**
- **Google**
- **Generic** (supports any standard OAuth2 provider via custom configuration)

## Environment Variables

| Variable | Description |
| :--- | :--- |
| `DATABASE_URL` | PostgreSQL connection string |
| `HTTP_ADDRESS` | Server address (default: `:80`) |
| `OAUTH2_PROVIDER` | Provider name (`discord`, `google`, or other) |
| `OAUTH2_CLIENT_ID` | OAuth2 Client ID |
| `OAUTH2_CLIENT_SECRET` | OAuth2 Client Secret |
| `OAUTH2_REDIRECT_URL` | Redirect URL for the OAuth2 flow |
| `OAUTH2_SCOPES` | Comma-separated OAuth2 scopes |
| `OAUTH2_AUTH_URL` | Required for `generic` provider |
| `OAUTH2_TOKEN_URL` | Required for `generic` provider |
| `OAUTH2_USER_INFO_URL` | Required for `generic` provider |
| `OAUTH2_REVOKE_URL` | Optional |
| `REDIS_URL` | Redis connection string (for token caching) |
| `REFRESH_TOKEN_TTL` | TTL for refresh tokens (e.g., `24h`) |
| `HTTP_MODE` | Set to `development` for dev features |
| `ALLOW_REGISTRATION` | Set to `true` to enable user registration |
| `PASS_OAUTH_TOKEN` | Set to `true` to pass OAuth provider tokens to the client (see Operational Scenarios). |

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
