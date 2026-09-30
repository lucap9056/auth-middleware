package device

import (
	"time"

	"github.com/redis/rueidis"
)

const (
	secretTTL           = 5 * time.Minute
	secretMemoryMaxSize = 100_000
)

type SecretCache interface {
	GetSecret(deviceID string) (string, bool)
	SetSecret(deviceID, secret string)
	DeleteSecret(deviceID string)
	AddUserDevice(userID, deviceID string)
	RemoveUserDevice(userID, deviceID string)
	PopAllUserDevices(userID string) []string
	Close() error
}

func NewSecretCache(client rueidis.Client) (SecretCache, error) {
	if client == nil {
		return newMemorySecretCache(secretMemoryMaxSize, secretTTL)
	}
	return newRedisSecretCache(client), nil
}
