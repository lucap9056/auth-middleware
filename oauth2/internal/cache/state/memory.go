package state

import (
	"context"
	"sync"
	"time"
)

const shardCount = 32

type memoryCache struct {
	shards []*cacheShard
	ttl    time.Duration
}

type cacheShard struct {
	mu   sync.RWMutex
	data map[string]stateItem
}

type stateItem struct {
	verifier   string
	expiration int64
}

func newMemoryCache(ttl time.Duration) *memoryCache {
	mc := &memoryCache{
		shards: make([]*cacheShard, shardCount),
		ttl:    ttl,
	}
	for i := range shardCount {
		mc.shards[i] = &cacheShard{data: make(map[string]stateItem)}
	}
	go mc.evictExpiredLoop(ttl)
	return mc
}

func (m *memoryCache) getShard(key string) *cacheShard {
	var hash uint32 = 2166136261
	for i := 0; i < len(key); i++ {
		hash *= 16777619
		hash ^= uint32(key[i])
	}
	return m.shards[hash%shardCount]
}

func (m *memoryCache) Get(_ context.Context, key string) (string, error) {
	shard := m.getShard(key)
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	it, ok := shard.data[key]
	if !ok || (it.expiration > 0 && time.Now().UnixNano() > it.expiration) {
		return "", nil
	}
	return it.verifier, nil
}

func (m *memoryCache) Set(_ context.Context, key, verifier string) error {
	shard := m.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	shard.data[key] = stateItem{verifier: verifier, expiration: time.Now().Add(m.ttl).UnixNano()}
	return nil
}

func (m *memoryCache) Delete(_ context.Context, key string) error {
	shard := m.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	delete(shard.data, key)
	return nil
}

func (m *memoryCache) evictExpiredLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now().UnixNano()
		for _, shard := range m.shards {
			shard.mu.Lock()
			for k, v := range shard.data {
				if v.expiration > 0 && now > v.expiration {
					delete(shard.data, k)
				}
			}
			shard.mu.Unlock()
		}
	}
}
