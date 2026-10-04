package device

import (
	"time"

	"github.com/lucap9056/corvauth/server/internal/cache"
)

const (
	secretTTL             = 5 * time.Minute
	secretMemoryMaxSize   = 100_000
	deletedSecretExtraTTL = time.Minute
)

type Secret struct {
	Value      string
	Generation int
}

func (s Secret) deleted() bool {
	return s.Value == ""
}

type SecretCache interface {
	GetSecret(deviceID string) (Secret, bool)
	SetSecret(deviceID string, secret Secret)
	DeleteSecret(deviceID string)
	Close() error
}

func NewSecretCache(client *cache.RedisClient) (SecretCache, error) {
	if client == nil {
		return newMemorySecretCache(secretMemoryMaxSize, secretTTL)
	}
	return newRedisSecretCache(client), nil
}
