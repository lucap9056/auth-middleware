package database

import (
	"time"

	"github.com/lucap9056/auth-middleware/database/v2/schema"
)

type options struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	CleanupInterval time.Duration

	Schema           *schema.Params
	AutoCreateSchema bool
}

type Option func(*options)

func defaultOptions() *options {
	return &options{
		MaxOpenConns:    20,
		MaxIdleConns:    15,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 2 * time.Minute,
		CleanupInterval: 24 * time.Hour,
		Schema:          schema.DefaultParams(),
	}
}

func WithMaxOpenConns(n int) Option {
	return func(o *options) {
		o.MaxOpenConns = n
	}
}

func WithMaxIdleConns(n int) Option {
	return func(o *options) {
		o.MaxIdleConns = n
	}
}

func WithCleanupInterval(d time.Duration) Option {
	return func(o *options) {
		o.CleanupInterval = d
	}
}

func WithConnMaxLifetime(d time.Duration) Option {
	return func(o *options) {
		o.ConnMaxLifetime = d
	}
}

func WithConnMaxIdleTime(d time.Duration) Option {
	return func(o *options) {
		o.ConnMaxIdleTime = d
	}
}

func WithAutoCreateSchema(enabled bool) Option {
	return func(o *options) {
		o.AutoCreateSchema = enabled
	}
}

func WithUserEmailReference(reference string) (Option, error) {
	params, err := schema.ParseUserEmailReference(reference)
	if err != nil {
		return nil, err
	}

	return func(o *options) {
		o.Schema = params
	}, nil
}
