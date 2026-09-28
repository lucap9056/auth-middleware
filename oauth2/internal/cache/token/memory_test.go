package token

import (
	"context"
	"testing"
	"time"
)

func TestMemoryCache_SetGet(t *testing.T) {
	ctx := context.Background()
	c, err := NewCache(nil)
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}

	if got, err := c.Get(ctx, "missing"); got != nil || err != nil {
		t.Fatalf("Get(missing) = %v, %v; want nil, nil", got, err)
	}

	want := TokenPair{AccessToken: "a", RefreshToken: "r"}
	if err := c.Set(ctx, "k", want); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := c.Get(ctx, "k")
	if err != nil || got == nil || *got != want {
		t.Fatalf("Get = %v, %v; want %v", got, err, want)
	}
}

func TestMemoryCache_Expires(t *testing.T) {
	ctx := context.Background()
	c, err := NewCache(nil, WithTTL(50*time.Millisecond))
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}

	c.Set(ctx, "k", TokenPair{AccessToken: "a"})
	time.Sleep(100 * time.Millisecond)

	if got, _ := c.Get(ctx, "k"); got != nil {
		t.Fatalf("Get after TTL = %v; want nil", got)
	}
}

func TestNewCache_InvalidMaximumSize(t *testing.T) {
	if _, err := NewCache(nil, WithMaximumSize(0)); err == nil {
		t.Fatal("expected error for zero maximum size")
	}
}
