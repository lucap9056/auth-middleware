package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/rueidis"
)

type RedisClient struct {
	rueidis.Client
}

func NewRedisClient(url string) (*RedisClient, error) {
	if url == "" {
		return nil, nil
	}

	opt, err := rueidis.ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("invalid redis url: %w", err)
	}

	opt.Dialer.Timeout = 5 * time.Second

	client, err := rueidis.NewClient(opt)
	if err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Do(ctx, client.B().Ping().Build()).Error(); err != nil {
		client.Close()
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &RedisClient{Client: client}, nil
}

func (c *RedisClient) Get(ctx context.Context, key string) rueidis.RedisResult {
	return c.Do(ctx, c.B().Get().Key(key).Build())
}

func (c *RedisClient) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	return c.Do(ctx, c.B().Set().Key(key).Value(value).Px(ttl).Build()).Error()
}

func (c *RedisClient) SetNX(ctx context.Context, key, value string, ttl time.Duration) error {
	return c.Do(ctx, c.B().Set().Key(key).Value(value).Nx().Px(ttl).Build()).Error()
}

func (c *RedisClient) Del(ctx context.Context, key string) error {
	return c.Do(ctx, c.B().Del().Key(key).Build()).Error()
}
