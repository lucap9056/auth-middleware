package token

import (
	"context"
	"time"

	"github.com/redis/rueidis"
)

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type Cache interface {
	Get(context.Context, string) (*TokenPair, error)
	Set(context.Context, string, TokenPair) error
}

type config struct {
	TTL         time.Duration
	MaximumSize int
}

type Option func(*config)

func WithTTL(ttl time.Duration) Option {
	return func(c *config) {
		c.TTL = ttl
	}
}

func WithMaximumSize(size int) Option {
	return func(c *config) {
		c.MaximumSize = size
	}
}

func NewCache(client rueidis.Client, opts ...Option) (Cache, error) {
	cfg := &config{TTL: 30 * time.Second, MaximumSize: 100_000}
	for _, opt := range opts {
		opt(cfg)
	}

	if client == nil {
		return newMemoryCache(cfg.MaximumSize, cfg.TTL)
	}
	return newRedisCache(client, cfg.TTL), nil
}
