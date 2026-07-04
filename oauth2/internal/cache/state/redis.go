package state

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const redisKeyPrefix = "oauth2:state:"

type redisCache struct {
	client *redis.Client
	ttl    time.Duration
}

func newRedisCache(client *redis.Client, ttl time.Duration) *redisCache {
	return &redisCache{client: client, ttl: ttl}
}

func (r *redisCache) Get(ctx context.Context, state string) (string, error) {
	val, err := r.client.Get(ctx, redisKeyPrefix+state).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("redis get failed: %w", err)
	}
	return val, nil
}

func (r *redisCache) Set(ctx context.Context, state, verifier string) error {
	return r.client.Set(ctx, redisKeyPrefix+state, verifier, r.ttl).Err()
}

func (r *redisCache) Delete(ctx context.Context, state string) error {
	return r.client.Del(ctx, redisKeyPrefix+state).Err()
}
