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

func TestRedisSecretCache_WithoutOverwriteKeepsNewerSecret(t *testing.T) {
	c := newTestRedisCache(t)
	deviceID := uniqueDeviceID(t)

	c.SetSecret(deviceID, "rotated", true)
	c.SetSecret(deviceID, "stale", false)

	if got, _ := c.GetSecret(deviceID); got != "rotated" {
		t.Fatalf("GetSecret = %q; want rotated", got)
	}
}

func TestRedisSecretCache_WithoutOverwriteCannotRestoreDeleted(t *testing.T) {
	c := newTestRedisCache(t)
	deviceID := uniqueDeviceID(t)

	c.SetSecret(deviceID, "s1", true)
	c.DeleteSecret(deviceID)
	c.SetSecret(deviceID, "s1", false)

	if _, ok := c.GetSecret(deviceID); ok {
		t.Fatal("deleted secret must not be restored by a stale fill")
	}
}

func TestRedisSecretCache_WithoutOverwriteFillsMiss(t *testing.T) {
	c := newTestRedisCache(t)
	deviceID := uniqueDeviceID(t)

	c.SetSecret(deviceID, "s1", false)

	if got, ok := c.GetSecret(deviceID); !ok || got != "s1" {
		t.Fatalf("GetSecret = %q, %v; want s1, true", got, ok)
	}
}
