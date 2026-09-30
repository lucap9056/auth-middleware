package device

import (
	"time"

	"github.com/lucap9056/auth-middleware/oauth2/internal/cache"
	"github.com/maypok86/otter/v2"
)

type memorySecretCache struct {
	secrets *otter.Cache[string, string]
}

func newMemorySecretCache(maximumSize int, ttl time.Duration) (*memorySecretCache, error) {
	secrets, err := cache.NewMemoryCache[string](maximumSize, func(string) time.Duration {
		return ttl
	})
	if err != nil {
		return nil, err
	}
	return &memorySecretCache{secrets: secrets}, nil
}

func (m *memorySecretCache) GetSecret(deviceID string) (string, bool) {
	return m.secrets.GetIfPresent(deviceID)
}

func (m *memorySecretCache) SetSecret(deviceID, secret string) {
	m.secrets.Set(deviceID, secret)
}

func (m *memorySecretCache) DeleteSecret(deviceID string) {
	m.secrets.Invalidate(deviceID)
}

func (m *memorySecretCache) Close() error {
	m.secrets.StopAllGoroutines()
	return nil
}
