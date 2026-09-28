package token

import (
	"context"
	"time"

	"github.com/lucap9056/auth-middleware/oauth2/internal/cache"
	"github.com/maypok86/otter/v2"
)

type memoryCache struct {
	pairs *otter.Cache[string, TokenPair]
}

func newMemoryCache(maximumSize int, ttl time.Duration) (*memoryCache, error) {
	pairs, err := cache.NewMemoryCache[string](maximumSize, func(TokenPair) time.Duration {
		return ttl
	})
	if err != nil {
		return nil, err
	}
	return &memoryCache{pairs: pairs}, nil
}

func (m *memoryCache) Get(_ context.Context, key string) (*TokenPair, error) {
	pair, ok := m.pairs.GetIfPresent(key)
	if !ok {
		return nil, nil
	}
	return &pair, nil
}

func (m *memoryCache) Set(_ context.Context, key string, value TokenPair) error {
	m.pairs.Set(key, value)
	return nil
}
