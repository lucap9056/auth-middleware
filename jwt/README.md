# JWT Module

This module issues and verifies access and refresh tokens. Each device has its own signing secret.

## Installation

```bash
go get github.com/lucap9056/auth-middleware/jwt/v2
```

## Database

`JWTManager` reads and updates a device's secret and generation through the `Database` interface:

```go
type Database interface {
	UpdateDeviceSecret(deviceID string) (int, error)
	GetDeviceSecret(deviceID string) (string, int, error)
}
```

| Method | Behavior |
|---|---|
| `UpdateDeviceSecret(deviceID)` | Increments the device's generation and returns the new value; returns an error if the device does not exist |
| `GetDeviceSecret(deviceID)` | Returns the device's secret and current generation |

## Usage

```go
import "github.com/lucap9056/auth-middleware/jwt/v2"

manager := jwt.NewJWTManager(db,
	jwt.WithIssuer("auth-service"),
	jwt.WithAudience("web"),
)
```

### Login

The caller generates the secret, stores it in the database, and passes it to `GenerateRefresh` for signing. A newly created device starts at generation `1`:

```go
secret := randomHex(32)
deviceID, err := db.SaveDeviceSecret(userEmail, deviceName, secret)

refreshToken, err := manager.GenerateRefresh(userEmail, deviceID, secret, 1)
accessToken, err := manager.GenerateAccess(refreshToken, username)
```

### Rotating the refresh token

The claims returned by `RotateRefresh` are those of the new refresh token on success, and those of the given token on verification failure:

```go
newRefreshToken, claims, err := manager.RotateRefresh(refreshToken)

username := lookupUsername(claims.Subject)
accessToken, err := manager.GenerateAccess(newRefreshToken, username)
```

### Verifying an access token

```go
claims, err := manager.VerifyAccess(accessToken)
```

## Tokens

Both tokens are signed with HS256 using the device's secret, and their purpose is marked by the JWS `typ` header (RFC 8725 §3.11):

| Token | `typ` | Claims |
|---|---|---|
| Access token | `access+jwt` | `sub`, `user_email`, `username`, `device_id`, `generation`, `iss`, `aud`, `iat`, `exp` |
| Refresh token | `refresh+jwt` | `sub`, `device_id`, `generation`, `iss`, `aud`, `iat`, `exp` |

`sub` is the user's email.

### Verification

`VerifyAccess`, `VerifyRefresh`, and `GenerateAccess` (which verifies the given refresh token) check, in order:

1. `typ` matches the token kind, case-insensitively, with an optional `application/` prefix
2. The device exists
3. The signature is valid and the token has not expired
4. `iss` / `aud` match when `WithIssuer` / `WithAudience` are set
5. The token's `generation` equals the device's current generation

### Errors

| Error | When |
|---|---|
| `ErrTokenRevoked` | Only check 5 fails: the token was issued by this service but has since been rotated |
| `ErrInvalidToken` | Any other verification failure |

`ErrTokenRevoked` wraps `ErrInvalidToken`, so `errors.Is(err, jwt.ErrInvalidToken)` holds for both. Check `ErrTokenRevoked` first when you need to tell them apart.

## Options

| Option | Default | Description |
|---|---|---|
| `WithAccessTokenDuration(d)` | `15m` | Access token lifetime |
| `WithRefreshTokenDuration(d)` | `7d` | Refresh token lifetime |
| `WithIssuer(iss)` | empty | Written to `iss`; must match on verification when non-empty |
| `WithAudience(aud)` | empty | Written to `aud`; must be contained on verification when non-empty |

## Testing

```bash
go test ./...
```
