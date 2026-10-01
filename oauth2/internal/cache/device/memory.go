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
	secrets, err := cache.NewMemoryCache[string](maximumSize, func(secret string) time.Duration {
		if secret == "" {
			return ttl + deletedSecretExtraTTL
		}
		return ttl
	})
	if err != nil {
		return nil, err
	}
	return &memorySecretCache{secrets: secrets}, nil
}

func (m *memorySecretCache) GetSecret(deviceID string) (string, bool) {
	secret, found := m.secrets.GetIfPresent(deviceID)
	if !found || secret == "" {
		return "", false
	}
	return secret, true
}

func (m *memorySecretCache) SetSecret(deviceID, secret string, overwrite bool) {
	if overwrite {
		m.secrets.Set(deviceID, secret)
		return
	}
	m.secrets.SetIfAbsent(deviceID, secret)
}

func (m *memorySecretCache) DeleteSecret(deviceID string) {
	m.secrets.Set(deviceID, "")
}

func (m *memorySecretCache) Close() error {
	m.secrets.StopAllGoroutines()
	return nil
}
