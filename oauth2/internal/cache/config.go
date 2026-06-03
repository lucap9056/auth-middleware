package cache

import "time"

type CacheConfig struct {
	TTL time.Duration
}

type CacheOption func(*CacheConfig)

func WithTTL(ttl time.Duration) CacheOption {
	return func(c *CacheConfig) {
		c.TTL = ttl
	}
}
