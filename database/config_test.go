package database

import (
	"os"
	"testing"
	"time"
)

func TestFromEnv(t *testing.T) {
	envKeys := []string{"DB_MAX_OPEN_CONNS", "DB_MAX_IDLE_CONNS", "DB_CONN_MAX_LIFETIME", "DB_CONN_MAX_IDLE_TIME", "DB_CLEANUP_INTERVAL"}
	for _, key := range envKeys {
		os.Unsetenv(key)
	}

	tests := []struct {
		name     string
		setupEnv func()
		validate func(*testing.T, *databaseOptions)
	}{
		{
			name: "Valid environment variables",
			setupEnv: func() {
				os.Setenv("DB_MAX_OPEN_CONNS", "50")
				os.Setenv("DB_MAX_IDLE_CONNS", "10")
				os.Setenv("DB_CONN_MAX_LIFETIME", "10")
				os.Setenv("DB_CONN_MAX_IDLE_TIME", "5")
				os.Setenv("DB_CLEANUP_INTERVAL", "48")
			},
			validate: func(t *testing.T, o *databaseOptions) {
				if o.MaxOpenConns != 50 {
					t.Errorf("expected MaxOpenConns 50, got %d", o.MaxOpenConns)
				}
				if o.MaxIdleConns != 10 {
					t.Errorf("expected MaxIdleConns 10, got %d", o.MaxIdleConns)
				}
				if o.ConnMaxLifetime != 10*time.Minute {
					t.Errorf("expected ConnMaxLifetime 10m, got %v", o.ConnMaxLifetime)
				}
				if o.ConnMaxIdleTime != 5*time.Minute {
					t.Errorf("expected ConnMaxIdleTime 5m, got %v", o.ConnMaxIdleTime)
				}
				if o.CleanupInterval != 48*time.Hour {
					t.Errorf("expected CleanupInterval 48h, got %v", o.CleanupInterval)
				}
			},
		},
		{
			name: "Invalid environment variables should fallback to current value",
			setupEnv: func() {
				os.Setenv("DB_MAX_OPEN_CONNS", "invalid_number")
				os.Setenv("DB_MAX_IDLE_CONNS", "not_a_number")
				os.Setenv("DB_CONN_MAX_LIFETIME", "abc")
				os.Setenv("DB_CONN_MAX_IDLE_TIME", "xyz")
				os.Setenv("DB_CLEANUP_INTERVAL", "!!!")
			},
			validate: func(t *testing.T, o *databaseOptions) {
				if o.MaxOpenConns != 20 {
					t.Errorf("expected MaxOpenConns to remain 20, got %d", o.MaxOpenConns)
				}
				if o.MaxIdleConns != 15 {
					t.Errorf("expected MaxIdleConns to remain 15, got %d", o.MaxIdleConns)
				}
				if o.ConnMaxLifetime != 5*time.Minute {
					t.Errorf("expected ConnMaxLifetime to remain 5m, got %v", o.ConnMaxLifetime)
				}
				if o.ConnMaxIdleTime != 2*time.Minute {
					t.Errorf("expected ConnMaxIdleTime to remain 2m, got %v", o.ConnMaxIdleTime)
				}
				if o.CleanupInterval != 24*time.Hour {
					t.Errorf("expected CleanupInterval to remain 24h, got %v", o.CleanupInterval)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, key := range envKeys {
				os.Unsetenv(key)
			}

			tt.setupEnv()
			opts := defaultOptions()
			FromEnv()(opts)
			tt.validate(t, opts)
		})
	}
}
