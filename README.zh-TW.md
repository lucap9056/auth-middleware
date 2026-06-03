# Auth Middleware

一套基於 Go 語言的模組化認證與授權中介軟體系統，旨在提供高度的靈活性與安全性。

## 架構

本專案結構為一組獨立的 Go 模組元件：

- **[`/database`](./database/README.zh-TW.md)**: 使用 PostgreSQL 的資料持久化層。提供資料庫結構初始化腳本，並定義了裝置金鑰管理的標準介面。
- **[`/jwt`](./jwt/README.zh-TW.md)**: 核心 JWT 管理模組。負責權杖生成、驗證，並利用 `Database` 介面實現安全的裝置綁定權杖生命週期。
- **[`/oauth2`](./oauth2/README.zh-TW.md)**: 認證伺服器模組，實作 OAuth2 流程，並整合 `database` 與 `jwt` 模組以提供穩健的會話管理。

## 快速開始

1.  **資料庫設定**: 使用 `database/init.sql` 初始化您的 PostgreSQL 資料庫。
2.  **配置**: 根據各個模組的 `README.zh-TW.md` 進行配置。
3.  **部署**: 將 `oauth2` 模組部署為您的認證服務，或是直接將 `jwt` 與 `database` 模組整合進您現有的 Go 應用程式中。
