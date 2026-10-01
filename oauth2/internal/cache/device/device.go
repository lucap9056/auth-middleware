package device

import (
	"time"

	"github.com/lucap9056/auth-middleware/oauth2/internal/cache"
)

const (
	secretTTL             = 5 * time.Minute
	secretMemoryMaxSize   = 100_000
	deletedSecretExtraTTL = time.Minute
)

type SecretCache interface {
	GetSecret(deviceID string) (string, bool)
	SetSecret(deviceID, secret string, overwrite bool)
	DeleteSecret(deviceID string)
	Close() error
}

func NewSecretCache(client *cache.RedisClient) (SecretCache, error) {
	if client == nil {
		return newMemorySecretCache(secretMemoryMaxSize, secretTTL)
	}
	return newRedisSecretCache(client), nil
}
