package device

import (
	"context"
	"log"
	"time"

	"github.com/lucap9056/auth-middleware/oauth2/internal/cache"
	"github.com/redis/rueidis"
)

const (
	secretKeyPrefix = "oauth2:device:secret:"
	opTimeout       = time.Second
)

type redisSecretCache struct {
	client *cache.RedisClient
	ttl    time.Duration
}

func newRedisSecretCache(client *cache.RedisClient) *redisSecretCache {
	return &redisSecretCache{client: client, ttl: secretTTL}
}

func (r *redisSecretCache) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), opTimeout)
}

func (r *redisSecretCache) GetSecret(deviceID string) (string, bool) {
	ctx, cancel := r.ctx()
	defer cancel()

	secret, err := r.client.Get(ctx, secretKeyPrefix+deviceID).ToString()
	if err != nil {
		if !rueidis.IsRedisNil(err) {
			log.Printf("[WARN] Failed to get device secret from redis: %v", err)
		}
		return "", false
	}
	if secret == "" {
		return "", false
	}
	return secret, true
}

func (r *redisSecretCache) SetSecret(deviceID, secret string, overwrite bool) {
	ctx, cancel := r.ctx()
	defer cancel()

	if overwrite {
		r.client.Set(ctx, secretKeyPrefix+deviceID, secret, r.ttl)
		return
	}
	r.client.SetNX(ctx, secretKeyPrefix+deviceID, secret, r.ttl)
}

func (r *redisSecretCache) DeleteSecret(deviceID string) {
	ctx, cancel := r.ctx()
	defer cancel()
	r.client.Set(ctx, secretKeyPrefix+deviceID, "", r.ttl+deletedSecretExtraTTL)
}

func (r *redisSecretCache) Close() error {
	return nil
}
