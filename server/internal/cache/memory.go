package cache

import (
	"errors"
	"time"

	"github.com/maypok86/otter/v2"
)

var (
	ErrInvalidMaximumSize = errors.New("memory cache maximum size must be positive")
	ErrMissingTTLFunc     = errors.New("memory cache ttl func is required")
)

func NewMemoryCache[K comparable, V any](maximumSize int, ttl func(value V) time.Duration) (*otter.Cache[K, V], error) {
	if maximumSize <= 0 {
		return nil, ErrInvalidMaximumSize
	}
	if ttl == nil {
		return nil, ErrMissingTTLFunc
	}

	return otter.New(&otter.Options[K, V]{
		MaximumSize: maximumSize,
		ExpiryCalculator: otter.ExpiryWritingFunc(func(e otter.Entry[K, V]) time.Duration {
			return ttl(e.Value)
		}),
	})
}
