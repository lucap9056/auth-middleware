# Auth Middleware

Device-bound session authentication for Go services, built from three modules that can be used together or on their own.

## Modules

| Module | Import path | What it does |
|---|---|---|
| [`oauth2`](./oauth2/README.md) | service (`oauth2/cmd`) | Authentication server: OAuth2 / OIDC login, session tokens, token verification for a reverse proxy |
| [`jwt`](./jwt/README.md) | `github.com/lucap9056/auth-middleware/jwt/v2` | Issues and verifies access and refresh tokens, each device signed with its own secret |
| [`database`](./database/README.md) | `github.com/lucap9056/auth-middleware/database/v2` | Stores device sessions in PostgreSQL, linked to your users table by email |

```
              ┌──────────────────────── oauth2 ────────────────────────┐
 browser ───▶ │ /login, /callback ──▶ OAuth2 / OIDC provider           │
 proxy   ───▶ │ /verify, /refresh ──▶ jwt ──▶ database ──▶ PostgreSQL  │
              └────────────────────────────────────────────────────────┘
```

- Every login creates a **device session** with its own signing secret. Logging out or deleting a device invalidates only that device's tokens.
- Refresh tokens rotate on every use. A rotated-out refresh token is rejected (`ErrTokenRevoked`).
- Users are identified by **email**. The `auth_user_devices` table references a users table through a foreign key, so deleting a user also removes their sessions.

## Getting Started

### Run the authentication server

The quickest way to see everything working is the `oauth2` compose setup, which starts PostgreSQL, Redis, a fake OIDC provider, and the server:

```bash
cd oauth2
docker compose up --build -d
```

For real deployments, configure `oauth2` through environment variables. It can manage its own users table or use one owned by your application. See [oauth2 → Deployment Modes](./oauth2/README.md#deployment-modes).

### Use the libraries in your own service

```bash
go get github.com/lucap9056/auth-middleware/database/v2
go get github.com/lucap9056/auth-middleware/jwt/v2
```

1. Make sure your users table has an email column that is a `PRIMARY KEY` or `UNIQUE`.
2. Open the device store with `database.NewDatabase(dsn, ...)`, or `database.New(sqlDB, ...)` to share an existing pool. `database.WithAutoCreateSchema(true)` creates the `auth_user_devices` table on startup.
3. Pass it to `jwt.NewJWTManager(db, ...)` to issue and verify tokens.

The schema is created by the `database` module itself. There is no separate SQL file to run. See the [database](./database/README.md) and [jwt](./jwt/README.md) READMEs for details.

## Requirements

- Go 1.24+ for the libraries, Go 1.25+ for `oauth2`
- PostgreSQL 14+
- Redis (optional; required for multi-instance `oauth2` deployments)
