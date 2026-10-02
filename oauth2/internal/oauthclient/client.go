package oauthclient

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"slices"
	"time"

	"github.com/lucap9056/auth-middleware/oauth2/internal/providers"
	"golang.org/x/oauth2"
)

type Config struct {
	Provider     string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	AuthURL      string
	TokenURL     string
	UserinfoURL  string
	RevokeURL    string
	Scopes       []string
}

type OIDCConfig struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

type Client struct {
	config        *oauth2.Config
	provider      providers.Provider
	issuer        string
	requireIssuer bool
}

func New(cfg Config, opts ...providers.Option) *Client {
	config := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       cfg.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  cfg.AuthURL,
			TokenURL: cfg.TokenURL,
		},
	}

	provider := providers.New(cfg.Provider, config, cfg.UserinfoURL, cfg.RevokeURL, opts...)

	warmTokenEndpoint(config.Endpoint.TokenURL)

	return &Client{
		config:   config,
		provider: provider,
	}
}

// NewOIDC builds a Client using OIDC discovery. It fetches the provider's
// well-known configuration document and auto-populates all endpoints.
// The "openid" scope is automatically added if not already present.
func NewOIDC(ctx context.Context, cfg OIDCConfig, opts ...providers.Option) (*Client, error) {
	discovery, err := providers.FetchDiscovery(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, err
	}

	scopes := cfg.Scopes
	if !slices.Contains(scopes, "openid") {
		scopes = append([]string{"openid"}, scopes...)
	}

	config := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  discovery.AuthorizationEndpoint,
			TokenURL: discovery.TokenEndpoint,
		},
	}

	warmTokenEndpoint(discovery.TokenEndpoint)

	return &Client{
		config:        config,
		provider:      providers.NewOIDCProvider(config, discovery, opts...),
		issuer:        discovery.Issuer,
		requireIssuer: discovery.AuthorizationResponseIssParameterSupported,
	}, nil
}

func (c *Client) AuthURL(state string, verifier string) string {
	return c.config.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(verifier),
	)
}

func (c *Client) ValidateIssuer(iss string) error {
	if iss == "" {
		if c.requireIssuer {
			return fmt.Errorf("authorization response missing iss parameter")
		}
		return nil
	}
	if c.issuer != "" && iss != c.issuer {
		return fmt.Errorf("authorization response iss mismatch: got %q, want %q", iss, c.issuer)
	}
	return nil
}

func (c *Client) Exchange(ctx context.Context, code string, verifier string) (*oauth2.Token, error) {
	return c.config.Exchange(ctx, code, oauth2.VerifierOption(verifier))
}

func (c *Client) GetUser(ctx context.Context, token *oauth2.Token) (*providers.Userinfo, error) {
	return c.provider.GetUser(ctx, token)
}

func (c *Client) Revoke(ctx context.Context, token *oauth2.Token) error {
	return c.provider.Revoke(ctx, token)
}

func warmTokenEndpoint(tokenURL string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL, nil)
		if err != nil {
			return
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("[WARN] token endpoint warm-up failed: %v", err)
			return
		}
		resp.Body.Close()
	}()
}
