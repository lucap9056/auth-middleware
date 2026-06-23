package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newGoogleServer(t *testing.T, userHandler, revokeHandler http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	if userHandler != nil {
		mux.HandleFunc("/oauth2/v2/userinfo", userHandler)
	}
	if revokeHandler != nil {
		mux.HandleFunc("/revoke", revokeHandler)
	}
	return httptest.NewServer(mux)
}

func TestGoogleProvider_GetUser_Success(t *testing.T) {
	server := newGoogleServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(GoogleUser{
			ID:    "google-uid-001",
			Email: "test@gmail.com",
			Name:  "Test User",
		})
	}, nil)
	defer server.Close()

	provider := NewGoogleProvider(newTestConfig(server.URL))
	user, err := provider.GetUser(testContextWithClient(server.URL), newTestToken())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != "google-uid-001" {
		t.Errorf("ID: got %q, want %q", user.ID, "google-uid-001")
	}
	if user.Email != "test@gmail.com" {
		t.Errorf("Email: got %q, want %q", user.Email, "test@gmail.com")
	}
	if user.Name != "Test User" {
		t.Errorf("Name: got %q, want %q", user.Name, "Test User")
	}
}

func TestGoogleProvider_GetUser_NonOKStatus(t *testing.T) {
	server := newGoogleServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}, nil)
	defer server.Close()

	provider := NewGoogleProvider(newTestConfig(server.URL))
	_, err := provider.GetUser(testContextWithClient(server.URL), newTestToken())

	if err == nil {
		t.Fatal("expected error for non-OK status, got nil")
	}
}

func TestGoogleProvider_GetUser_InvalidJSON(t *testing.T) {
	server := newGoogleServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("{invalid"))
	}, nil)
	defer server.Close()

	provider := NewGoogleProvider(newTestConfig(server.URL))
	_, err := provider.GetUser(testContextWithClient(server.URL), newTestToken())

	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestGoogleProvider_Revoke_Success(t *testing.T) {
	server := newGoogleServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil || r.FormValue("token") == "" {
			http.Error(w, "missing token", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	provider := NewGoogleProvider(newTestConfig(server.URL))
	provider.httpClient = newTestClient(server.URL)

	if err := provider.Revoke(context.Background(), newTestToken()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGoogleProvider_Revoke_UsesRefreshToken(t *testing.T) {
	var receivedToken string
	server := newGoogleServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		receivedToken = r.FormValue("token")
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	provider := NewGoogleProvider(newTestConfig(server.URL))
	provider.httpClient = newTestClient(server.URL)

	tok := newTestToken()
	provider.Revoke(context.Background(), tok)

	if receivedToken != tok.RefreshToken {
		t.Errorf("expected refresh token %q to be sent, got %q", tok.RefreshToken, receivedToken)
	}
}

func TestGoogleProvider_Revoke_NonOKStatus(t *testing.T) {
	server := newGoogleServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	})
	defer server.Close()

	provider := NewGoogleProvider(newTestConfig(server.URL))
	provider.httpClient = newTestClient(server.URL)

	if err := provider.Revoke(context.Background(), newTestToken()); err == nil {
		t.Fatal("expected error for non-OK status, got nil")
	}
}
