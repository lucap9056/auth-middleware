package usersdb

import "github.com/lucap9056/auth-middleware/database/v2"

type options struct {
	autoCreateSchema bool
	databaseOptions  []database.Option
}

type Option func(*options)

func newOptions(opts []Option) *options {
	cfg := &options{}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

func WithDatabaseOptions(opts ...database.Option) Option {
	return func(o *options) {
		o.databaseOptions = append(o.databaseOptions, opts...)
	}
}

func WithAutoCreateSchema(enabled bool) Option {
	return func(o *options) {
		o.autoCreateSchema = enabled
	}
}
