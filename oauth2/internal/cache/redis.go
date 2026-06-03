package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type redisCache struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisCache(connectionString string, ttl time.Duration) (*redisCache, error) {
	opt, err := redis.ParseURL(connectionString)
	if err != nil {
		return nil, fmt.Errorf("invalid redis url: %w", err)
	}

	opt.PoolSize = 20
	opt.MinIdleConns = 5
	opt.DialTimeout = 5 * time.Second

	rdb := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &redisCache{client: rdb, ttl: ttl}, nil
}

func (r *redisCache) Get(ctx context.Context, key string) (*TokenPair, error) {
	val, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
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

	return r.client.Set(ctx, key, valueBytes, r.ttl).Err()
}

func (r *redisCache) Close() error {
	return r.client.Close()
}
