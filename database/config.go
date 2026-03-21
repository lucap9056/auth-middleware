package database

import (
	"os"
	"strconv"
	"time"
)

type databaseOptions struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	CleanupInterval time.Duration
}

type DatabaseOption func(*databaseOptions)

func defaultOptions() *databaseOptions {
	return &databaseOptions{
		MaxOpenConns:    20,
		MaxIdleConns:    15,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 2 * time.Minute,
		CleanupInterval: 24 * time.Hour,
	}
}

func WithMaxOpenConns(n int) DatabaseOption {
	return func(o *databaseOptions) {
		o.MaxOpenConns = n
	}
}

func WithMaxIdleConns(n int) DatabaseOption {
	return func(o *databaseOptions) {
		o.MaxIdleConns = n
	}
}

func WithCleanupInterval(d time.Duration) DatabaseOption {
	return func(o *databaseOptions) {
		o.CleanupInterval = d
	}
}

func WithConnMaxLifetime(d time.Duration) DatabaseOption {
	return func(o *databaseOptions) {
		o.ConnMaxLifetime = d
	}
}

func WithConnMaxIdleTime(d time.Duration) DatabaseOption {
	return func(o *databaseOptions) {
		o.ConnMaxIdleTime = d
	}
}

func FromEnv() DatabaseOption {
	return func(o *databaseOptions) {
		if val := os.Getenv("DB_MAX_OPEN_CONNS"); val != "" {
			if n, err := strconv.Atoi(val); err == nil {
				o.MaxOpenConns = n
			}
		}

		if val := os.Getenv("DB_MAX_IDLE_CONNS"); val != "" {
			if n, err := strconv.Atoi(val); err == nil {
				o.MaxIdleConns = n
			}
		}

		if val := os.Getenv("DB_CONN_MAX_LIFETIME"); val != "" {
			if n, err := strconv.Atoi(val); err == nil {
				o.ConnMaxLifetime = time.Duration(n) * time.Minute
			}
		}

		if val := os.Getenv("DB_CONN_MAX_IDLE_TIME"); val != "" {
			if n, err := strconv.Atoi(val); err == nil {
				o.ConnMaxIdleTime = time.Duration(n) * time.Minute
			}
		}

		if val := os.Getenv("DB_CLEANUP_INTERVAL"); val != "" {
			if n, err := strconv.Atoi(val); err == nil {
				o.CleanupInterval = time.Duration(n) * time.Hour
			}
		}
	}
}
