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
