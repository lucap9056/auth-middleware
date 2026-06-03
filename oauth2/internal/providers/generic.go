package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
)

type GenericUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type GenericProvider struct {
	config      *oauth2.Config
	userInfoURL string
	revokeURL   string
}

func NewGenericProvider(config *oauth2.Config, userInfoURL string, revokeURL string) *GenericProvider {
	return &GenericProvider{config: config, userInfoURL: userInfoURL, revokeURL: revokeURL}
}

func (p *GenericProvider) GetUser(ctx context.Context, token *oauth2.Token) (*UserInfo, error) {
	client := p.config.Client(ctx, token)
	resp, err := client.Get(p.userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("provider api returned status: %s", resp.Status)
	}

	var user GenericUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &UserInfo{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
	}, nil
}

func (p *GenericProvider) Revoke(ctx context.Context, token *oauth2.Token) error {
	if p.revokeURL == "" {
		return fmt.Errorf("revocation URL is not configured for this provider")
	}

	tokenToRevoke := token.AccessToken
	if token.RefreshToken != "" {
		tokenToRevoke = token.RefreshToken
	}

	data := url.Values{}
	data.Set("token", tokenToRevoke)
	data.Set("client_id", p.config.ClientID)
	data.Set("client_secret", p.config.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, "POST", p.revokeURL, strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create revoke request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send revoke request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("revoke failed with status: %d", resp.StatusCode)
	}

	return nil
}
