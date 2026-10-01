package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/lucap9056/auth-middleware/database"
	"github.com/lucap9056/auth-middleware/oauth2/internal/handlers/options"
)

func TestHealth(t *testing.T) {
	stub := newOAuthStub("", "", "")
	defer stub.Close()

	env := newTestEnv(stub, newMockDB())
	w := env.do(httptest.NewRequest(http.MethodGet, "/health", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
}

func TestLogin(t *testing.T) {
	stub := newOAuthStub("", "", "")
	defer stub.Close()

	env := newTestEnv(stub, newMockDB())
	w := env.do(httptest.NewRequest(http.MethodGet, "/login", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}

	var resp struct {
		Success bool `json:"success"`
		Message struct {
			URL string `json:"url"`
		} `json:"message"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Success {
		t.Fatal("want success=true")
	}
	if !strings.Contains(resp.Message.URL, stub.URL) {
		t.Errorf("URL %q should point to stub server %s", resp.Message.URL, stub.URL)
	}
	u, err := url.Parse(resp.Message.URL)
	if err != nil {
		t.Fatalf("parse URL: %v", err)
	}
	if u.Query().Get("state") == "" {
		t.Error("URL should contain a non-empty state parameter")
	}
	if u.Query().Get("code_challenge") == "" {
		t.Error("URL should contain a non-empty code_challenge parameter")
	}
}

// loginState calls /login and extracts the state parameter from the returned URL.
func loginState(t *testing.T, env *testEnv) string {
	t.Helper()
	w := env.do(httptest.NewRequest(http.MethodGet, "/login", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("login failed: %d %s", w.Code, w.Body.String())
	}
	var resp struct {
		Message struct {
			URL string `json:"url"`
		} `json:"message"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	u, err := url.Parse(resp.Message.URL)
	if err != nil {
		t.Fatalf("parse login URL: %v", err)
	}
	s := u.Query().Get("state")
	if s == "" {
		t.Fatal("login URL missing state parameter")
	}
	return s
}

// TestCallback_ExistingUser verifies the full OAuth2 callback for a known user:
// code exchange → userinfo fetch → DB lookup → JWT issuance → cookie set.
func TestCallback_ExistingUser(t *testing.T) {
	const (
		userID   = "uid-existing"
		email    = "existing@example.com"
		username = "existinguser"
	)

	stub := newOAuthStub("oauth-uid", email, username)
	defer stub.Close()

	db := newMockDB()
	db.seedUser(&database.User{UserID: userID, Username: username, Email: email})
	env := newTestEnv(stub, db)

	stateVal := loginState(t, env)
	req := httptest.NewRequest(http.MethodGet, "/callback?code=testcode&state="+stateVal, nil)
	w := env.do(req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Message struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"message"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Success {
		t.Fatal("want success=true")
	}
	if resp.Message.AccessToken == "" || resp.Message.RefreshToken == "" {
		t.Error("want non-empty access and refresh tokens")
	}

	var cookieSet bool
	for _, c := range w.Result().Cookies() {
		if c.Name == "refresh_token" && c.Value == resp.Message.RefreshToken {
			cookieSet = true
		}
	}
	if !cookieSet {
		t.Error("refresh_token cookie not set or value mismatch")
	}
}

func TestCallback_RegistrationDisabled(t *testing.T) {
	stub := newOAuthStub("u1", "new@example.com", "New User")
	defer stub.Close()

	env := newTestEnv(stub, newMockDB()) // AllowRegistration defaults to false

	stateVal := loginState(t, env)
	req := httptest.NewRequest(http.MethodGet, "/callback?code=testcode&state="+stateVal, nil)
	w := env.do(req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCallback_RegistrationEnabled(t *testing.T) {
	const email = "newuser@example.com"

	stub := newOAuthStub("u1", email, "New User")
	defer stub.Close()

	db := newMockDB()
	env := newTestEnv(stub, db, options.WithAllowRegistration(true))

	stateVal := loginState(t, env)
	req := httptest.NewRequest(http.MethodGet, "/callback?code=testcode&state="+stateVal, nil)
	w := env.do(req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Success {
		t.Fatal("want success=true")
	}

	created, err := db.GetUserFromEmail(email)
	if err != nil || created == nil {
		t.Errorf("user should have been created in DB: err=%v user=%v", err, created)
	}
}

func TestCallback_MissingState(t *testing.T) {
	stub := newOAuthStub("u1", "a@b.com", "A")
	defer stub.Close()

	env := newTestEnv(stub, newMockDB())

	req := httptest.NewRequest(http.MethodGet, "/callback?code=testcode", nil) // no state
	w := env.do(req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestCallback_InvalidState(t *testing.T) {
	stub := newOAuthStub("u1", "a@b.com", "A")
	defer stub.Close()

	env := newTestEnv(stub, newMockDB())

	req := httptest.NewRequest(http.MethodGet, "/callback?code=testcode&state=unknown-state", nil)
	w := env.do(req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}

func TestCallback_StateReplay(t *testing.T) {
	stub := newOAuthStub("u1", "a@b.com", "A")
	defer stub.Close()

	env := newTestEnv(stub, nil)

	stateVal := loginState(t, env)

	// First use succeeds
	req := httptest.NewRequest(http.MethodGet, "/callback?code=testcode&state="+stateVal, nil)
	w := env.do(req)
	if w.Code != http.StatusOK {
		t.Fatalf("first callback: want 200, got %d: %s", w.Code, w.Body.String())
	}

	// Replay with the same state must fail
	req2 := httptest.NewRequest(http.MethodGet, "/callback?code=testcode&state="+stateVal, nil)
	w2 := env.do(req2)
	if w2.Code != http.StatusBadRequest {
		t.Fatalf("replay callback: want 400, got %d", w2.Code)
	}
}

// TestCallback_NoDB_OAuthTokens verifies that without a DB the raw OAuth tokens
// are returned in the response body.
func TestCallback_NoDB_OAuthTokens(t *testing.T) {
	stub := newOAuthStub("u1", "a@b.com", "A")
	defer stub.Close()

	env := newTestEnv(stub, nil)

	stateVal := loginState(t, env)
	req := httptest.NewRequest(http.MethodGet, "/callback?code=testcode&state="+stateVal, nil)
	w := env.do(req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Message struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"message"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Success {
		t.Fatal("want success=true")
	}
	if resp.Message.AccessToken != "stub-access-token" || resp.Message.RefreshToken != "stub-refresh-token" {
		t.Errorf("expected stub oauth tokens, got access=%q refresh=%q",
			resp.Message.AccessToken, resp.Message.RefreshToken)
	}
}

// TestCallback_NoDB_PassOAuthToken verifies that with PassOAuthToken=true the
// OAuth tokens are forwarded via response headers instead of the body.
func TestCallback_NoDB_PassOAuthToken(t *testing.T) {
	stub := newOAuthStub("u1", "a@b.com", "A")
	defer stub.Close()

	env := newTestEnv(stub, nil, options.WithPassOAuthToken(true))

	stateVal := loginState(t, env)
	req := httptest.NewRequest(http.MethodGet, "/callback?code=testcode&state="+stateVal, nil)
	w := env.do(req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("X-Forwarded-Access-Token"); got != "stub-access-token" {
		t.Errorf("X-Forwarded-Access-Token: want %q, got %q", "stub-access-token", got)
	}
	if got := w.Header().Get("X-Forwarded-Refresh-Token"); got != "stub-refresh-token" {
		t.Errorf("X-Forwarded-Refresh-Token: want %q, got %q", "stub-refresh-token", got)
	}
}

func TestRefresh_WithBody(t *testing.T) {
	const (
		userID   = "uid-1"
		username = "user1"
		email    = "user1@example.com"
	)

	stub := newOAuthStub("", "", "")
	defer stub.Close()

	db := newMockDB()
	db.seedUser(&database.User{UserID: userID, Username: username, Email: email})
	env := newTestEnv(stub, db)

	refresh, _ := env.issueTokens(userID, username, "device")

	body, _ := json.Marshal(map[string]string{"refresh_token": refresh})
	req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := env.do(req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Success bool `json:"success"`
		Message struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"message"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Success {
		t.Fatal("want success=true")
	}
	if resp.Message.AccessToken == "" || resp.Message.RefreshToken == "" {
		t.Error("want non-empty rotated token pair")
	}
	if resp.Message.RefreshToken == refresh {
		t.Error("refresh token should have been rotated")
	}
}

func TestRefresh_WithCookie(t *testing.T) {
	const (
		userID   = "uid-2"
		username = "user2"
		email    = "user2@example.com"
	)

	stub := newOAuthStub("", "", "")
	defer stub.Close()

	db := newMockDB()
	db.seedUser(&database.User{UserID: userID, Username: username, Email: email})
	env := newTestEnv(stub, db)

	refresh, _ := env.issueTokens(userID, username, "device")

	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: refresh})
	w := env.do(req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Success bool `json:"success"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Success {
		t.Fatal("want success=true")
	}
}

func TestRefresh_InvalidToken(t *testing.T) {
	stub := newOAuthStub("", "", "")
	defer stub.Close()

	env := newTestEnv(stub, newMockDB())

	body, _ := json.Marshal(map[string]string{"refresh_token": "not.a.valid.jwt"})
	req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := env.do(req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}

func TestRefreshAccess(t *testing.T) {
	const (
		userID   = "uid-3"
		username = "user3"
		email    = "user3@example.com"
	)

	stub := newOAuthStub("", "", "")
	defer stub.Close()

	db := newMockDB()
	db.seedUser(&database.User{UserID: userID, Username: username, Email: email})
	env := newTestEnv(stub, db)

	refresh, _ := env.issueTokens(userID, username, "device")

	body, _ := json.Marshal(map[string]string{"refresh_token": refresh})
	req := httptest.NewRequest(http.MethodPost, "/refresh-access", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := env.do(req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.Success || resp.Message == "" {
		t.Error("want success=true and a new access token in message")
	}
}

func TestVerify_Valid(t *testing.T) {
	const (
		userID   = "uid-4"
		username = "user4"
	)

	stub := newOAuthStub("", "", "")
	defer stub.Close()

	db := newMockDB()
	db.seedUser(&database.User{UserID: userID, Username: username, Email: "user4@example.com"})
	env := newTestEnv(stub, db)

	_, access := env.issueTokens(userID, username, "device")

	req := httptest.NewRequest(http.MethodGet, "/verify", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w := env.do(req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d: %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("X-Forwarded-User-ID"); got != userID {
		t.Errorf("X-Forwarded-User-ID: want %q, got %q", userID, got)
	}
}

func TestVerify_MissingToken(t *testing.T) {
	stub := newOAuthStub("", "", "")
	defer stub.Close()

	env := newTestEnv(stub, newMockDB())

	w := env.do(httptest.NewRequest(http.MethodGet, "/verify", nil))

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}

func TestVerify_InvalidToken(t *testing.T) {
	stub := newOAuthStub("", "", "")
	defer stub.Close()

	env := newTestEnv(stub, newMockDB())

	req := httptest.NewRequest(http.MethodGet, "/verify", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.value")
	w := env.do(req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", w.Code)
	}
}

func TestLogout(t *testing.T) {
	const (
		userID   = "uid-5"
		username = "user5"
	)

	stub := newOAuthStub("", "", "")
	defer stub.Close()

	db := newMockDB()
	db.seedUser(&database.User{UserID: userID, Username: username, Email: "user5@example.com"})
	env := newTestEnv(stub, db)

	refresh, _ := env.issueTokens(userID, username, "device")

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: refresh})
	w := env.do(req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	var cleared bool
	for _, c := range w.Result().Cookies() {
		if c.Name == "refresh_token" && c.Value == "" {
			cleared = true
		}
	}
	if !cleared {
		t.Error("refresh_token cookie should be cleared after logout")
	}
}

func TestDeleteMe(t *testing.T) {
	const (
		userID   = "uid-6"
		username = "user6"
		email    = "user6@example.com"
	)

	stub := newOAuthStub("", "", "")
	defer stub.Close()

	db := newMockDB()
	db.seedUser(&database.User{UserID: userID, Username: username, Email: email})
	env := newTestEnv(stub, db)

	_, access := env.issueTokens(userID, username, "device")

	req := httptest.NewRequest(http.MethodDelete, "/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w := env.do(req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	deleted, err := db.GetUserFromEmail(email)
	if err != nil || deleted != nil {
		t.Errorf("user should have been deleted from DB: err=%v user=%v", err, deleted)
	}
}
