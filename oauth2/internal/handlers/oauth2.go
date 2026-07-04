package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"slices"
	"time"

	"github.com/lucap9056/auth-middleware/oauth2/internal/providers"
	"golang.org/x/oauth2"
)

const (
	ProviderDiscordName = "discord"
	ProviderGitHubName  = "github"
	ProviderGoogleName  = "google"
	ProviderOIDCName    = "oidc"
)

type OAuth2Handler struct {
	config      *oauth2.Config
	userinfoURL string
	provider    providers.Provider
}

func NewOAuth2Handler(providerName, clientID, clientSecret, redirectURL, authURL, tokenURL string, scopes []string, userinfoURL string, revokeURL string) *OAuth2Handler {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  authURL,
			TokenURL: tokenURL,
		},
	}

	var p providers.Provider
	switch providerName {
	case ProviderDiscordName:
		p = providers.NewDiscordProvider(providers.WithDiscord(config))
	case ProviderGitHubName:
		p = providers.NewGitHubProvider(providers.WithGitHub(config))
	case ProviderGoogleName:
		p = providers.NewGoogleProvider(providers.WithGoogle(config))
	default:
		p = providers.NewGenericProvider(config, userinfoURL, revokeURL)
	}

	warmTokenEndpoint(tokenURL)

	return &OAuth2Handler{
		config:      config,
		userinfoURL: userinfoURL,
		provider:    p,
	}
}

// NewOIDCHandler builds an OAuth2Handler using OIDC discovery. It fetches the
// provider's well-known configuration document and auto-populates all endpoints.
// The "openid" scope is automatically added if not already present.
func NewOIDCHandler(ctx context.Context, issuerURL, clientID, clientSecret, redirectURL string, scopes []string) (*OAuth2Handler, error) {
	discovery, err := providers.FetchDiscovery(ctx, issuerURL)
	if err != nil {
		return nil, err
	}

	if !slices.Contains(scopes, "openid") {
		scopes = append([]string{"openid"}, scopes...)
	}

	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:  discovery.AuthorizationEndpoint,
			TokenURL: discovery.TokenEndpoint,
		},
	}

	warmTokenEndpoint(discovery.TokenEndpoint)

	return &OAuth2Handler{
		config:      config,
		userinfoURL: discovery.UserinfoEndpoint,
		provider:    providers.NewOIDCProvider(config, discovery),
	}, nil
}

func generateState() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (h *OAuth2Handler) AuthURL(state string, verifier string) string {
	return h.config.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(verifier),
	)
}

func (h *OAuth2Handler) Exchange(ctx context.Context, code string, verifier string) (*oauth2.Token, error) {
	return h.config.Exchange(ctx, code, oauth2.VerifierOption(verifier))
}

func (h *OAuth2Handler) GetUser(ctx context.Context, token *oauth2.Token) (*providers.Userinfo, error) {
	return h.provider.GetUser(ctx, token)
}

func (h *OAuth2Handler) Revoke(ctx context.Context, token *oauth2.Token) error {
	return h.provider.Revoke(ctx, token)
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
