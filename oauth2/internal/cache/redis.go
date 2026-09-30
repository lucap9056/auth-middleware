package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/rueidis"
)

func NewRedisClient(url string) (rueidis.Client, error) {
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

	return client, nil
}
