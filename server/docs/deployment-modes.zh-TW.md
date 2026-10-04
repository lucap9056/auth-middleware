# 部署模式

模式由 `DATABASE_URL` 與 `DB_USER_EMAIL_REFERENCE` 決定，比較表見 [README](../README.zh-TW.md#部署模式)。

## Stateless proxy

輕量的 OAuth2 client：client secret、`state` 與 PKCE verifier 都留在 server 端，最後把 provider 的 token 交給 client。不需要 database；`DB_*`、`JWT_*`、`ALLOW_REGISTRATION`、`ALLOW_UNVERIFIED_EMAIL` 都沒有作用。下游 service 必須自行驗證 provider 的 token。

## Managed users

本 server 管理一個 `users` table：

```sql
CREATE TABLE IF NOT EXISTS users (
    user_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE
);
```

設定 `ALLOW_REGISTRATION=true` 時，第一次登入的使用者會被 insert，`username` 使用 provider 提供的顯示名稱。未開啟時，只有已存在於 `users` 的 email 能登入。`DB_AUTO_CREATE_SCHEMA` 不是 `true` 時，table 必須事先存在，啟動時會檢查。

## External users

Users table 由你的應用程式管理，透過 `DB_USER_EMAIL_REFERENCE` 告訴本 server 它的位置，例如 `public.accounts(email):citext`。Email column 必須是 `PRIMARY KEY` 或有 `UNIQUE` constraint。本 server 不會寫入這個 table，只有已存在的 email 能登入。在你的應用程式中刪除 user 時，對應的 session 會透過 foreign key 一併刪除。

若要在 `username` claim 放入顯示名稱，將 `DB_USER_USERNAME_COLUMN` 設為同一個 table 中的 column。需要該 table 的 `SELECT` 權限，啟動時會檢查，`NULL` 會視為空字串。
