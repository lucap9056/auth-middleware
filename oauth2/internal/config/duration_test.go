package config

import (
	"errors"
	"testing"
	"time"
)

func TestParseDuration_Valid(t *testing.T) {
	cases := []struct {
		value    string
		bareUnit time.Duration
		expected time.Duration
	}{
		{"15m", 0, 15 * time.Minute},
		{"7d", 0, 7 * 24 * time.Hour},
		{"1d12h", 0, 36 * time.Hour},
		{"1h30m15s", 0, time.Hour + 30*time.Minute + 15*time.Second},
		{" 2H ", 0, 2 * time.Hour},
		{"0s", 0, 0},
		{"5", time.Minute, 5 * time.Minute},
		{"24", time.Hour, 24 * time.Hour},
		{"30s", time.Minute, 30 * time.Second},
	}
	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			got, err := parseDuration(tc.value, tc.bareUnit)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.expected {
				t.Errorf("got %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestParseDuration_Invalid(t *testing.T) {
	cases := []struct {
		name     string
		value    string
		bareUnit time.Duration
	}{
		{"empty", "", 0},
		{"bare number without unit", "15", 0},
		{"unknown unit", "15w", 0},
		{"trailing digits", "1h30", 0},
		{"missing digits", "h", 0},
		{"negative", "-5m", 0},
		{"fraction", "1.5h", 0},
		{"go style milliseconds", "500ms", 0},
		{"unit overflow", "999999999999d", 0},
		{"sum overflow", "106751d106751d", 0},
		{"bare overflow", "9999999999999", time.Hour},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := parseDuration(tc.value, tc.bareUnit); !errors.Is(err, ErrInvalidDuration) {
				t.Fatalf("got error %v, want %v", err, ErrInvalidDuration)
			}
		})
	}
}
