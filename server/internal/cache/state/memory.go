package state

import (
	"context"
	"time"

	"github.com/lucap9056/corvauth/server/internal/cache"
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

func (m *memoryCache) Take(_ context.Context, key string) (string, error) {
	var verifier string
	m.verifiers.Compute(key, func(oldValue string, found bool) (string, otter.ComputeOp) {
		verifier = oldValue
		return "", otter.InvalidateOp
	})
	return verifier, nil
}

func (m *memoryCache) Set(_ context.Context, key, verifier string) error {
	m.verifiers.Set(key, verifier)
	return nil
}
