package token

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/rueidis"
)

type redisCache struct {
	client rueidis.Client
	ttl    time.Duration
}

func newRedisCache(client rueidis.Client, ttl time.Duration) *redisCache {
	return &redisCache{client: client, ttl: ttl}
}

func (r *redisCache) Get(ctx context.Context, key string) (*TokenPair, error) {
	val, err := r.client.Do(ctx, r.client.B().Get().Key(key).Build()).AsBytes()
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
	return r.client.Do(ctx, r.client.B().Set().Key(key).Value(rueidis.BinaryString(valueBytes)).Px(r.ttl).Build()).Error()
}
