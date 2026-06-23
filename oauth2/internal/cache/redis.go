package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient(url string) (*redis.Client, error) {
	if url == "" {
		return nil, nil
	}

	opt, err := redis.ParseURL(url)
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
		rdb.Close()
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return rdb, nil
}
