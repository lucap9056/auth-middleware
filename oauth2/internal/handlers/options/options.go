package options

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

type Options struct {
	DevMode           bool
	AllowRegistration bool
	PassOAuthToken    bool
	ClientPKCE        bool
}

type Option func(*Options)

func New(opts ...Option) *Options {
	o := &Options{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

func WithDevMode(enabled bool) Option {
	return func(c *Options) {
		c.DevMode = enabled
	}
}

func WithAllowRegistration(enabled bool) Option {
	return func(c *Options) {
		c.AllowRegistration = enabled
	}
}

func WithPassOAuthToken(enabled bool) Option {
	return func(c *Options) {
		c.PassOAuthToken = enabled
	}
}

func WithClientPKCE(enabled bool) Option {
	return func(c *Options) { c.ClientPKCE = enabled }
}
