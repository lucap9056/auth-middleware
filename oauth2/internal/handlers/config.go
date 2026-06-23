package handlers

import "github.com/lucap9056/auth-middleware/database"

type DB interface {
	GetUserFromEmail(email string) (*database.User, error)
	GetUserFromID(userID string) (*database.User, error)
	CreateUser(username, email string) (*database.User, error)
	SaveDeviceSecret(userID, deviceName, secret string) (string, error)
	DeleteDevice(userID, deviceID string) error
	DeleteAllDevices(userID string) error
	DeleteUser(userID string) error
}

type AuthConfig struct {
	DevMode           bool
	AllowRegistration bool
	PassOAuthToken    bool
}

type AuthOption func(*AuthConfig)

func WithDevMode(enabled bool) AuthOption {
	return func(c *AuthConfig) {
		c.DevMode = enabled
	}
}

func WithAllowRegistration(enabled bool) AuthOption {
	return func(c *AuthConfig) {
		c.AllowRegistration = enabled
	}
}

func WithPassOAuthToken(enabled bool) AuthOption {
	return func(c *AuthConfig) {
		c.PassOAuthToken = enabled
	}
}
