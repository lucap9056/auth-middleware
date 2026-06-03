# JWT 模組

此模組為 `auth-middleware` 專案提供 JWT (JSON Web Token) 管理功能。它處理存取令牌（Access Token）與重新整理令牌（Refresh Token）的生成、驗證與生命週期管理，並透過資料庫管理裝置專屬的簽署金鑰，以提升安全性。

## 特色

- **裝置綁定安全性**: 簽署金鑰透過資料庫介面管理，確保每個裝置擁有唯一的秘密。
- **令牌生命週期**: 完整支援 Access Token 與 Refresh Token 的產生與驗證。
- **介面導向**: 可輕易整合任何實作 `Database` 介面的後端儲存系統。

## 使用方式

要使用此模組，您必須實作 `Database` 介面。可以在此介面的實作中整合快取機制（例如 Redis）以提升效能。

```go
type Database interface {
	UpdateDeviceSecret(deviceID, secret string) error
	GetDeviceSecret(deviceID string) (string, error)
}
```

### 範例

```go
import "github.com/lucap9056/auth-middleware/jwt"

// 使用您的資料庫實作進行初始化
manager := jwt.NewJWTManager(db)

// 生成 Refresh Token
token, err := manager.GenerateRefresh(userID, deviceID)

// 生成 Access Token
accessToken, err := manager.GenerateAccess(refreshToken, username)

// 驗證 Access Token
claims, err := manager.VerifyAccess(accessToken)
```

## 測試

在模組目錄中執行測試：

```bash
go test -v ./...
```
