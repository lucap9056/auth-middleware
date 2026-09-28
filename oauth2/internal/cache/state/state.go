package state

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type Cache interface {
	Set(ctx context.Context, state, verifier string) error
	Get(ctx context.Context, state string) (string, error)
	Delete(ctx context.Context, state string) error
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

func NewCache(client *redis.Client, opts ...Option) (Cache, error) {
	cfg := &config{TTL: 10 * time.Minute, MaximumSize: 100_000}
	for _, opt := range opts {
		opt(cfg)
	}

	if client == nil {
		return newMemoryCache(cfg.MaximumSize, cfg.TTL)
	}
	return newRedisCache(client, cfg.TTL), nil
}
