package providers

import (
	"context"

	"golang.org/x/oauth2"
)

type UserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type Provider interface {
	GetUser(ctx context.Context, token *oauth2.Token) (*UserInfo, error)
	Revoke(ctx context.Context, token *oauth2.Token) error
}
