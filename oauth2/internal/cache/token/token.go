package token

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
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
	TTL time.Duration
}

type Option func(*config)

func WithTTL(ttl time.Duration) Option {
	return func(c *config) {
		c.TTL = ttl
	}
}

func NewCache(client *redis.Client, opts ...Option) Cache {
	cfg := &config{TTL: 30 * time.Second}
	for _, opt := range opts {
		opt(cfg)
	}

	if client == nil {
		return newMemoryCache(cfg.TTL, cfg.TTL)
	}
	return newRedisCache(client, cfg.TTL)
}
