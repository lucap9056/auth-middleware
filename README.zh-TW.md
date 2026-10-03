# Auth Middleware

給 Go service 使用的 device-bound session 認證方案，由三個 module 組成，可以搭配使用，也可以單獨使用。

## Modules

| Module | Import path | 用途 |
|---|---|---|
| [`oauth2`](./oauth2/README.zh-TW.md) | service（`oauth2/cmd`） | 認證 server：OAuth2 / OIDC 登入、簽發 session token、提供 reverse proxy 驗證 token |
| [`jwt`](./jwt/README.zh-TW.md) | `github.com/lucap9056/auth-middleware/jwt/v2` | 簽發與驗證 access token、refresh token，每個 device 使用各自的 secret 簽章 |
| [`database`](./database/README.zh-TW.md) | `github.com/lucap9056/auth-middleware/database/v2` | 把 device session 存在 PostgreSQL，透過 email 關聯到你的 users table |

```
              ┌──────────────────────── oauth2 ────────────────────────┐
 browser ───▶ │ /login, /callback ──▶ OAuth2 / OIDC provider           │
 proxy   ───▶ │ /verify, /refresh ──▶ jwt ──▶ database ──▶ PostgreSQL  │
              └────────────────────────────────────────────────────────┘
```

- 每次登入都會建立一個 **device session**，有自己的 signing secret。登出或刪除 device 只會讓該 device 的 token 失效。
- Refresh token 每次使用都會 rotate，被換掉的舊 refresh token 會被拒絕（`ErrTokenRevoked`）。
- 使用者以 **email** 識別。`auth_user_devices` table 透過 foreign key 參照 users table，刪除 user 時對應的 session 也會一併刪除。

## 快速開始

### 執行認證 server

最快看到整套運作的方式是 `oauth2` 的 compose 設定，會一次啟動 PostgreSQL、Redis、假的 OIDC provider 和 server：

```bash
cd oauth2
docker compose up --build -d
```

正式部署時透過環境變數設定 `oauth2`。它可以自己管理 users table，也可以使用你的應用程式既有的 users table，詳見 [oauth2 → 部署模式](./oauth2/README.zh-TW.md#部署模式)。

### 在自己的 service 中使用 library

```bash
go get github.com/lucap9056/auth-middleware/database/v2
go get github.com/lucap9056/auth-middleware/jwt/v2
```

1. 確認 users table 的 email column 是 `PRIMARY KEY` 或有 `UNIQUE` constraint。
2. 用 `database.NewDatabase(dsn, ...)` 開啟 device store；若要共用既有的 connection pool，改用 `database.New(sqlDB, ...)`。加上 `database.WithAutoCreateSchema(true)` 會在啟動時建立 `auth_user_devices` table。
3. 把它傳給 `jwt.NewJWTManager(db, ...)` 來簽發與驗證 token。

Schema 由 `database` module 自己建立，不需要另外執行 SQL 檔。細節請見 [database](./database/README.zh-TW.md) 與 [jwt](./jwt/README.zh-TW.md) 的 README。

## 需求

- Library 需要 Go 1.24+，`oauth2` 需要 Go 1.25+
- PostgreSQL 14+
- Redis（選用；`oauth2` 多實例部署時必須）
