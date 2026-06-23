package token

import (
	"context"
	"sync"
	"time"
)

const shardCount = 32

type memoryCache struct {
	shards []*cacheShard
	stopCh chan struct{}
	ttl    time.Duration
}

type cacheShard struct {
	mu   sync.RWMutex
	data map[string]item
}

type item struct {
	value      *TokenPair
	expiration int64
}

func newMemoryCache(cleanupInterval time.Duration, ttl time.Duration) *memoryCache {
	mc := &memoryCache{
		shards: make([]*cacheShard, shardCount),
		stopCh: make(chan struct{}),
		ttl:    ttl,
	}
	for i := 0; i < shardCount; i++ {
		mc.shards[i] = &cacheShard{data: make(map[string]item)}
	}
	go mc.evictExpiredLoop(cleanupInterval)
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

func (m *memoryCache) Get(_ context.Context, key string) (*TokenPair, error) {
	shard := m.getShard(key)
	shard.mu.RLock()
	defer shard.mu.RUnlock()
	it, ok := shard.data[key]
	if !ok || (it.expiration > 0 && time.Now().UnixNano() > it.expiration) {
		return nil, nil
	}
	return it.value, nil
}

func (m *memoryCache) Set(_ context.Context, key string, value TokenPair) error {
	shard := m.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()
	shard.data[key] = item{value: &value, expiration: time.Now().Add(m.ttl).UnixNano()}
	return nil
}

func (m *memoryCache) evictExpiredLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
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
		case <-m.stopCh:
			return
		}
	}
}
