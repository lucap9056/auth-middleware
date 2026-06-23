package device

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	secretKeyPrefix   = "oauth2:device:secret:"
	userDevicesPrefix = "oauth2:user:devices:"
	userDevicesTTL    = 30 * 24 * time.Hour
	opTimeout         = time.Second
)

type redisSecretCache struct {
	client *redis.Client
	ttl    time.Duration
}

func newRedisSecretCache(client *redis.Client) *redisSecretCache {
	return &redisSecretCache{client: client, ttl: secretTTL}
}

func (r *redisSecretCache) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), opTimeout)
}

func (r *redisSecretCache) GetSecret(deviceID string) (string, bool) {
	ctx, cancel := r.ctx()
	defer cancel()
	val, err := r.client.Get(ctx, secretKeyPrefix+deviceID).Result()
	if err != nil {
		return "", false
	}
	return val, true
}

func (r *redisSecretCache) SetSecret(deviceID, secret string) {
	ctx, cancel := r.ctx()
	defer cancel()
	r.client.Set(ctx, secretKeyPrefix+deviceID, secret, r.ttl)
}

func (r *redisSecretCache) DeleteSecret(deviceID string) {
	ctx, cancel := r.ctx()
	defer cancel()
	r.client.Del(ctx, secretKeyPrefix+deviceID)
}

func (r *redisSecretCache) AddUserDevice(userID, deviceID string) {
	ctx, cancel := r.ctx()
	defer cancel()
	key := userDevicesPrefix + userID
	r.client.SAdd(ctx, key, deviceID)
	r.client.Expire(ctx, key, userDevicesTTL)
}

func (r *redisSecretCache) RemoveUserDevice(userID, deviceID string) {
	ctx, cancel := r.ctx()
	defer cancel()
	r.client.SRem(ctx, userDevicesPrefix+userID, deviceID)
}

func (r *redisSecretCache) PopAllUserDevices(userID string) []string {
	ctx, cancel := r.ctx()
	defer cancel()
	key := userDevicesPrefix + userID
	ids, err := r.client.SMembers(ctx, key).Result()
	if err != nil {
		return nil
	}
	r.client.Del(ctx, key)
	return ids
}
