# API

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

## Error Headers

When a `401` is caused by the token's device session, `/refresh`, `/refresh-access`, `/verify`, and `DELETE /users/me` add an `X-Auth-Error` header, so a gateway can tell these failures apart from expired or malformed tokens:

| Value | Cause |
|---|---|
| `device_not_found` | The device session no longer exists |
| `invalid_signature` | The signature does not match the device secret |
