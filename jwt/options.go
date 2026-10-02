package jwt

import (
	"time"
)

type options struct {
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
}

type Option func(*options)

func defaultOptions() *options {
	return &options{
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 7 * 24 * time.Hour,
	}
}

func WithAccessTokenDuration(d time.Duration) Option {
	return func(o *options) {
		o.AccessTokenDuration = d
	}
}

func WithRefreshTokenDuration(d time.Duration) Option {
	return func(o *options) {
		o.RefreshTokenDuration = d
	}
}
