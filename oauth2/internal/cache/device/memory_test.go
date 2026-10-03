package device

import (
	"testing"
	"time"
)

var _ SecretCache = (*memorySecretCache)(nil)

func newTestMemoryCache(t *testing.T, ttl time.Duration) *memorySecretCache {
	t.Helper()
	c, err := newMemorySecretCache(100, ttl)
	if err != nil {
		t.Fatalf("newMemorySecretCache: %v", err)
	}
	return c
}

func TestMemorySecretCache_SetGet(t *testing.T) {
	c := newTestMemoryCache(t, time.Minute)
	c.SetSecret("d1", Secret{Value: "s1", Generation: 1})

	got, ok := c.GetSecret("d1")
	if !ok || got != (Secret{Value: "s1", Generation: 1}) {
		t.Fatalf("GetSecret = %+v, %v; want {s1 1}, true", got, ok)
	}
	if _, ok := c.GetSecret("missing"); ok {
		t.Fatal("GetSecret(missing) should miss")
	}
}

func TestMemorySecretCache_Expires(t *testing.T) {
	c := newTestMemoryCache(t, 50*time.Millisecond)
	c.SetSecret("d1", Secret{Value: "s1", Generation: 1})
	time.Sleep(100 * time.Millisecond)

	if _, ok := c.GetSecret("d1"); ok {
		t.Fatal("secret should have expired")
	}
}

func TestMemorySecretCache_Delete(t *testing.T) {
	c := newTestMemoryCache(t, time.Minute)
	c.SetSecret("d1", Secret{Value: "s1", Generation: 1})
	c.SetSecret("d2", Secret{Value: "s2", Generation: 1})

	c.DeleteSecret("d1")
	if _, ok := c.GetSecret("d1"); ok {
		t.Fatal("d1 should be deleted")
	}
	if _, ok := c.GetSecret("d2"); !ok {
		t.Fatal("d2 should remain")
	}
}

func TestNewMemorySecretCache_InvalidSize(t *testing.T) {
	if _, err := newMemorySecretCache(0, time.Minute); err == nil {
		t.Fatal("expected error for zero maximum size")
	}
}

func TestMemorySecretCache_NewerGenerationReplaces(t *testing.T) {
	c := newTestMemoryCache(t, time.Minute)
	c.SetSecret("d1", Secret{Value: "s1", Generation: 1})
	c.SetSecret("d1", Secret{Value: "s1", Generation: 2})

	if got, _ := c.GetSecret("d1"); got.Generation != 2 {
		t.Fatalf("Generation = %d; want 2", got.Generation)
	}
}

func TestMemorySecretCache_StaleGenerationIgnored(t *testing.T) {
	c := newTestMemoryCache(t, time.Minute)
	c.SetSecret("d1", Secret{Value: "s1", Generation: 3})
	c.SetSecret("d1", Secret{Value: "s1", Generation: 2})

	if got, _ := c.GetSecret("d1"); got.Generation != 3 {
		t.Fatalf("Generation = %d; want 3", got.Generation)
	}
}

func TestMemorySecretCache_CannotRestoreDeleted(t *testing.T) {
	c := newTestMemoryCache(t, time.Minute)
	c.SetSecret("d1", Secret{Value: "s1", Generation: 1})
	c.DeleteSecret("d1")
	c.SetSecret("d1", Secret{Value: "s1", Generation: 2})

	if _, ok := c.GetSecret("d1"); ok {
		t.Fatal("deleted secret must not be restored")
	}
}
