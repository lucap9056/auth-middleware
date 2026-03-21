package jwt

import (
	"os"
	"testing"
	"time"
)

func TestParseDurationWithDays(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"15m", 15 * time.Minute, false},
		{"1h", time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"30d", 30 * 24 * time.Hour, false},
		{"invalid", 0, true},
		{"", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseDurationWithDays(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDurationWithDays(%s) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("parseDurationWithDays(%s) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFromEnv(t *testing.T) {
	os.Setenv("JWT_ACCESS_TOKEN_DURATION", "30m")
	os.Setenv("JWT_REFRESH_TOKEN_DURATION", "14d")
	defer func() {
		os.Unsetenv("JWT_ACCESS_TOKEN_DURATION")
		os.Unsetenv("JWT_REFRESH_TOKEN_DURATION")
	}()

	cfg := defaultOptions()
	FromEnv()(cfg)

	if cfg.AccessTokenDuration != 30*time.Minute {
		t.Errorf("expected access token duration 30m, got %v", cfg.AccessTokenDuration)
	}
	if cfg.RefreshTokenDuration != 14*24*time.Hour {
		t.Errorf("expected refresh token duration 14d, got %v", cfg.RefreshTokenDuration)
	}
}

func TestWithDurations(t *testing.T) {
	cfg := defaultOptions()
	
	WithAccessTokenDuration(time.Hour)(cfg)
	if cfg.AccessTokenDuration != time.Hour {
		t.Errorf("expected access token duration 1h, got %v", cfg.AccessTokenDuration)
	}

	WithRefreshTokenDuration(24*time.Hour)(cfg)
	if cfg.RefreshTokenDuration != 24*time.Hour {
		t.Errorf("expected refresh token duration 24h, got %v", cfg.RefreshTokenDuration)
	}
}
