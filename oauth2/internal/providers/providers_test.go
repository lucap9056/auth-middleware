package providers

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/oauth2"
)

// redirectTransport rewrites every request's host to the test server,
// preserving the original path. This lets us intercept hardcoded URLs
// (discord.com, googleapis.com) without changing production code.
type redirectTransport struct {
	serverURL string
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	target, _ := url.Parse(t.serverURL)
	r.URL.Scheme = target.Scheme
	r.URL.Host = target.Host
	return http.DefaultTransport.RoundTrip(r)
}

func newTestClient(serverURL string) *http.Client {
	return &http.Client{Transport: &redirectTransport{serverURL: serverURL}}
}

// testContextWithClient injects a redirect client via oauth2.HTTPClient so
// that config.Client(ctx, token) also routes through the test server.
func testContextWithClient(serverURL string) context.Context {
	return context.WithValue(context.Background(), oauth2.HTTPClient, newTestClient(serverURL))
}

func newTestToken() *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		Expiry:       time.Now().Add(time.Hour),
	}
}

func newTestConfig(serverURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  serverURL + "/auth",
			TokenURL: serverURL + "/token",
		},
	}
}
