# OAuth2.0 Module

對接 OAuth2.0 / OIDC provider：產生登入 URL、交換 code、取得 userinfo、revoke token

## Packages

| Package | 用途 |
|---|---|
| `oauthclient` | 登入流程：`AuthURL`、`Exchange`、`GetUser`、`Revoke`，支援 PKCE |
| `providers` | 各 provider 的 userinfo 與 revoke 實作 |

## Providers

| 名稱 | 建立方式 |
|---|---|
| `discord`、`github`、`google` | `oauthclient.New`，endpoint 內建 |
| 其他名稱（generic） | `oauthclient.New`，需提供 `AuthURL`、`TokenURL`、`UserinfoURL` |
| OIDC | `oauthclient.NewOIDC`，從 `{issuer}/.well-known/openid-configuration` 取得 endpoint |

## Email 驗證

- Provider 未回傳 email：`ErrMissingEmail`
- Email 未驗證：`ErrUnverifiedEmail`，可用 `providers.WithAllowUnverifiedEmail(true)` 放寬

## 測試

```bash
go test ./...
```
