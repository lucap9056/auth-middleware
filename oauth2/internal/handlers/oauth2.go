package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"

	"github.com/lucap9056/auth-middleware/oauth2/internal/providers"
	"golang.org/x/oauth2"
)

const (
	ProviderDiscordName = "discord"
	ProviderGoogleName  = "google"
)

type OAuth2Handler struct {
	config      *oauth2.Config
	userInfoURL string
	provider    providers.Provider
}

func NewOAuth2Handler(providerName, clientID, clientSecret, redirectURL, authURL, tokenURL string, scopes []string, userInfoURL string, revokeURL string) *OAuth2Handler {
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
	case ProviderGoogleName:
		p = providers.NewGoogleProvider(providers.WithGoogle(config))
	default:
		p = providers.NewGenericProvider(config, userInfoURL, revokeURL)
	}

	return &OAuth2Handler{
		config:      config,
		userInfoURL: userInfoURL,
		provider:    p,
	}
}

func generatePKCE() (verifier string, challenge string) {
	b := make([]byte, 32)
	rand.Read(b)
	verifier = base64.RawURLEncoding.EncodeToString(b)

	h := sha256.New()
	h.Write([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	return verifier, challenge
}

func (h *OAuth2Handler) AuthURL(state string, challenge string) string {
	return h.config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.S256ChallengeOption(challenge))
}

func (h *OAuth2Handler) Exchange(ctx context.Context, code string, verifier string) (*oauth2.Token, error) {
	return h.config.Exchange(ctx, code, oauth2.VerifierOption(verifier))
}

func (h *OAuth2Handler) GetUser(ctx context.Context, token *oauth2.Token) (*providers.UserInfo, error) {
	return h.provider.GetUser(ctx, token)
}

func (h *OAuth2Handler) Revoke(ctx context.Context, token *oauth2.Token) error {
	return h.provider.Revoke(ctx, token)
}
