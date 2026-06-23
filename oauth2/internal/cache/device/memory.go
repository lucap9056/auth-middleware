package device

import (
	"sync"
	"time"
)

const secretShardCount = 16

type secretEntry struct {
	value     string
	expiresAt int64
}

type secretShard struct {
	mu   sync.RWMutex
	data map[string]secretEntry
}

type memorySecretCache struct {
	shards      []*secretShard
	userDevices sync.Map
	ttl         time.Duration
}

func newMemorySecretCache() *memorySecretCache {
	shards := make([]*secretShard, secretShardCount)
	for i := range shards {
		shards[i] = &secretShard{data: make(map[string]secretEntry)}
	}
	return &memorySecretCache{shards: shards, ttl: secretTTL}
}

func (m *memorySecretCache) getShard(key string) *secretShard {
	var hash uint32 = 2166136261
	for i := 0; i < len(key); i++ {
		hash *= 16777619
		hash ^= uint32(key[i])
	}
	return m.shards[hash%secretShardCount]
}

func (m *memorySecretCache) GetSecret(deviceID string) (string, bool) {
	shard := m.getShard(deviceID)
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	entry, ok := shard.data[deviceID]
	if !ok || time.Now().UnixNano() > entry.expiresAt {
		return "", false
	}
	return entry.value, true
}

func (m *memorySecretCache) SetSecret(deviceID, secret string) {
	shard := m.getShard(deviceID)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	shard.data[deviceID] = secretEntry{
		value:     secret,
		expiresAt: time.Now().Add(m.ttl).UnixNano(),
	}
}

func (m *memorySecretCache) DeleteSecret(deviceID string) {
	shard := m.getShard(deviceID)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	delete(shard.data, deviceID)
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
