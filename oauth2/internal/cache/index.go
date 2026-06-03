package cache

import (
	"context"
	"time"
)

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type Cache interface {
	Get(context.Context, string) (*TokenPair, error)
	Set(context.Context, string, TokenPair) error
}

func NewCache(url string, opts ...CacheOption) (Cache, error) {
	config := &CacheConfig{
		TTL: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(config)
	}

	if url == "" {
		cacahe := NewMemoryCache(config.TTL, config.TTL)
		return cacahe, nil
	}
	return NewRedisCache(url, config.TTL)
}
