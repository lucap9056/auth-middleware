package state

import (
	"context"
	"time"

	"github.com/lucap9056/auth-middleware/oauth2/internal/cache"
	"github.com/maypok86/otter/v2"
)

type memoryCache struct {
	verifiers *otter.Cache[string, string]
}

func newMemoryCache(maximumSize int, ttl time.Duration) (*memoryCache, error) {
	verifiers, err := cache.NewMemoryCache[string](maximumSize, func(string) time.Duration {
		return ttl
	})
	if err != nil {
		return nil, err
	}
	return &memoryCache{verifiers: verifiers}, nil
}

func (m *memoryCache) Get(_ context.Context, key string) (string, error) {
	verifier, _ := m.verifiers.GetIfPresent(key)
	return verifier, nil
}

func (m *memoryCache) Set(_ context.Context, key, verifier string) error {
	m.verifiers.Set(key, verifier)
	return nil
}

func (m *memoryCache) Delete(_ context.Context, key string) error {
	m.verifiers.Invalidate(key)
	return nil
}
