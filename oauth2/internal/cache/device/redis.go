package device

import (
	"context"
	"log"
	"time"

	"github.com/redis/rueidis"
)

const (
	secretKeyPrefix   = "oauth2:device:secret:"
	userDevicesPrefix = "oauth2:user:devices:"
	userDevicesTTL    = 30 * 24 * time.Hour
	opTimeout         = time.Second
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

func (r *redisSecretCache) AddUserDevice(userID, deviceID string) {
	ctx, cancel := r.ctx()
	defer cancel()
	key := userDevicesPrefix + userID
	r.client.Do(ctx, r.client.B().Sadd().Key(key).Member(deviceID).Build())
	r.client.Do(ctx, r.client.B().Expire().Key(key).Seconds(int64(userDevicesTTL/time.Second)).Build())
}

func (r *redisSecretCache) RemoveUserDevice(userID, deviceID string) {
	ctx, cancel := r.ctx()
	defer cancel()
	r.client.Do(ctx, r.client.B().Srem().Key(userDevicesPrefix+userID).Member(deviceID).Build())
}

func (r *redisSecretCache) PopAllUserDevices(userID string) []string {
	ctx, cancel := r.ctx()
	defer cancel()
	key := userDevicesPrefix + userID
	ids, err := r.client.Do(ctx, r.client.B().Smembers().Key(key).Build()).AsStrSlice()
	if err != nil {
		return nil
	}
	r.client.Do(ctx, r.client.B().Del().Key(key).Build())
	return ids
}
