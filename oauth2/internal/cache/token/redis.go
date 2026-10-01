package token

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lucap9056/auth-middleware/oauth2/internal/cache"
	"github.com/redis/rueidis"
)

type redisCache struct {
	client *cache.RedisClient
	ttl    time.Duration
}

func newRedisCache(client *cache.RedisClient, ttl time.Duration) *redisCache {
	return &redisCache{client: client, ttl: ttl}
}

func (r *redisCache) Get(ctx context.Context, key string) (*TokenPair, error) {
	val, err := r.client.Get(ctx, key).AsBytes()
	if rueidis.IsRedisNil(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis get failed: %w", err)
	}

	var pair TokenPair
	if err := json.Unmarshal(val, &pair); err != nil {
		return nil, fmt.Errorf("unmarshal tokens failed: %w", err)
	}
	return &pair, nil
}

func (r *redisCache) Set(ctx context.Context, key string, value TokenPair) error {
	valueBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal tokens failed: %w", err)
	}
	return r.client.Set(ctx, key, rueidis.BinaryString(valueBytes), r.ttl)
}
