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
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	GivenName     string `json:"given_name"`
	FamilyName    string `json:"family_name"`
	Picture       string `json:"picture"`
	Locale        string `json:"locale"`
}

type GoogleProvider struct {
	config     *oauth2.Config
	options    Options
	httpClient *http.Client
}

func NewGoogleProvider(config *oauth2.Config, opts ...Option) *GoogleProvider {
	config.Scopes = []string{
		"openid",
		"email",
		"profile",
	}
	config.Endpoint = google.Endpoint
	return &GoogleProvider{config: config, options: newOptions(opts), httpClient: http.DefaultClient}
}

func (p *GoogleProvider) GetUser(ctx context.Context, token *oauth2.Token) (*Userinfo, error) {
	client := p.config.Client(ctx, token)

	resp, err := client.Get("https://openidconnect.googleapis.com/v1/userinfo")
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

	if err := p.options.checkEmail(user.Email, user.EmailVerified); err != nil {
		return nil, err
	}

	return &Userinfo{
		ID:    user.Sub,
		Email: user.Email,
		Name:  user.Name,
	}, nil
}

func (p *GoogleProvider) Revoke(ctx context.Context, token *oauth2.Token) error {
	const revokeURL = "https://oauth2.googleapis.com/revoke"
	data := url.Values{}
	data.Set("token", revocableToken(token))

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
