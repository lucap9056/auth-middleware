package providers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newDiscordServer(t *testing.T, userHandler, revokeHandler http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	if userHandler != nil {
		mux.HandleFunc("/api/users/@me", userHandler)
	}
	if revokeHandler != nil {
		mux.HandleFunc("/api/oauth2/token/revoke", revokeHandler)
	}
	return httptest.NewServer(mux)
}

func TestDiscordProvider_GetUser_Success(t *testing.T) {
	server := newDiscordServer(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(DiscordUser{
			ID:       "123456789",
			Username: "testuser",
			Email:    "test@discord.com",
		})
	}, nil)
	defer server.Close()

	provider := NewDiscordProvider(newTestConfig(server.URL))
	user, err := provider.GetUser(testContextWithClient(server.URL), newTestToken())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != "123456789" {
		t.Errorf("ID: got %q, want %q", user.ID, "123456789")
	}
	if user.Email != "test@discord.com" {
		t.Errorf("Email: got %q, want %q", user.Email, "test@discord.com")
	}
	if user.Name != "testuser" {
		t.Errorf("Name: got %q, want %q", user.Name, "testuser")
	}
}

func TestDiscordProvider_GetUser_NonOKStatus(t *testing.T) {
	server := newDiscordServer(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}, nil)
	defer server.Close()

	provider := NewDiscordProvider(newTestConfig(server.URL))
	_, err := provider.GetUser(testContextWithClient(server.URL), newTestToken())

	if err == nil {
		t.Fatal("expected error for non-OK status, got nil")
	}
}

func TestDiscordProvider_GetUser_InvalidJSON(t *testing.T) {
	server := newDiscordServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not valid json"))
	}, nil)
	defer server.Close()

	provider := NewDiscordProvider(newTestConfig(server.URL))
	_, err := provider.GetUser(testContextWithClient(server.URL), newTestToken())

	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestDiscordProvider_Revoke_Success(t *testing.T) {
	server := newDiscordServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		if r.FormValue("token") == "" {
			http.Error(w, "missing token", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	provider := NewDiscordProvider(newTestConfig(server.URL))
	provider.httpClient = newTestClient(server.URL)

	if err := provider.Revoke(context.Background(), newTestToken()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDiscordProvider_Revoke_NonOKStatus(t *testing.T) {
	server := newDiscordServer(t, nil, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	})
	defer server.Close()

	provider := NewDiscordProvider(newTestConfig(server.URL))
	provider.httpClient = newTestClient(server.URL)

	if err := provider.Revoke(context.Background(), newTestToken()); err == nil {
		t.Fatal("expected error for non-OK status, got nil")
	}
}
