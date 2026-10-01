package providers

import (
	"context"

	"golang.org/x/oauth2"
)

type Userinfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
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

func New(name string, config *oauth2.Config, userinfoURL, revokeURL string) Provider {
	switch name {
	case DiscordName:
		return NewDiscordProvider(config)
	case GitHubName:
		return NewGitHubProvider(config)
	case GoogleName:
		return NewGoogleProvider(config)
	default:
		return NewGenericProvider(config, userinfoURL, revokeURL)
	}
}
