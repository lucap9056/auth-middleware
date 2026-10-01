# OAuth2 模組

此模組提供了一個認證與授權伺服器，實作了 OAuth2 流程並管理基於 JWT 的會話。

## 內建服務供應商

此模組內建支援以下 OAuth2 服務：

- **Discord**
- **Google**
- **GitHub**
- **Generic OIDC** (任何 OpenID Connect 供應商，透過 discovery 自動設定端點 — 詳見 `OIDC_ISSUER_URL`)
- **Generic** (透過自訂端點設定支援任何標準 OAuth2 服務)

## 環境變數 (ENV)

| 變數 | 說明 |
| :--- | :--- |
| `DATABASE_URL` | PostgreSQL 連線字串 |
| `HTTP_ADDRESS` | 伺服器位址 (預設: `:80`) |
| `OAUTH2_PROVIDER` | 服務供應商名稱 (`discord`, `github`, `google`, 或其他即為 `generic`) |
| `OAUTH2_CLIENT_ID` | OAuth2 Client ID |
| `OAUTH2_CLIENT_SECRET` | OAuth2 Client Secret |
| `OAUTH2_REDIRECT_URL` | OAuth2 流程的重新導向網址 |
| `OAUTH2_SCOPES` | 以逗號分隔的 OAuth2 權限範圍 (Scopes) |
| `OAUTH2_AUTH_URL` | `generic` 供應商必要設定 |
| `OAUTH2_TOKEN_URL` | `generic` 供應商必要設定 |
| `OAUTH2_USERINFO_URL` | `generic` 供應商必要設定 |
| `OAUTH2_REVOKE_URL` | 選填 |
| `OIDC_ISSUER_URL` | 啟用 OIDC discovery 模式：自動從 `{issuer}/.well-known/openid-configuration` 取得 auth/token/userinfo/revocation 端點。需搭配 `OAUTH2_CLIENT_ID`、`OAUTH2_CLIENT_SECRET`、`OAUTH2_REDIRECT_URL`；設定後優先於 `OAUTH2_PROVIDER`。 |
| `OAUTH2_CLIENT_PKCE` | 設定為 `true` 讓用戶端自行持有 PKCE verifier（由 `/login` 回傳，呼叫 `/callback` 時須透過 `X-PKCE-Verifier` header 帶回）。預設為 `false`，即由伺服器內部管理 verifier — 詳見下方「登入流程」。 |
| `REDIS_URL` | Redis 連線字串 (用於 OAuth `state` 儲存、device secret 快取，以及 `/callback` 與 `/refresh` 的跨實例去重；未設定時退回程序內實作，不適合多實例部署) |
| `HTTP_MODE` | 設定為 `development` 以啟用開發功能 |
| `ALLOW_REGISTRATION` | 設定為 `true` 以允許使用者註冊 |
| `PASS_OAUTH_TOKEN` | 設定為 `true` 以將 OAuth 供應商權杖傳遞給用戶端 (詳見操作情境)。 |

## 登入流程 (State 與 PKCE)

`GET /login`會啟動流程並回傳 `{ "url": "..." }`。伺服器會自行產生 PKCE verifier，並以一組 `state` 值為 key 儲存它（若設定 `REDIS_URL` 則存進 Redis，否則存進程序內快取）。回傳的授權 `url` 中會帶有這個 `state`。

`GET /callback` 由供應商以 `code` 與 `state` 兩個 query 參數呼叫——預設情況下用戶端不需要自行處理任何 PKCE 邏輯。伺服器會用 `state` 查出對應的 verifier、完成 code 交換，並刪除該 `state`（重複使用同一個 `state` 會被拒絕）。

若你需要讓用戶端自行持有 verifier（例如完全由用戶端驅動的 PKCE 流程），可設定 `OAUTH2_CLIENT_PKCE=true`：此時 `/login` 的回應會多一個 `verifier` 欄位，用戶端呼叫 `/callback` 時必須透過 `X-PKCE-Verifier` header 將其帶回。

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

## 端對端測試 (E2E)

`docker-compose.yml` 會啟動 Postgres、Redis、本地假 OIDC 供應商 (`test-provider/`)，並透過 `OIDC_ISSUER_URL` 串接本模組：

```
docker compose up --build -d
go test -v ./tests/e2e/
```
