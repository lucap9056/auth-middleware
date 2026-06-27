package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/ravener/discord-oauth2"
	"golang.org/x/oauth2"
)

type DiscordUser struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	Avatar        string `json:"avatar"`
	Bot           bool   `json:"bot"`
	System        bool   `json:"system"`
	MFAEnabled    bool   `json:"mfa_enabled"`
	Locale        string `json:"locale"`
	Verified      bool   `json:"verified"`
	Email         string `json:"email"`
	Flags         int    `json:"flags"`
	PremiumType   int    `json:"premium_type"`
	PublicFlags   int    `json:"public_flags"`
}

func WithDiscord(cfg *oauth2.Config) *oauth2.Config {
	cfg.Scopes = []string{discord.ScopeIdentify, discord.ScopeEmail}
	cfg.Endpoint = discord.Endpoint
	return cfg
}

type DiscordProvider struct {
	config     *oauth2.Config
	httpClient *http.Client
}

func NewDiscordProvider(config *oauth2.Config) *DiscordProvider {
	return &DiscordProvider{config: config, httpClient: http.DefaultClient}
}

func (p *DiscordProvider) GetUser(ctx context.Context, token *oauth2.Token) (*Userinfo, error) {
	client := p.config.Client(ctx, token)
	resp, err := client.Get("https://discord.com/api/users/@me")
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("discord api returned status: %s", resp.Status)
	}

	var user DiscordUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &Userinfo{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Username,
	}, nil
}

func (p *DiscordProvider) Revoke(ctx context.Context, token *oauth2.Token) error {
	data := url.Values{}
	data.Set("client_id", p.config.ClientID)
	data.Set("client_secret", p.config.ClientSecret)
	data.Set("token", token.AccessToken)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://discord.com/api/oauth2/token/revoke", strings.NewReader(data.Encode()))
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
		return fmt.Errorf("discord api returned status: %s", resp.Status)
	}
	return nil
}
