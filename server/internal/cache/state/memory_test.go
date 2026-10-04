package state

import (
	"context"
	"testing"
	"time"
)

func TestMemoryCache_TakeIsSingleUse(t *testing.T) {
	ctx := context.Background()
	c, err := NewCache(nil)
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}

	if err := c.Set(ctx, "s1", "v1"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if got, _ := c.Take(ctx, "s1"); got != "v1" {
		t.Fatalf("Take = %q; want v1", got)
	}
	if got, _ := c.Take(ctx, "s1"); got != "" {
		t.Fatalf("second Take = %q; want empty", got)
	}
}

func TestMemoryCache_Expires(t *testing.T) {
	ctx := context.Background()
	c, err := NewCache(nil, WithTTL(50*time.Millisecond))
	if err != nil {
		t.Fatalf("NewCache: %v", err)
	}

	c.Set(ctx, "s1", "v1")
	time.Sleep(100 * time.Millisecond)

	if got, _ := c.Take(ctx, "s1"); got != "" {
		t.Fatalf("Take after TTL = %q; want empty", got)
	}
}

func TestNewCache_InvalidMaximumSize(t *testing.T) {
	if _, err := NewCache(nil, WithMaximumSize(0)); err == nil {
		t.Fatal("expected error for zero maximum size")
	}
}
