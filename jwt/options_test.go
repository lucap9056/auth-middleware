package jwt

import (
	"testing"
	"time"
)

func TestDefaultOptions(t *testing.T) {
	cfg := defaultOptions()

	if cfg.AccessTokenDuration != 15*time.Minute {
		t.Errorf("expected access token duration 15m, got %v", cfg.AccessTokenDuration)
	}
	if cfg.RefreshTokenDuration != 7*24*time.Hour {
		t.Errorf("expected refresh token duration 7d, got %v", cfg.RefreshTokenDuration)
	}
	if cfg.Issuer != "" || cfg.Audience != "" {
		t.Errorf("expected empty issuer and audience, got %q and %q", cfg.Issuer, cfg.Audience)
	}
}
