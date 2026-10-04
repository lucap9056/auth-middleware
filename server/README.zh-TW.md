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

## 文件

- [部署模式](docs/deployment-modes.zh-TW.md)：各模式的 users table 設定
- [Session](docs/sessions.zh-TW.md)：登入流程、token、rotation、provider token
- [API](docs/api.zh-TW.md)：endpoint 與 error header
- [設定](docs/configuration.zh-TW.md)：環境變數

## 測試

```bash
go test ./internal/... ./tests/
```

Unit 與 integration test 不需要外部服務。其他測試需要對應的環境：

| 測試 | 需求 | 指令 |
|---|---|---|
| `internal/cache/device` 與 `internal/flight` 的 Redis test | `TEST_REDIS_URL` | `go test ./internal/...`（未設定時 skip） |
| End-to-end 流程 | Compose 環境已啟動 | `go test ./tests/e2e/` |
