package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

type GitHubUser struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type GitHubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func WithGitHub(cfg *oauth2.Config) *oauth2.Config {
	cfg.Scopes = []string{"read:user", "user:email"}
	cfg.Endpoint = github.Endpoint
	return cfg
}

type GitHubProvider struct {
	config     *oauth2.Config
	httpClient *http.Client
}

func NewGitHubProvider(config *oauth2.Config) *GitHubProvider {
	return &GitHubProvider{config: config, httpClient: http.DefaultClient}
}

func (p *GitHubProvider) GetUser(ctx context.Context, token *oauth2.Token) (*Userinfo, error) {
	client := p.config.Client(ctx, token)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create user request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status: %s", resp.Status)
	}

	var user GitHubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode GitHub user: %w", err)
	}

	email := user.Email
	if email == "" {
		email, err = p.fetchPrimaryEmail(ctx, client)
		if err != nil {
			return nil, err
		}
	}

	name := user.Name
	if name == "" {
		name = user.Login
	}

	return &Userinfo{
		ID:    strconv.FormatInt(user.ID, 10),
		Email: email,
		Name:  name,
	}, nil
}

// fetchPrimaryEmail calls /user/emails to find the verified primary email,
// used when the user's email is set to private on GitHub.
func (p *GitHubProvider) fetchPrimaryEmail(ctx context.Context, client *http.Client) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create emails request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get GitHub emails: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub emails API returned status: %s", resp.Status)
	}

	var emails []GitHubEmail
	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", fmt.Errorf("failed to decode GitHub emails: %w", err)
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}

	return "", fmt.Errorf("no verified primary email found on GitHub account")
}

// Revoke uses GitHub's DELETE /applications/{client_id}/token endpoint
// with Basic Auth, which differs from the standard form-post revocation.
func (p *GitHubProvider) Revoke(ctx context.Context, token *oauth2.Token) error {
	revokeURL := "https://api.github.com/applications/" + p.config.ClientID + "/token"

	body, err := json.Marshal(map[string]string{"access_token": token.AccessToken})
	if err != nil {
		return fmt.Errorf("failed to marshal revoke body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, revokeURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create revoke request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(p.config.ClientID, p.config.ClientSecret)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send revoke request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("GitHub revoke returned status: %d", resp.StatusCode)
	}

	return nil
}
