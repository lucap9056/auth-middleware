# API

All JSON responses have the shape `{ "success": bool, "message": ... }`. A missing or invalid token returns `401` with a `WWW-Authenticate: Bearer` header.

| Endpoint | Available in | Input | Success |
|---|---|---|---|
| `GET /health` | all modes | — | `200` |
| `GET /login` | all modes | — | `200`, `message`: `{ "url", "verifier"? }` |
| `GET /callback` | all modes | `code`, `state` query | `200`, `message`: `{ "access_token", "refresh_token" }`. Session modes return session tokens and set the cookie; stateless proxy returns the provider's tokens |
| `POST /refresh` | session modes | refresh token | `200`, `message`: new token pair, sets cookie |
| `POST /refresh-access` | session modes | refresh token | `200`, `message`: new access token |
| `GET /verify` | session modes | Bearer access token | `204` with the user headers (see [Identity Token](#identity-token)) |
| `POST /logout` | session modes | refresh token (optional) | `200`, deletes the device session and clears the cookie |
| `DELETE /users/me` | managed users | Bearer access token | `200`, deletes all sessions and the user |

`/verify` is meant for a reverse proxy's auth request (e.g. nginx `auth_request`, Traefik `forwardAuth`): forward the returned headers to the upstream service.

## Identity Token

Setting `IDENTITY_JWT_SECRET` is recommended: `/verify` then returns a signed header that the upstream verifies with the same secret, so the user headers stay trustworthy.

Without it, the user headers are plain text. Make sure every upstream service is only reachable through the proxy, and that the proxy strips client-supplied copies of these headers.

| Header | `IDENTITY_JWT_SECRET` unset | `IDENTITY_JWT_SECRET` set |
|---|---|---|
| `X-Forwarded-User-Email` | User email | — |
| `X-Forwarded-Device-ID` | Device ID | — |
| `X-Forwarded-Username` | Username, omitted when empty | — |
| `X-Forwarded-Identity` | — | Identity token |

The identity token is signed with `HS256` and has the header `typ`: `identity+jwt`. Its claims:

```json
{
  "sub": "user@example.com",
  "username": "User",
  "device_id": "3f1c2a9e-...",
  "iss": "https://auth.example.com",
  "aud": ["api"],
  "iat": 1767225600,
  "exp": 1767225660
}
```

- `username` is omitted when empty; `iss` / `aud` are present only when `JWT_ISSUER` / `JWT_AUDIENCE` are set
- `exp` is 1 minute after `iat`, or the access token's expiry if sooner. The proxy calls `/verify` on every request, so a fresh token is issued each time

The upstream should check the signature, `alg`, `typ`, `exp`, and `iss` / `aud` if configured. Any service holding the secret can also sign identity tokens, so share it only with trusted upstreams.

## Error Headers

When a `401` is caused by the token's device session, `/refresh`, `/refresh-access`, `/verify`, and `DELETE /users/me` add an `X-Auth-Error` header, so a gateway can tell these failures apart from expired or malformed tokens:

| Value | Cause |
|---|---|
| `device_not_found` | The device session no longer exists |
| `invalid_signature` | The signature does not match the device secret |
