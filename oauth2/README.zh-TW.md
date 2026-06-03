# OAuth2 模組

此模組提供了一個認證與授權伺服器，實作了 OAuth2 流程並管理基於 JWT 的會話。

## 內建服務供應商

此模組內建支援以下 OAuth2 服務：

- **Discord**
- **Google**
- **Generic** (透過自訂設定支援任何標準 OAuth2 服務)

## 環境變數 (ENV)

| 變數 | 說明 |
| :--- | :--- |
| `DATABASE_URL` | PostgreSQL 連線字串 |
| `HTTP_ADDRESS` | 伺服器位址 (預設: `:80`) |
| `OAUTH2_PROVIDER` | 服務供應商名稱 (`discord`, `google`, 或其他) |
| `OAUTH2_CLIENT_ID` | OAuth2 Client ID |
| `OAUTH2_CLIENT_SECRET` | OAuth2 Client Secret |
| `OAUTH2_REDIRECT_URL` | OAuth2 流程的重新導向網址 |
| `OAUTH2_SCOPES` | 以逗號分隔的 OAuth2 權限範圍 (Scopes) |
| `OAUTH2_AUTH_URL` | `generic` 供應商必要設定 |
| `OAUTH2_TOKEN_URL` | `generic` 供應商必要設定 |
| `OAUTH2_USER_INFO_URL` | `generic` 供應商必要設定 |
| `OAUTH2_REVOKE_URL` | 選填 |
| `REDIS_URL` | Redis 連線字串 (用於權杖快取) |
| `REFRESH_TOKEN_TTL` | Refresh token 的過期時間 (例如 `24h`) |
| `HTTP_MODE` | 設定為 `development` 以啟用開發功能 |
| `ALLOW_REGISTRATION` | 設定為 `true` 以允許使用者註冊 |
| `PASS_OAUTH_TOKEN` | 設定為 `true` 以將 OAuth 供應商權杖傳遞給用戶端 (詳見操作情境)。 |

## 操作情境

本模組根據資料庫配置與權杖傳遞設定的不同，會有三種運作情境：

### 1. 有狀態 (傳遞)
*   **條件**: 已設定 `DATABASE_URL`，`PASS_OAUTH_TOKEN=true`。
*   **行為**: 模組會建立內部 JWT 會話。外部 OAuth2 供應商的 Access/Refresh Token **也會透過回應 Header** (`X-Forwarded-Refresh-Token`, `X-Forwarded-Access-Token`) 傳遞給用戶端。
*   **責任**: 用戶端或下游服務必須自行負責安全地儲存與管理內部 JWT 會話與傳遞出去的外部 OAuth 權杖。

### 2. 有狀態 (撤銷)
*   **條件**: 已設定 `DATABASE_URL`，`PASS_OAUTH_TOKEN=false` (或未設定)。
*   **行為**: 模組會建立內部 JWT 會話。在內部會話建立後，外部 OAuth2 權杖會被**立即撤銷 (Revoke) 並捨棄**。
*   **責任**: 本模組透過內部 JWT 管理所有會話生命週期。下游服務無法存取外部權杖。

### 3. 無狀態 (代理)
*   **條件**: 未設定 `DATABASE_URL`。
*   **行為**: 模組僅作為輕量級 OAuth2 代理。不建立內部會話，直接將外部 OAuth2 權杖返回給用戶端。
*   **責任**: 用戶端完全負責儲存與管理由 OAuth2 供應商提供的原始權杖。


## API 端點 (Endpoints)

- `GET /health`: 伺服器健康檢查。
- `POST /refresh`: 更新會話權杖。
- `POST /refresh-access`: 使用 Refresh token 換取新的 Access token。
- `GET /verify`: 驗證 Access token。
- `POST /logout`: 登出並失效化會話。
- `GET /login`: 啟動 OAuth2 登入流程 (僅在啟用 OAuth2 時可用)。
- `GET /callback`: OAuth2 供應商回調端點。
