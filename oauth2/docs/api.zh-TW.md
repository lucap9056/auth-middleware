# API

所有 JSON 回應的格式為 `{ "success": bool, "message": ... }`。缺少或無效的 token 會回傳 `401`，並帶有 `WWW-Authenticate: Bearer` header。

| Endpoint | 提供於 | 輸入 | 成功時 |
|---|---|---|---|
| `GET /health` | 所有模式 | — | `200` |
| `GET /login` | 所有模式 | — | `200`，`message`：`{ "url", "verifier"? }` |
| `GET /callback` | 所有模式 | `code`、`state` query | `200`，`message`：`{ "access_token", "refresh_token" }`。Session 模式回傳 session token 並設定 cookie；stateless proxy 回傳 provider 的 token |
| `POST /refresh` | session 模式 | refresh token | `200`，`message`：新的 token pair，並設定 cookie |
| `POST /refresh-access` | session 模式 | refresh token | `200`，`message`：新的 access token |
| `GET /verify` | session 模式 | Bearer access token | `204`，帶有使用者 header（見 [Identity Token](#identity-token)） |
| `POST /logout` | session 模式 | refresh token（選填） | `200`，刪除 device session 並清除 cookie |
| `DELETE /users/me` | managed users | Bearer access token | `200`，刪除所有 session 與該使用者 |

`/verify` 設計給 reverse proxy 的 auth request 使用（例如 nginx `auth_request`、Traefik `forwardAuth`），將回傳的 header 轉發給 upstream service。

## Identity Token

建議設定 `IDENTITY_JWT_SECRET`：`/verify` 會改為回傳簽章過的 header，upstream 使用同一個 secret 驗證，藉此確保 header 的可信度。

未設定時使用者 header 為明文，請確保所有 upstream service 都只能經由 proxy 存取，且 proxy 會清除 client 自帶的同名 header。

| Header | 未設定 `IDENTITY_JWT_SECRET` | 有設定 `IDENTITY_JWT_SECRET` |
|---|---|---|
| `X-Forwarded-User-Email` | 使用者 email | — |
| `X-Forwarded-Device-ID` | Device ID | — |
| `X-Forwarded-Username` | Username，為空時不帶 | — |
| `X-Forwarded-Identity` | — | Identity token |

Identity token 使用 `HS256` 簽章，header `typ` 為 `identity+jwt`。Claims 如下：

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

- `username` 為空時省略；`iss`、`aud` 只在設定 `JWT_ISSUER`、`JWT_AUDIENCE` 時出現
- `exp`：`iat` 後 1 分鐘，若 access token 更早過期則以其為準

## Error Header

當 `401` 是由 token 的 device session 造成時，`/refresh`、`/refresh-access`、`/verify` 與 `DELETE /users/me` 會加上 `X-Auth-Error` header，讓 gateway 能與過期或格式錯誤的 token 區分：

| 值 | 原因 |
|---|---|
| `device_not_found` | Device session 已不存在 |
| `invalid_signature` | Signature 與 device secret 不符 |
