package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newGitHubServer(t *testing.T, userHandler, emailsHandler, revokeHandler http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	if userHandler != nil {
		mux.HandleFunc("GET /user", userHandler)
	}
	if emailsHandler != nil {
		mux.HandleFunc("GET /user/emails", emailsHandler)
	}
	if revokeHandler != nil {
		mux.HandleFunc("DELETE /applications/test-client-id/token", revokeHandler)
	}
	return httptest.NewServer(mux)
}

func TestGitHubProvider_GetUser_PublicEmail(t *testing.T) {
	srv := newGitHubServer(t, func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(GitHubUser{
			ID:    12345,
			Login: "testuser",
			Name:  "Test User",
			Email: "test@example.com",
		})
	}, nil, nil)
	defer srv.Close()

	provider := NewGitHubProvider(newTestConfig(srv.URL))
	user, err := provider.GetUser(testContextWithClient(srv.URL), newTestToken())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != "12345" {
		t.Errorf("ID: got %q, want %q", user.ID, "12345")
	}
	if user.Email != "test@example.com" {
		t.Errorf("Email: got %q, want %q", user.Email, "test@example.com")
	}
	if user.Name != "Test User" {
		t.Errorf("Name: got %q, want %q", user.Name, "Test User")
	}
}

func TestGitHubProvider_GetUser_PrivateEmailFallback(t *testing.T) {
	srv := newGitHubServer(t,
		func(w http.ResponseWriter, _ *http.Request) {
			json.NewEncoder(w).Encode(GitHubUser{
				ID:    99999,
				Login: "privateuser",
				Name:  "Private User",
				Email: "",
			})
		},
		func(w http.ResponseWriter, _ *http.Request) {
			json.NewEncoder(w).Encode([]GitHubEmail{
				{Email: "secondary@example.com", Primary: false, Verified: true},
				{Email: "private@example.com", Primary: true, Verified: true},
			})
		},
		nil,
	)
	defer srv.Close()

	provider := NewGitHubProvider(newTestConfig(srv.URL))
	user, err := provider.GetUser(testContextWithClient(srv.URL), newTestToken())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Email != "private@example.com" {
		t.Errorf("Email: got %q, want primary email %q", user.Email, "private@example.com")
	}
}

func TestGitHubProvider_GetUser_EmptyNameFallsBackToLogin(t *testing.T) {
	srv := newGitHubServer(t, func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(GitHubUser{
			ID:    77777,
			Login: "loginonly",
			Name:  "",
			Email: "login@example.com",
		})
	}, nil, nil)
	defer srv.Close()

	provider := NewGitHubProvider(newTestConfig(srv.URL))
	user, err := provider.GetUser(testContextWithClient(srv.URL), newTestToken())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Name != "loginonly" {
		t.Errorf("Name: got %q, want login as fallback %q", user.Name, "loginonly")
	}
}

func TestGitHubProvider_GetUser_NoVerifiedPrimaryEmail(t *testing.T) {
	srv := newGitHubServer(t,
		func(w http.ResponseWriter, _ *http.Request) {
			json.NewEncoder(w).Encode(GitHubUser{ID: 1, Login: "u", Email: ""})
		},
		func(w http.ResponseWriter, _ *http.Request) {
			json.NewEncoder(w).Encode([]GitHubEmail{
				{Email: "unverified@example.com", Primary: true, Verified: false},
			})
		},
		nil,
	)
	defer srv.Close()

	provider := NewGitHubProvider(newTestConfig(srv.URL))
	_, err := provider.GetUser(testContextWithClient(srv.URL), newTestToken())
	if err == nil {
		t.Fatal("expected error when no verified primary email found, got nil")
	}
}

func TestGitHubProvider_GetUser_UnverifiedPrimaryEmailAllowed(t *testing.T) {
	srv := newGitHubServer(t,
		func(w http.ResponseWriter, _ *http.Request) {
			json.NewEncoder(w).Encode(GitHubUser{ID: 1, Login: "u", Email: ""})
		},
		func(w http.ResponseWriter, _ *http.Request) {
			json.NewEncoder(w).Encode([]GitHubEmail{
				{Email: "unverified@example.com", Primary: true, Verified: false},
			})
		},
		nil,
	)
	defer srv.Close()

	provider := NewGitHubProvider(newTestConfig(srv.URL), WithAllowUnverifiedEmail(true))
	user, err := provider.GetUser(testContextWithClient(srv.URL), newTestToken())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Email != "unverified@example.com" {
		t.Errorf("Email: got %q, want %q", user.Email, "unverified@example.com")
	}
}

func TestGitHubProvider_GetUser_NonOKStatus(t *testing.T) {
	srv := newGitHubServer(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}, nil, nil)
	defer srv.Close()

	provider := NewGitHubProvider(newTestConfig(srv.URL))
	_, err := provider.GetUser(testContextWithClient(srv.URL), newTestToken())
	if err == nil {
		t.Fatal("expected error for non-OK status, got nil")
	}
}

func TestGitHubProvider_Revoke_Success(t *testing.T) {
	var receivedToken string
	srv := newGitHubServer(t, nil, nil, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		receivedToken = body["access_token"]
		w.WriteHeader(http.StatusNoContent)
	})
	defer srv.Close()

	provider := NewGitHubProvider(newTestConfig(srv.URL))
	provider.httpClient = newTestClient(srv.URL)

	tok := newTestToken()
	if err := provider.Revoke(context.Background(), tok); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedToken != tok.AccessToken {
		t.Errorf("access_token sent: got %q, want %q", receivedToken, tok.AccessToken)
	}
}

func TestGitHubProvider_Revoke_NonOKStatus(t *testing.T) {
	srv := newGitHubServer(t, nil, nil, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	})
	defer srv.Close()

	provider := NewGitHubProvider(newTestConfig(srv.URL))
	provider.httpClient = newTestClient(srv.URL)

	if err := provider.Revoke(context.Background(), newTestToken()); err == nil {
		t.Fatal("expected error for non-204 status, got nil")
	}
}
