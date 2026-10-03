# 設定

時間格式由一或多段 `<整數><單位>` 組成，單位為 `d`、`h`、`m`、`s`（例如 `30m`、`7d`、`1d12h`）。格式錯誤時 server 會在啟動時失敗。

## Server

| 變數 | 預設值 | 說明 |
|---|---|---|
| `HTTP_ADDRESS` | `:80` | 監聽位址。`unix:///path/to.sock` 會改為監聽 Unix socket |
| `HTTP_MODE` | `production` | 設為 `development` 時 `refresh_token` cookie 不會標記 `Secure`，用於在本機以 HTTP 測試 |
| `REDIS_URL` | — | 儲存 `state`、快取 device secret，並跨實例去除重複的 `/callback` 與 `/refresh`。未設定時改用記憶體實作，只適用單一實例 |

## OAuth2 Provider

內建 provider：`discord`、`github`、`google`，`OAUTH2_PROVIDER` 為其他值時使用 generic provider。設定 `OIDC_ISSUER_URL` 會改用 OIDC discovery，並優先於 `OAUTH2_PROVIDER`。

| 變數 | 說明 |
|---|---|
| `OAUTH2_CLIENT_ID`、`OAUTH2_CLIENT_SECRET`、`OAUTH2_REDIRECT_URL` | Client credentials 與向 provider 註冊的 redirect URL。三者都設定時才會啟用 OAuth2 |
| `OAUTH2_PROVIDER` | `discord`、`github`、`google`，或其他名稱表示 generic provider |
| `OAUTH2_SCOPES` | 以逗號分隔的 scope |
| `OAUTH2_AUTH_URL`、`OAUTH2_TOKEN_URL`、`OAUTH2_USERINFO_URL` | Generic provider 必填 |
| `OAUTH2_REVOKE_URL` | 選填，用於 revoke provider token |
| `OIDC_ISSUER_URL` | 從 `{issuer}/.well-known/openid-configuration` 讀取 endpoint |
| `OAUTH2_CLIENT_PKCE` | `true` 時由 client 持有 PKCE verifier（見[登入流程](sessions.zh-TW.md#登入流程)） |

## Database

| 變數 | 預設值 | 說明 |
|---|---|---|
| `DATABASE_URL` | — | PostgreSQL 連線字串。未設定時為 stateless proxy 模式 |
| `DB_USER_EMAIL_REFERENCE` | — | 你的 users table，格式為 `[<schema>.]<table>(<column>)[:<type>]`（見 [database → User Email Reference](../../database/README.zh-TW.md#user-email-reference)）。未設定時為 managed users 模式 |
| `DB_USER_USERNAME_COLUMN` | — | 該 table 中作為 `username` claim 的 column。需要同時設定 `DB_USER_EMAIL_REFERENCE` |
| `DB_AUTO_CREATE_SCHEMA` | `false` | `true` 時於啟動時建立 table（見[部署模式](../README.zh-TW.md#部署模式)），需要 DDL 權限 |
| `DB_CLEANUP_INTERVAL` | `24h` | 刪除 7 天未 refresh 的 device session 的頻率。純整數視為小時 |
| `DB_MAX_OPEN_CONNS` | `20` | 最大連線數 |
| `DB_MAX_IDLE_CONNS` | `15` | 最大閒置連線數 |
| `DB_CONN_MAX_LIFETIME` | `5m` | 單一連線最長存活時間。純整數視為分鐘 |
| `DB_CONN_MAX_IDLE_TIME` | `2m` | 單一連線最長閒置時間。純整數視為分鐘 |

## Token

| 變數 | 預設值 | 說明 |
|---|---|---|
| `JWT_ACCESS_TOKEN_DURATION` | `15m` | Access token 有效期，必須帶單位 |
| `JWT_REFRESH_TOKEN_DURATION` | `7d` | Refresh token 與 cookie 的有效期，必須帶單位 |
| `JWT_ISSUER` | — | 簽發 token 時寫入的 `iss` claim，驗證時也會檢查 |
| `JWT_AUDIENCE` | — | 簽發 token 時寫入的 `aud` claim，驗證時也會檢查 |

## 登入政策

| 變數 | 預設值 | 說明 |
|---|---|---|
| `ALLOW_REGISTRATION` | `false` | 將第一次登入的使用者 insert 到 `users`（僅 managed users 模式） |
| `ALLOW_UNVERIFIED_EMAIL` | `false` | 接受 provider 未驗證的 email。預設要求 `email_verified` 為 true |
| `PASS_OAUTH_TOKEN` | `false` | 將 provider token 回傳給 client（見 [Provider Token](sessions.zh-TW.md#provider-token)） |
