package device

import (
	"time"

	"github.com/redis/go-redis/v9"
)

const secretTTL = 5 * time.Minute

type SecretCache interface {
	GetSecret(deviceID string) (string, bool)
	SetSecret(deviceID, secret string)
	DeleteSecret(deviceID string)
	AddUserDevice(userID, deviceID string)
	RemoveUserDevice(userID, deviceID string)
	PopAllUserDevices(userID string) []string
}

func NewSecretCache(client *redis.Client) SecretCache {
	if client == nil {
		return newMemorySecretCache()
	}
	return newRedisSecretCache(client)
}
