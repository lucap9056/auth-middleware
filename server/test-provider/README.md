# test-provider

A minimal OIDC/OAuth2.0 provider for integration testing. Implements the
Authorization Code + PKCE flow with no credential verification — `/auth`
issues an authorization code immediately and redirects back to
`redirect_uri`. User info is fixed via environment variables, making it
easy to auto-complete login flows in E2E tests.

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET  | `/.well-known/openid-configuration` | OIDC discovery document |
| GET  | `/auth` | Authorization endpoint, issues a code and 302 redirects to `redirect_uri` |
| POST | `/token` | Exchanges the code for an access token / id token (verifies PKCE) |
| GET  | `/userinfo` | Returns the fixed user info |

The id_token is signed with `alg: none` — for parsing test fields only,
with no security guarantees.

## Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ISSUER_URL` | `http://test-provider:5556` | Issuer / base URL for all endpoints in the discovery document |
| `LISTEN_ADDR` | `:5556` | Listen address; supports a `unix://` prefix for Unix domain sockets |
| `USER_SUB` | `test-user-001` | Fixed user `sub` returned |
| `USER_EMAIL` | `user@example.com` | Fixed user email returned |
| `USER_NAME` | `Test User` | Fixed user name returned |

## Building the image

```bash
docker build -t corvauth-test-provider:local .
```

The container exposes port `5556`.

## Use case

Serves as an integration test target for OIDC/OAuth2.0 client, letting
E2E tests exercise a full OAuth2.0/OIDC authorization flow without relying
on an external third-party login service.
