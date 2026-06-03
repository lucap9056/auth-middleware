package handlers

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
