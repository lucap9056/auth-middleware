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

## Documentation

- [Deployment Modes](docs/deployment-modes.md): users table setup for each mode
- [Sessions](docs/sessions.md): login flow, tokens, rotation, provider tokens
- [API](docs/api.md): endpoints and error headers
- [Configuration](docs/configuration.md): environment variables

## Testing

```bash
go test ./internal/... ./tests/
```

Unit and integration tests run without external services. The other suites need their environment:

| Suite | Requirement | Command |
|---|---|---|
| Redis tests in `internal/cache/device` and `internal/flight` | `TEST_REDIS_URL` | `go test ./internal/...` (skipped when unset) |
| End-to-end flow | the compose setup running | `go test ./tests/e2e/` |
