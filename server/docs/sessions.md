# Sessions

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

## Tokens

- **Access token**: short-lived (`JWT_ACCESS_TOKEN_DURATION`). Send it as `Authorization: Bearer <token>`. Claims include `user_email`, `device_id`, and `username`.
- **Refresh token**: read from the `refresh_token` cookie, or from a JSON body `{ "refresh_token": "..." }`. The cookie is `HttpOnly`, `SameSite=Lax`, and expires with the token.
- **Rotation**: every `/refresh` issues a new refresh token and invalidates the previous one, so active sessions keep extending. Concurrent `/refresh` calls with the same token share one rotation (across instances when Redis is configured).
- **Reuse detection**: presenting an already-rotated refresh token is treated as token theft and deletes the device session.
- **Username updates**: `username` is read from the users table on login and on every refresh, so renames show up within one access token lifetime.

## Provider Tokens

When a session is created, `PASS_OAUTH_TOKEN` decides what happens to the provider's tokens:

- `false` (default): they are revoked and discarded right after login.
- `true`: they are returned in the `X-Forwarded-Access-Token` and `X-Forwarded-Refresh-Token` response headers of `/callback`, and the client is responsible for them.

In stateless proxy mode the provider's tokens are always returned: in the response body by default, or only in the headers above when `PASS_OAUTH_TOKEN=true`.
