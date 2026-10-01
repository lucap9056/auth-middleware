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
	c.SetSecret("d1", "s1", true)

	got, ok := c.GetSecret("d1")
	if !ok || got != "s1" {
		t.Fatalf("GetSecret = %q, %v; want s1, true", got, ok)
	}
	if _, ok := c.GetSecret("missing"); ok {
		t.Fatal("GetSecret(missing) should miss")
	}
}

func TestMemorySecretCache_Expires(t *testing.T) {
	c := newTestMemoryCache(t, 50*time.Millisecond)
	c.SetSecret("d1", "s1", true)
	time.Sleep(100 * time.Millisecond)

	if _, ok := c.GetSecret("d1"); ok {
		t.Fatal("secret should have expired")
	}
}

func TestMemorySecretCache_Delete(t *testing.T) {
	c := newTestMemoryCache(t, time.Minute)
	c.SetSecret("d1", "s1", true)
	c.SetSecret("d2", "s2", true)

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

func TestMemorySecretCache_Close(t *testing.T) {
	c := newTestMemoryCache(t, time.Minute)
	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestMemorySecretCache_WithoutOverwriteKeepsNewerSecret(t *testing.T) {
	c := newTestMemoryCache(t, time.Minute)
	c.SetSecret("d1", "rotated", true)
	c.SetSecret("d1", "stale", false)

	if got, _ := c.GetSecret("d1"); got != "rotated" {
		t.Fatalf("GetSecret = %q; want rotated", got)
	}
}

func TestMemorySecretCache_WithoutOverwriteCannotRestoreDeleted(t *testing.T) {
	c := newTestMemoryCache(t, time.Minute)
	c.SetSecret("d1", "s1", true)
	c.DeleteSecret("d1")
	c.SetSecret("d1", "s1", false)

	if _, ok := c.GetSecret("d1"); ok {
		t.Fatal("deleted secret must not be restored by a stale fill")
	}
}

func TestMemorySecretCache_WithoutOverwriteFillsMiss(t *testing.T) {
	c := newTestMemoryCache(t, time.Minute)
	c.SetSecret("d1", "s1", false)

	if got, ok := c.GetSecret("d1"); !ok || got != "s1" {
		t.Fatalf("GetSecret = %q, %v; want s1, true", got, ok)
	}
}
