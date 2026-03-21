package jwt

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type jwtOptions struct {
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
}

type JWTOption func(*jwtOptions)

func defaultOptions() *jwtOptions {
	return &jwtOptions{
		AccessTokenDuration:  15 * time.Minute,
		RefreshTokenDuration: 7 * 24 * time.Hour,
	}
}

func WithAccessTokenDuration(d time.Duration) JWTOption {
	return func(o *jwtOptions) {
		o.AccessTokenDuration = d
	}
}

func WithRefreshTokenDuration(d time.Duration) JWTOption {
	return func(o *jwtOptions) {
		o.RefreshTokenDuration = d
	}
}

func FromEnv() JWTOption {
	return func(o *jwtOptions) {
		if val := os.Getenv("JWT_ACCESS_TOKEN_DURATION"); val != "" {
			if d, err := parseDurationWithDays(val); err == nil {
				o.AccessTokenDuration = d
			}
		}

		if val := os.Getenv("JWT_REFRESH_TOKEN_DURATION"); val != "" {
			if d, err := parseDurationWithDays(val); err == nil {
				o.RefreshTokenDuration = d
			}
		}
	}
}

func parseDurationWithDays(s string) (time.Duration, error) {
	s = strings.ToLower(strings.TrimSpace(s))

	if strings.HasSuffix(s, "d") {
		dayStr := strings.TrimSuffix(s, "d")
		days, err := strconv.Atoi(dayStr)
		if err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}

	return time.ParseDuration(s)
}
