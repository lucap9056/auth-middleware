# JWT Module

Issues and verifies access and refresh tokens, with one signing secret per device

## Installation

```bash
go get github.com/lucap9056/corvauth/jwt
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
| `UpdateDeviceSecret(deviceID)` | Increments the generation and returns it; errors if the device does not exist |
| `GetDeviceSecret(deviceID)` | Returns the secret and current generation |

## Usage

```go
import "github.com/lucap9056/corvauth/jwt"

manager := jwt.NewJWTManager(db,
	jwt.WithIssuer("auth-service"),
	jwt.WithAudience("web"),
)
```

### Login

The caller generates the secret, stores it, and passes it to `GenerateRefresh`

New devices start at generation `1`:

```go
secret := randomHex(32)
deviceID, err := db.SaveDeviceSecret(userEmail, deviceName, secret)

refreshToken, err := manager.GenerateRefresh(userEmail, deviceID, secret, 1)
accessToken, err := manager.GenerateAccess(refreshToken, username)
```

### Rotating the refresh token

Claims returned by `RotateRefresh`:

- success: the new refresh token's claims
- verification failure: the given token's claims

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

Both are signed with HS256 using the device secret; the JWS `typ` header marks their purpose (RFC 8725 §3.11)

| Token | `typ` | Claims |
|---|---|---|
| Access token | `access+jwt` | `sub`, `user_email`, `username`, `device_id`, `generation`, `iss`, `aud`, `iat`, `exp` |
| Refresh token | `refresh+jwt` | `sub`, `device_id`, `generation`, `iss`, `aud`, `iat`, `exp` |

`sub` is the user's email

### Verification

`VerifyAccess`, `VerifyRefresh`, and `GenerateAccess` (on the given refresh token) check, in order:

1. `typ` matches the token kind (case-insensitive, optional `application/` prefix)
2. The device exists
3. The signature is valid and not expired
4. `iss` / `aud` match when `WithIssuer` / `WithAudience` are set
5. The token's `generation` equals the device's current generation

### Errors

| Error | When |
|---|---|
| `ErrTokenRevoked` | Only check 5 fails: issued here but since rotated |
| `ErrInvalidToken` | Any other failure |

`ErrTokenRevoked` wraps `ErrInvalidToken`, so `errors.Is(err, jwt.ErrInvalidToken)` holds for both

Check `ErrTokenRevoked` first to tell them apart

## Options

| Option | Default | Description |
|---|---|---|
| `WithAccessTokenDuration(d)` | `15m` | Access token lifetime |
| `WithRefreshTokenDuration(d)` | `7d` | Refresh token lifetime |
| `WithIssuer(iss)` | empty | Written to `iss`; must match when non-empty |
| `WithAudience(aud)` | empty | Written to `aud`; must be contained when non-empty |

## Testing

```bash
go test ./...
```
