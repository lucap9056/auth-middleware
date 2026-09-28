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
	c.SetSecret("d1", "s1")

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
	c.SetSecret("d1", "s1")
	time.Sleep(100 * time.Millisecond)

	if _, ok := c.GetSecret("d1"); ok {
		t.Fatal("secret should have expired")
	}
}

func TestMemorySecretCache_Delete(t *testing.T) {
	c := newTestMemoryCache(t, time.Minute)
	c.SetSecret("d1", "s1")
	c.SetSecret("d2", "s2")

	c.DeleteSecret("d1")
	if _, ok := c.GetSecret("d1"); ok {
		t.Fatal("d1 should be deleted")
	}
	if _, ok := c.GetSecret("d2"); !ok {
		t.Fatal("d2 should remain")
	}
}

func TestMemorySecretCache_UserDevices(t *testing.T) {
	c := newTestMemoryCache(t, time.Minute)
	c.AddUserDevice("u1", "d1")
	c.AddUserDevice("u1", "d2")
	c.RemoveUserDevice("u1", "d2")

	ids := c.PopAllUserDevices("u1")
	if len(ids) != 1 || ids[0] != "d1" {
		t.Fatalf("PopAllUserDevices = %v; want [d1]", ids)
	}
	if ids := c.PopAllUserDevices("u1"); ids != nil {
		t.Fatalf("second PopAllUserDevices = %v; want nil", ids)
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
