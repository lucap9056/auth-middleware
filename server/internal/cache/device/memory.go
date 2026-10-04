package device

import (
	"time"

	"github.com/lucap9056/corvauth/server/internal/cache"
	"github.com/maypok86/otter/v2"
)

type memorySecretCache struct {
	secrets *otter.Cache[string, Secret]
}

func newMemorySecretCache(maximumSize int, ttl time.Duration) (*memorySecretCache, error) {
	secrets, err := cache.NewMemoryCache[string](maximumSize, func(secret Secret) time.Duration {
		if secret.deleted() {
			return ttl + deletedSecretExtraTTL
		}
		return ttl
	})
	if err != nil {
		return nil, err
	}
	return &memorySecretCache{secrets: secrets}, nil
}

func (m *memorySecretCache) GetSecret(deviceID string) (Secret, bool) {
	secret, found := m.secrets.GetIfPresent(deviceID)
	if !found || secret.deleted() {
		return Secret{}, false
	}
	return secret, true
}

func (m *memorySecretCache) SetSecret(deviceID string, secret Secret) {
	m.secrets.Compute(deviceID, func(current Secret, found bool) (Secret, otter.ComputeOp) {
		if found && (current.deleted() || current.Generation >= secret.Generation) {
			return current, otter.CancelOp
		}
		return secret, otter.WriteOp
	})
}

func (m *memorySecretCache) DeleteSecret(deviceID string) {
	m.secrets.Set(deviceID, Secret{})
}

func (m *memorySecretCache) Close() error {
	m.secrets.StopAllGoroutines()
	return nil
}
