package device

import (
	"sync"
	"time"

	"github.com/lucap9056/auth-middleware/oauth2/internal/cache"
	"github.com/maypok86/otter/v2"
)

type memorySecretCache struct {
	secrets     *otter.Cache[string, string]
	userDevices sync.Map
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

func (m *memorySecretCache) AddUserDevice(userID, deviceID string) {
	actual, _ := m.userDevices.LoadOrStore(userID, &sync.Map{})
	actual.(*sync.Map).Store(deviceID, struct{}{})
}

func (m *memorySecretCache) RemoveUserDevice(userID, deviceID string) {
	if devMap, ok := m.userDevices.Load(userID); ok {
		devMap.(*sync.Map).Delete(deviceID)
	}
}

func (m *memorySecretCache) PopAllUserDevices(userID string) []string {
	devMap, ok := m.userDevices.LoadAndDelete(userID)
	if !ok {
		return nil
	}
	var ids []string
	devMap.(*sync.Map).Range(func(k, _ any) bool {
		ids = append(ids, k.(string))
		return true
	})
	return ids
}
