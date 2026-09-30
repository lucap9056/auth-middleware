package state

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/rueidis"
)

const redisKeyPrefix = "oauth2:state:"

type redisCache struct {
	client rueidis.Client
	ttl    time.Duration
}

func newRedisCache(client rueidis.Client, ttl time.Duration) *redisCache {
	return &redisCache{client: client, ttl: ttl}
}

func (r *redisCache) Get(ctx context.Context, state string) (string, error) {
	val, err := r.client.Do(ctx, r.client.B().Get().Key(redisKeyPrefix+state).Build()).ToString()
	if rueidis.IsRedisNil(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("redis get failed: %w", err)
	}
	return val, nil
}

func (r *redisCache) Set(ctx context.Context, state, verifier string) error {
	return r.client.Do(ctx, r.client.B().Set().Key(redisKeyPrefix+state).Value(verifier).Px(r.ttl).Build()).Error()
}

func (r *redisCache) Delete(ctx context.Context, state string) error {
	return r.client.Do(ctx, r.client.B().Del().Key(redisKeyPrefix+state).Build()).Error()
}
