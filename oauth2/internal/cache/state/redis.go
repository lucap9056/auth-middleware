package state

import (
	"context"
	"fmt"
	"time"

	"github.com/lucap9056/auth-middleware/oauth2/internal/cache"
	"github.com/redis/rueidis"
)

const redisKeyPrefix = "oauth2:state:"

type redisCache struct {
	client *cache.RedisClient
	ttl    time.Duration
}

func newRedisCache(client *cache.RedisClient, ttl time.Duration) *redisCache {
	return &redisCache{client: client, ttl: ttl}
}

func (r *redisCache) Get(ctx context.Context, state string) (string, error) {
	val, err := r.client.Get(ctx, redisKeyPrefix+state).ToString()
	if rueidis.IsRedisNil(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("redis get failed: %w", err)
	}
	return val, nil
}

func (r *redisCache) Set(ctx context.Context, state, verifier string) error {
	return r.client.Set(ctx, redisKeyPrefix+state, verifier, r.ttl)
}

func (r *redisCache) Delete(ctx context.Context, state string) error {
	return r.client.Del(ctx, redisKeyPrefix+state)
}
