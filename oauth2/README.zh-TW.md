# OAuth2 Module

透過 OAuth2 / OIDC provider 登入，並簽發 device-bound session token 的認證 server，建立在 [`jwt`](../jwt/README.zh-TW.md) 與 [`database`](../database/README.zh-TW.md) module 之上。

## 快速開始

`docker-compose.yml` 會啟動 PostgreSQL、Redis、假的 OIDC provider（`test-provider/`），以及跑在 `http://localhost:8080` 的 server：

```bash
docker compose up --build -d
```

Compose 設定使用 **managed users** 模式並開啟註冊，假 provider 的任何使用者都能登入。

## 部署模式

模式由 `DATABASE_URL` 與 `DB_USER_EMAIL_REFERENCE` 兩個變數決定：

| | Stateless proxy | Managed users | External users |
|---|---|---|---|
| **條件** | 未設定 `DATABASE_URL` | 設定 `DATABASE_URL`，未設定 `DB_USER_EMAIL_REFERENCE` | 兩者都設定 |
| **Users table** | 無 | `users`，由本 server 管理 | 你的 table，以 email 參照 |
| **Session token** | 無，直接回傳 provider 的 token | 有 | 有 |
| **`/refresh`、`/verify`、`/logout`** | 不提供 | 提供 | 提供 |
| **`ALLOW_REGISTRATION`** | 無效 | 把新使用者 insert 到 `users` | 無效（啟動時印出警告） |
| **`DELETE /users/me`** | 不提供 | 刪除使用者與所有 session | 不提供 |
| **`username` claim** | — | `users.username` | `DB_USER_USERNAME_COLUMN`，未設定則為空 |
| **`DB_AUTO_CREATE_SCHEMA=true` 建立的 table** | — | `users`、`auth_user_devices` | `auth_user_devices` |

### Stateless proxy

輕量的 OAuth2 client：client secret、`state` 與 PKCE verifier 都留在 server 端，最後把 provider 的 token 交給 client。不需要 database；`DB_*`、`JWT_*`、`ALLOW_REGISTRATION`、`ALLOW_UNVERIFIED_EMAIL` 都沒有作用。下游 service 必須自行驗證 provider 的 token。

### Managed users

本 server 管理一個 `users` table：

```sql
CREATE TABLE IF NOT EXISTS users (
    user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE
);
```

設定 `ALLOW_REGISTRATION=true` 時，第一次登入的使用者會被 insert，`username` 使用 provider 提供的顯示名稱。未開啟時，只有已存在於 `users` 的 email 能登入。`DB_AUTO_CREATE_SCHEMA` 不是 `true` 時，table 必須事先存在，啟動時會檢查。

### External users

Users table 由你的應用程式管理，透過 `DB_USER_EMAIL_REFERENCE` 告訴本 server 它的位置，例如 `public.accounts(email):citext`。Email column 必須是 `PRIMARY KEY` 或有 `UNIQUE` constraint。本 server 不會寫入這個 table，只有已存在的 email 能登入。在你的應用程式中刪除 user 時，對應的 session 會透過 foreign key 一併刪除。

若要在 `username` claim 放入顯示名稱，將 `DB_USER_USERNAME_COLUMN` 設為同一個 table 中的 column。需要該 table 的 `SELECT` 權限，啟動時會檢查，`NULL` 會視為空字串。

## Provider Token

建立 session 時，`PASS_OAUTH_TOKEN` 決定 provider token 的處理方式：

- `false`（預設）：登入後立即 revoke 並捨棄。
- `true`：透過 `/callback` 回應的 `X-Forwarded-Access-Token` 與 `X-Forwarded-Refresh-Token` header 回傳，由 client 負責保管。

Stateless proxy 模式一定會回傳 provider token：預設放在 response body；`PASS_OAUTH_TOKEN=true` 時只放在上述 header。

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

## Session

- **Access token**：有效期短（`JWT_ACCESS_TOKEN_DURATION`），以 `Authorization: Bearer <token>` 傳送。Claim 包含 `user_email`、`device_id` 與 `username`。
- **Refresh token**：從 `refresh_token` cookie 讀取，或從 JSON body `{ "refresh_token": "..." }` 讀取。Cookie 為 `HttpOnly`、`SameSite=Lax`，與 token 同時到期。
- **Rotation**：每次 `/refresh` 都會簽發新的 refresh token 並讓舊的失效，持續使用中的 session 會不斷延長。同一個 token 同時呼叫多次 `/refresh` 會共用同一次 rotation（設定 Redis 時跨實例也成立）。
- **Username 更新**：登入與每次 refresh 時都會從 users table 讀取 `username`，改名最晚在一個 access token 有效期後反映。

## API

所有 JSON 回應的格式為 `{ "success": bool, "message": ... }`。缺少或無效的 token 會回傳 `401`，並帶有 `WWW-Authenticate: Bearer` header。

| Endpoint | 提供於 | 輸入 | 成功時 |
|---|---|---|---|
| `GET /health` | 所有模式 | — | `200` |
| `GET /login` | 所有模式 | — | `200`，`message`：`{ "url", "verifier"? }` |
| `GET /callback` | 所有模式 | `code`、`state` query | `200`，`message`：`{ "access_token", "refresh_token" }`。Session 模式回傳 session token 並設定 cookie；stateless proxy 回傳 provider 的 token |
| `POST /refresh` | session 模式 | refresh token | `200`，`message`：新的 token pair，並設定 cookie |
| `POST /refresh-access` | session 模式 | refresh token | `200`，`message`：新的 access token |
| `GET /verify` | session 模式 | Bearer access token | `204`，帶有 `X-Forwarded-User-Email`、`X-Forwarded-Device-ID`，`username` 不為空時另帶 `X-Forwarded-Username` |
| `POST /logout` | session 模式 | refresh token（選填） | `200`，刪除 device session 並清除 cookie |
| `DELETE /users/me` | managed users | Bearer access token | `200`，刪除所有 session 與該使用者 |

`/verify` 設計給 reverse proxy 的 auth request 使用（例如 nginx `auth_request`、Traefik `forwardAuth`），將回傳的 header 轉發給 upstream service。

## 設定

時間格式由一或多段 `<整數><單位>` 組成，單位為 `d`、`h`、`m`、`s`（例如 `30m`、`7d`、`1d12h`）。格式錯誤時 server 會在啟動時失敗。

### Server

| 變數 | 預設值 | 說明 |
|---|---|---|
| `HTTP_ADDRESS` | `:80` | 監聽位址。`unix:///path/to.sock` 會改為監聽 Unix socket |
| `HTTP_MODE` | `production` | 設為 `development` 時 `refresh_token` cookie 不會標記 `Secure`，用於在本機以 HTTP 測試 |
| `REDIS_URL` | — | 儲存 `state`、快取 device secret，並跨實例去除重複的 `/callback` 與 `/refresh`。未設定時改用記憶體實作，只適用單一實例 |

### OAuth2 Provider

內建 provider：`discord`、`github`、`google`，`OAUTH2_PROVIDER` 為其他值時使用 generic provider。設定 `OIDC_ISSUER_URL` 會改用 OIDC discovery，並優先於 `OAUTH2_PROVIDER`。

| 變數 | 說明 |
|---|---|
| `OAUTH2_CLIENT_ID`、`OAUTH2_CLIENT_SECRET`、`OAUTH2_REDIRECT_URL` | Client credentials 與向 provider 註冊的 redirect URL。三者都設定時才會啟用 OAuth2 |
| `OAUTH2_PROVIDER` | `discord`、`github`、`google`，或其他名稱表示 generic provider |
| `OAUTH2_SCOPES` | 以逗號分隔的 scope |
| `OAUTH2_AUTH_URL`、`OAUTH2_TOKEN_URL`、`OAUTH2_USERINFO_URL` | Generic provider 必填 |
| `OAUTH2_REVOKE_URL` | 選填，用於 revoke provider token |
| `OIDC_ISSUER_URL` | 從 `{issuer}/.well-known/openid-configuration` 讀取 endpoint |
| `OAUTH2_CLIENT_PKCE` | `true` 時由 client 持有 PKCE verifier（見[登入流程](#登入流程)） |

### Database

| 變數 | 預設值 | 說明 |
|---|---|---|
| `DATABASE_URL` | — | PostgreSQL 連線字串。未設定時為 stateless proxy 模式 |
| `DB_USER_EMAIL_REFERENCE` | — | 你的 users table，格式為 `[<schema>.]<table>(<column>)[:<type>]`（見 [database → User Email Reference](../database/README.zh-TW.md#user-email-reference)）。未設定時為 managed users 模式 |
| `DB_USER_USERNAME_COLUMN` | — | 該 table 中作為 `username` claim 的 column。需要同時設定 `DB_USER_EMAIL_REFERENCE` |
| `DB_AUTO_CREATE_SCHEMA` | `false` | `true` 時於啟動時建立 table（見[部署模式](#部署模式)），需要 DDL 權限 |
| `DB_CLEANUP_INTERVAL` | `24h` | 刪除 7 天未 refresh 的 device session 的頻率。純整數視為小時 |
| `DB_MAX_OPEN_CONNS` | `20` | 最大連線數 |
| `DB_MAX_IDLE_CONNS` | `15` | 最大閒置連線數 |
| `DB_CONN_MAX_LIFETIME` | `5m` | 單一連線最長存活時間。純整數視為分鐘 |
| `DB_CONN_MAX_IDLE_TIME` | `2m` | 單一連線最長閒置時間。純整數視為分鐘 |

### Token

| 變數 | 預設值 | 說明 |
|---|---|---|
| `JWT_ACCESS_TOKEN_DURATION` | `15m` | Access token 有效期，必須帶單位 |
| `JWT_REFRESH_TOKEN_DURATION` | `7d` | Refresh token 與 cookie 的有效期，必須帶單位 |
| `JWT_ISSUER` | — | 簽發 token 時寫入的 `iss` claim，驗證時也會檢查 |
| `JWT_AUDIENCE` | — | 簽發 token 時寫入的 `aud` claim，驗證時也會檢查 |

### 登入政策

| 變數 | 預設值 | 說明 |
|---|---|---|
| `ALLOW_REGISTRATION` | `false` | 將第一次登入的使用者 insert 到 `users`（僅 managed users 模式） |
| `ALLOW_UNVERIFIED_EMAIL` | `false` | 接受 provider 未驗證的 email。預設要求 `email_verified` 為 true |
| `PASS_OAUTH_TOKEN` | `false` | 將 provider token 回傳給 client（見 [Provider Token](#provider-token)） |

## 測試

```bash
go test ./internal/... ./tests/
```

Unit 與 integration test 不需要外部服務。其他測試需要對應的環境：

| 測試 | 需求 | 指令 |
|---|---|---|
| `internal/cache/device` 與 `internal/flight` 的 Redis test | `TEST_REDIS_URL` | `go test ./internal/...`（未設定時 skip） |
| End-to-end 流程 | Compose 環境已啟動 | `go test ./tests/e2e/` |
