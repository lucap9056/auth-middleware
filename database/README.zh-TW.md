# Database Module

此 module 負責將 `auth-middleware` 的 device session 存放於 PostgreSQL。使用者帳號由你的應用程式管理，此 module 只透過 email 參考 users table。

## 需求

- **PostgreSQL 13+**
- **既有的 users table**：email column 必須是 `PRIMARY KEY` 或有 `UNIQUE` constraint，才能被 foreign key 參考

## 安裝

```bash
go get github.com/lucap9056/auth-middleware/database/v2
```

## 使用方式

```go
import "github.com/lucap9056/auth-middleware/database/v2"

userRef, err := database.WithUserEmailReference("users(email)")
if err != nil {
	return err
}

db, err := database.NewDatabase(dsn,
	database.WithAutoCreateSchema(true),
	userRef,
)
if err != nil {
	return err
}
defer db.Close()
```

## Options

| Option | 預設值 | 說明 |
|---|---|---|
| `WithMaxOpenConns(n)` | `20` | 最大連線數 |
| `WithMaxIdleConns(n)` | `15` | 最大閒置連線數 |
| `WithConnMaxLifetime(d)` | `5m` | 單一連線最長存活時間 |
| `WithConnMaxIdleTime(d)` | `2m` | 單一連線最長閒置時間 |
| `WithCleanupInterval(d)` | `24h` | 清除 7 天內未更新 device 的間隔；設為 `0` 則停用 |
| `WithAutoCreateSchema(bool)` | `false` | 啟動時自動建立 `auth_user_devices` table 與 index |
| `WithUserEmailReference(ref)` | `users(email)` | foreign key 參考的 users table 與 email column |

### User Email Reference

格式：`[<schema>.]<table>(<column>)[:<type>]`

| 範例 | 意義 |
|---|---|
| `users(email)` | `users.email`，type 為 `text` |
| `users(email):citext` | `users.email`，type 為 `citext` |
| `auth.members(mail):varchar(255)` | `auth.members.mail`，type 為 `varchar(255)` |

不分大小寫，一律轉為小寫處理。`<type>` 必須與被參考的 column type 一致。

### Auto Create Schema

啟用後，`NewDatabase` 會建立 schema，需要 DDL 權限。未啟用時，請透過 `schema.Generate` 或下方 SQL 自行建立。

## Schema

```sql
CREATE TABLE IF NOT EXISTS auth_user_devices (
    device_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_name TEXT NOT NULL,
    user_email TEXT NOT NULL,
    secret TEXT NOT NULL,
    generation INT NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_auth_user_devices FOREIGN KEY (user_email)
        REFERENCES users (email)
        ON DELETE CASCADE ON UPDATE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_auth_user_devices_user_email ON auth_user_devices(user_email);
CREATE INDEX IF NOT EXISTS idx_auth_user_devices_updated_at ON auth_user_devices(updated_at);
```

## API

| Method | 說明 |
|---|---|
| `SaveDeviceSecret(userEmail, deviceName, secret)` | 建立 device 並回傳 `device_id`；email 不存在於 users table 時回傳 `ErrUserNotFound` |
| `GetDeviceSecret(deviceID)` | 回傳 device 的 `secret` 與目前的 `generation` |
| `UpdateDeviceSecret(deviceID)` | 將 device 的 `generation` 加 1、更新 `updated_at`，並回傳新的 `generation` |
| `DeleteDevice(userEmail, deviceID)` | 刪除該使用者的單一 device |
| `DeleteAllDevices(userEmail)` | 刪除該使用者的所有 device |
| `DeleteAllDevicesReturningIDs(userEmail)` | 刪除該使用者的所有 device 並回傳其 ID |
| `Close()` | 停止 cleanup worker 並關閉 connection pool |

