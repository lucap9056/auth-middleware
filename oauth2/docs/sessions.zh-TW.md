# Session

## 登入流程

1. `GET /login` 回傳 `{ "url": "..." }`。Server 產生 PKCE verifier，以一次性的 `state` 為 key 儲存（有設定 `REDIS_URL` 時存在 Redis，否則存在記憶體）。將使用者導向 `url`。
2. Provider 導回 `GET /callback?code=...&state=...`。Server 用 `state` 取出 verifier 並立即刪除，避免 `state` 被重複使用，接著交換 code。
3. 在 session 模式下，server 從 provider 取得使用者 email、建立 device session、設定 `refresh_token` cookie，並回傳兩個 token。

`/callback` 可選的 request header：

| Header | 用途 |
|---|---|
| `X-Device-Name` | 存在 device session 上的名稱（預設 `Unknown Device`） |
| `X-PKCE-Verifier` | `OAUTH2_CLIENT_PKCE=true` 時必填 |

設定 `OAUTH2_CLIENT_PKCE=true` 時，`/login` 會多回傳 `verifier` 欄位，client 必須透過 `X-PKCE-Verifier` 帶回。適用於由 client 自行處理 PKCE 的情境。

## Token

- **Access token**：有效期短（`JWT_ACCESS_TOKEN_DURATION`），以 `Authorization: Bearer <token>` 傳送。Claim 包含 `user_email`、`device_id` 與 `username`。
- **Refresh token**：從 `refresh_token` cookie 讀取，或從 JSON body `{ "refresh_token": "..." }` 讀取。Cookie 為 `HttpOnly`、`SameSite=Lax`，與 token 同時到期。
- **Rotation**：每次 `/refresh` 都會簽發新的 refresh token 並讓舊的失效，持續使用中的 session 會不斷延長。同一個 token 同時呼叫多次 `/refresh` 會共用同一次 rotation（設定 Redis 時跨實例也成立）。
- **重複使用偵測**：送出已被 rotate 過的舊 refresh token 會視為 token 外洩，並刪除該 device session。
- **Username 更新**：登入與每次 refresh 時都會從 users table 讀取 `username`，改名最晚在一個 access token 有效期後反映。

## Provider Token

建立 session 時，`PASS_OAUTH_TOKEN` 決定 provider token 的處理方式：

- `false`（預設）：登入後立即 revoke 並捨棄。
- `true`：透過 `/callback` 回應的 `X-Forwarded-Access-Token` 與 `X-Forwarded-Refresh-Token` header 回傳，由 client 負責保管。

Stateless proxy 模式一定會回傳 provider token：預設放在 response body；`PASS_OAUTH_TOKEN=true` 時只放在上述 header。
