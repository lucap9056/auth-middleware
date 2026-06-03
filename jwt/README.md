# JWT Module

This module provides JWT (JSON Web Token) management for the `auth-middleware` project. It handles the generation, validation, and lifecycle management of access and refresh tokens, with a focus on security by utilizing a database to manage device-specific signing secrets.

## Features

- **Device-Bound Security**: Signing secrets are managed via a database interface, allowing each device to have a unique secret.
- **Token Lifecycle**: Full support for Access and Refresh token generation and verification.
- **Interface-Driven**: Easily integrable with any storage backend that implements the `Database` interface.

## Usage

To use this module, you must implement the `Database` interface. You can incorporate caching (e.g., Redis) within your implementation of this interface to improve performance.

```go
type Database interface {
	UpdateDeviceSecret(deviceID, secret string) error
	GetDeviceSecret(deviceID string) (string, error)
}
```

### Example

```go
import "github.com/lucap9056/auth-middleware/jwt"

// Initialize with your database implementation
manager := jwt.NewJWTManager(db)

// Generate Refresh Token
token, err := manager.GenerateRefresh(userID, deviceID)

// Generate Access Token
accessToken, err := manager.GenerateAccess(refreshToken, username)

// Verify Access Token
claims, err := manager.VerifyAccess(accessToken)
```

## Testing

Run the tests within the module directory:

```bash
go test -v ./...
```
