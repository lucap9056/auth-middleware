package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newGenericServer(t *testing.T, userHandler, revokeHandler http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	if userHandler != nil {
		mux.HandleFunc("/userinfo", userHandler)
	}
	if revokeHandler != nil {
		mux.HandleFunc("/revoke", revokeHandler)
	}
	return httptest.NewServer(mux)
}

func TestGenericProvider_GetUser_Success(t *testing.T) {
	server := newGenericServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(GenericUser{
			ID:    "generic-001",
			Email: "user@example.com",
			Name:  "Generic User",
		})
	}, nil)
	defer server.Close()

	provider := NewGenericProvider(newTestConfig(server.URL), server.URL+"/userinfo", "")
	user, err := provider.GetUser(testContextWithClient(server.URL), newTestToken())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != "generic-001" {
		t.Errorf("ID: got %q, want %q", user.ID, "generic-001")
	}
	if user.Email != "user@example.com" {
		t.Errorf("Email: got %q, want %q", user.Email, "user@example.com")
	}
	if user.Name != "Generic User" {
		t.Errorf("Name: got %q, want %q", user.Name, "Generic User")
	}
}

func TestGenericProvider_GetUser_NonOKStatus(t *testing.T) {
	server := newGenericServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}, nil)
	defer server.Close()

	provider := NewGenericProvider(newTestConfig(server.URL), server.URL+"/userinfo", "")
	_, err := provider.GetUser(testContextWithClient(server.URL), newTestToken())

	if err == nil {
		t.Fatal("expected error for non-OK status, got nil")
	}
}

func TestGenericProvider_Revoke_Success(t *testing.T) {
	server := newGenericServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	provider := NewGenericProvider(newTestConfig(server.URL), server.URL+"/userinfo", server.URL+"/revoke")
	provider.httpClient = newTestClient(server.URL)

	if err := provider.Revoke(context.Background(), newTestToken()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGenericProvider_Revoke_NoURL(t *testing.T) {
	provider := NewGenericProvider(newTestConfig("http://localhost"), "/userinfo", "")

	err := provider.Revoke(context.Background(), newTestToken())
	if err == nil {
		t.Fatal("expected error when revokeURL is empty, got nil")
	}
}

func TestGenericProvider_Revoke_PrefersRefreshToken(t *testing.T) {
	var receivedToken string
	server := newGenericServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		receivedToken = r.FormValue("token")
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	provider := NewGenericProvider(newTestConfig(server.URL), server.URL+"/userinfo", server.URL+"/revoke")
	provider.httpClient = newTestClient(server.URL)

	tok := newTestToken()
	provider.Revoke(context.Background(), tok)

	if receivedToken != tok.RefreshToken {
		t.Errorf("expected refresh token %q to be sent, got %q", tok.RefreshToken, receivedToken)
	}
}

func TestGenericProvider_Revoke_FallbackToAccessToken(t *testing.T) {
	var receivedToken string
	server := newGenericServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		receivedToken = r.FormValue("token")
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	provider := NewGenericProvider(newTestConfig(server.URL), server.URL+"/userinfo", server.URL+"/revoke")
	provider.httpClient = newTestClient(server.URL)

	tok := newTestToken()
	tok.RefreshToken = ""
	provider.Revoke(context.Background(), tok)

	if receivedToken != tok.AccessToken {
		t.Errorf("expected access token %q to be sent, got %q", tok.AccessToken, receivedToken)
	}
}
