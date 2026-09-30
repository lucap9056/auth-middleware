package device

import (
	"context"
	"log"
	"time"

	"github.com/redis/rueidis"
)

const (
	secretKeyPrefix = "oauth2:device:secret:"
	opTimeout       = time.Second
)

type redisSecretCache struct {
	client rueidis.Client
	ttl    time.Duration
}

func newRedisSecretCache(client rueidis.Client) *redisSecretCache {
	return &redisSecretCache{client: client, ttl: secretTTL}
}

func (r *redisSecretCache) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), opTimeout)
}

func (r *redisSecretCache) GetSecret(deviceID string) (string, bool) {
	ctx, cancel := r.ctx()
	defer cancel()

	secret, err := r.client.Do(ctx, r.client.B().Get().Key(secretKeyPrefix+deviceID).Build()).ToString()
	if err != nil {
		if !rueidis.IsRedisNil(err) {
			log.Printf("[WARN] Failed to get device secret from redis: %v", err)
		}
		return "", false
	}
	return secret, true
}

func (r *redisSecretCache) SetSecret(deviceID, secret string) {
	ctx, cancel := r.ctx()
	defer cancel()
	r.client.Do(ctx, r.client.B().Set().Key(secretKeyPrefix+deviceID).Value(secret).Px(r.ttl).Build())
}

func (r *redisSecretCache) DeleteSecret(deviceID string) {
	ctx, cancel := r.ctx()
	defer cancel()
	r.client.Do(ctx, r.client.B().Del().Key(secretKeyPrefix+deviceID).Build())
}

func (r *redisSecretCache) Close() error {
	return nil
}
