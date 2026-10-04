package providers

import (
	"context"
	"errors"

	"golang.org/x/oauth2"
)

type Userinfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

var (
	ErrMissingEmail    = errors.New("provider returned no email")
	ErrUnverifiedEmail = errors.New("provider email is not verified")
)

type Options struct {
	AllowUnverifiedEmail bool
}

type Option func(*Options)

func WithAllowUnverifiedEmail(allowed bool) Option {
	return func(o *Options) {
		o.AllowUnverifiedEmail = allowed
	}
}

func newOptions(opts []Option) Options {
	var o Options
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

func (o Options) checkEmail(email string, verified bool) error {
	if email == "" {
		return ErrMissingEmail
	}
	if !verified && !o.AllowUnverifiedEmail {
		return ErrUnverifiedEmail
	}
	return nil
}

func revocableToken(token *oauth2.Token) string {
	if token.RefreshToken != "" {
		return token.RefreshToken
	}
	return token.AccessToken
}

type Provider interface {
	GetUser(ctx context.Context, token *oauth2.Token) (*Userinfo, error)
	Revoke(ctx context.Context, token *oauth2.Token) error
}

const (
	DiscordName = "discord"
	GitHubName  = "github"
	GoogleName  = "google"
	OIDCName    = "oidc"
)

func IsBuiltin(name string) bool {
	switch name {
	case DiscordName, GitHubName, GoogleName:
		return true
	}
	return false
}

func New(name string, config *oauth2.Config, userinfoURL, revokeURL string, opts ...Option) Provider {
	switch name {
	case DiscordName:
		return NewDiscordProvider(config, opts...)
	case GitHubName:
		return NewGitHubProvider(config, opts...)
	case GoogleName:
		return NewGoogleProvider(config, opts...)
	default:
		return NewGenericProvider(config, userinfoURL, revokeURL, opts...)
	}
}
