package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type GoogleUser struct {
	ID            string `json:"id"`
	Email         string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

func WithGoogle(cfg *oauth2.Config) *oauth2.Config {
	cfg.Scopes = []string{
		"https://www.googleapis.com/auth/userinfo.email",
		"https://www.googleapis.com/auth/userinfo.profile",
	}
	cfg.Endpoint = google.Endpoint
	return cfg
}

type GoogleProvider struct {
	config     *oauth2.Config
	httpClient *http.Client
}

func NewGoogleProvider(config *oauth2.Config) *GoogleProvider {
	return &GoogleProvider{config: config, httpClient: http.DefaultClient}
}

func (p *GoogleProvider) GetUser(ctx context.Context, token *oauth2.Token) (*Userinfo, error) {
	client := p.config.Client(ctx, token)

	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google api returned status: %s", resp.Status)
	}

	var user GoogleUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &Userinfo{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
	}, nil
}

func (p *GoogleProvider) Revoke(ctx context.Context, token *oauth2.Token) error {
	const revokeURL = "https://oauth2.googleapis.com/revoke"
	data := url.Values{}
	data.Set("token", token.RefreshToken)

	req, err := http.NewRequestWithContext(ctx, "POST", revokeURL, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("google revoke failed with status: %d", resp.StatusCode)
	}

	return nil
}
