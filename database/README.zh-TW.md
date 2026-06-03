# 資料庫模組

此模組負責 `auth-middleware` 專案的資料持久化，底層使用 PostgreSQL。

## 先決條件

- **PostgreSQL**: 一個運作中的 PostgreSQL 實例。
- **UUID 擴充功能**: 資料庫必須支援並啟用 `uuid-ossp` 擴充功能。

## 初始化

必須執行 `init.sql` 腳本來建立必要的資料表與擴充功能：

```bash
# 使用 psql 的執行範例
psql -h <host> -U <username> -d <database_name> -f init.sql
```

## 資料表結構說明

此模組依賴兩張資料表：

1.  **`users`**: 儲存使用者資訊。
2.  **`user_devices`**: 儲存裝置資訊，包含用於 JWT 簽署的 `secret`。

> **重要提示**: 請確保您的資料庫使用者擁有足夠的權限來執行 `CREATE EXTENSION IF NOT EXISTS "uuid-ossp";` 以及對這些資料表進行 CRUD 操作。

## 實作要求

此模組設計為注入至其他模組（如 `jwt`）使用。請確保您的資料存取層實作妥善處理以下事項：

- **連線池 (Connection Pooling)**: 使用如 `pgx` 等函式庫並配合連線池以獲得更好的效能。
- **錯誤處理**: 正確將資料庫錯誤對應至應用程式層級的錯誤，特別是當記錄不存在時（例如 `GetDeviceSecret` 找不到記錄時）。
