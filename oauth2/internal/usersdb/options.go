package usersdb

import "github.com/lucap9056/auth-middleware/database/v2"

type externalOptions struct {
	userEmailReference string
	usernameColumn     string
}

type options struct {
	external         *externalOptions
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

func WithExternal(userEmailReference, usernameColumn string) Option {
	return func(o *options) {
		o.external = &externalOptions{
			userEmailReference: userEmailReference,
			usernameColumn:     usernameColumn,
		}
	}
}
