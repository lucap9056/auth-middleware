package device

import (
	"os"
	"testing"
	"time"

	"github.com/lucap9056/auth-middleware/oauth2/internal/cache"
)

var _ SecretCache = (*redisSecretCache)(nil)

func newTestRedisCache(t *testing.T) *redisSecretCache {
	t.Helper()
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		t.Skip("TEST_REDIS_URL not set")
	}

	client, err := cache.NewRedisClient(url)
	if err != nil {
		t.Fatalf("NewRedisClient: %v", err)
	}
	t.Cleanup(client.Close)
	return newRedisSecretCache(client)
}

func uniqueDeviceID(t *testing.T) string {
	return t.Name() + ":" + time.Now().Format(time.RFC3339Nano)
}

func TestRedisSecretCache_SetGet(t *testing.T) {
	c := newTestRedisCache(t)
	deviceID := uniqueDeviceID(t)

	c.SetSecret(deviceID, Secret{Value: "s1", Generation: 1})

	if got, ok := c.GetSecret(deviceID); !ok || got != (Secret{Value: "s1", Generation: 1}) {
		t.Fatalf("GetSecret = %+v, %v; want {s1 1}, true", got, ok)
	}
}

func TestRedisSecretCache_NewerGenerationReplaces(t *testing.T) {
	c := newTestRedisCache(t)
	deviceID := uniqueDeviceID(t)

	c.SetSecret(deviceID, Secret{Value: "s1", Generation: 1})
	c.SetSecret(deviceID, Secret{Value: "s1", Generation: 2})

	if got, _ := c.GetSecret(deviceID); got.Generation != 2 {
		t.Fatalf("Generation = %d; want 2", got.Generation)
	}
}

func TestRedisSecretCache_StaleGenerationIgnored(t *testing.T) {
	c := newTestRedisCache(t)
	deviceID := uniqueDeviceID(t)

	c.SetSecret(deviceID, Secret{Value: "s1", Generation: 3})
	c.SetSecret(deviceID, Secret{Value: "s1", Generation: 2})

	if got, _ := c.GetSecret(deviceID); got.Generation != 3 {
		t.Fatalf("Generation = %d; want 3", got.Generation)
	}
}

func TestRedisSecretCache_CannotRestoreDeleted(t *testing.T) {
	c := newTestRedisCache(t)
	deviceID := uniqueDeviceID(t)

	c.SetSecret(deviceID, Secret{Value: "s1", Generation: 1})
	c.DeleteSecret(deviceID)
	c.SetSecret(deviceID, Secret{Value: "s1", Generation: 2})

	if _, ok := c.GetSecret(deviceID); ok {
		t.Fatal("deleted secret must not be restored")
	}
}
