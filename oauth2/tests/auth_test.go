package integration

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

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
	assertNoStore(t, w)

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
		email    = "existing@example.com"
		username = "existinguser"
	)

	stub := newOAuthStub("oauth-uid", email, username)
	defer stub.Close()

	db := newMockDB()
	db.seedUser(email)
	env := newTestEnv(stub, db)

	stateVal := loginState(t, env)
	req := httptest.NewRequest(http.MethodGet, "/callback?code=testcode&state="+stateVal, nil)
	w := env.do(req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	assertNoStore(t, w)

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

	if !db.hasUser(email) {
		t.Error("user should have been created in DB")
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
	assertNoStore(t, w)

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
	assertNoStore(t, w)
	if got := w.Header().Get("X-Forwarded-Access-Token"); got != "stub-access-token" {
		t.Errorf("X-Forwarded-Access-Token: want %q, got %q", "stub-access-token", got)
	}
	if got := w.Header().Get("X-Forwarded-Refresh-Token"); got != "stub-refresh-token" {
		t.Errorf("X-Forwarded-Refresh-Token: want %q, got %q", "stub-refresh-token", got)
	}
}

func TestRefresh_WithBody(t *testing.T) {
	const email = "user1@example.com"

	stub := newOAuthStub("", "", "")
	defer stub.Close()

	db := newMockDB()
	db.seedUser(email)
	env := newTestEnv(stub, db)

	refresh, _ := env.issueTokens(email, "device")

	body, _ := json.Marshal(map[string]string{"refresh_token": refresh})
	req := httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := env.do(req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	assertNoStore(t, w)

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
	const email = "user2@example.com"

	stub := newOAuthStub("", "", "")
	defer stub.Close()

	db := newMockDB()
	db.seedUser(email)
	env := newTestEnv(stub, db)

	refresh, _ := env.issueTokens(email, "device")

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

func refreshWithCookie(env *testEnv, refreshToken string) (int, string) {
	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})
	w := env.do(req)

	var resp struct {
		Message struct {
			RefreshToken string `json:"refresh_token"`
		} `json:"message"`
	}
	json.NewDecoder(w.Body).Decode(&resp)
	return w.Code, resp.Message.RefreshToken
}

func TestRefresh_ConcurrentSameTokenSharesRotation(t *testing.T) {
	const email = "concurrent@example.com"

	stub := newOAuthStub("", "", "")
	defer stub.Close()

	db := newMockDB()
	db.seedUser(email)
	env := newTestEnv(stub, db)

	refresh, _ := env.issueTokens(email, "device")

	const n = 10
	codes := make([]int, n)
	rotated := make([]string, n)
	var wg sync.WaitGroup
	for i := range n {
		wg.Go(func() {
			codes[i], rotated[i] = refreshWithCookie(env, refresh)
		})
	}
	wg.Wait()

	for i := range n {
		if codes[i] != http.StatusOK {
			t.Fatalf("request %d: want 200, got %d", i, codes[i])
		}
		if rotated[i] != rotated[0] {
			t.Fatalf("request %d received a different rotated token; concurrent refreshes must share one rotation", i)
		}
	}

	if code, _ := refreshWithCookie(env, refresh); code != http.StatusOK {
		t.Errorf("retry with the original token inside the grace window: want 200, got %d", code)
	}

	if code, _ := refreshWithCookie(env, rotated[0]); code != http.StatusOK {
		t.Errorf("rotated token must stay valid after concurrent refreshes: want 200, got %d", code)
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
	const email = "user3@example.com"

	stub := newOAuthStub("", "", "")
	defer stub.Close()

	db := newMockDB()
	db.seedUser(email)
	env := newTestEnv(stub, db)

	refresh, _ := env.issueTokens(email, "device")

	body, _ := json.Marshal(map[string]string{"refresh_token": refresh})
	req := httptest.NewRequest(http.MethodPost, "/refresh-access", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := env.do(req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	assertNoStore(t, w)
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
	const email = "user4@example.com"

	stub := newOAuthStub("", "", "")
	defer stub.Close()

	db := newMockDB()
	db.seedUser(email)
	env := newTestEnv(stub, db)

	_, access := env.issueTokens(email, "device")

	req := httptest.NewRequest(http.MethodGet, "/verify", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w := env.do(req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d: %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("X-Forwarded-User-Email"); got != email {
		t.Errorf("X-Forwarded-User-Email: want %q, got %q", email, got)
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
	const email = "user5@example.com"

	stub := newOAuthStub("", "", "")
	defer stub.Close()

	db := newMockDB()
	db.seedUser(email)
	env := newTestEnv(stub, db)

	refresh, _ := env.issueTokens(email, "device")

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
	const email = "user6@example.com"

	stub := newOAuthStub("", "", "")
	defer stub.Close()

	db := newMockDB()
	db.seedUser(email)
	env := newTestEnv(stub, db)

	_, access := env.issueTokens(email, "device")

	req := httptest.NewRequest(http.MethodDelete, "/users/me", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w := env.do(req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	if db.hasUser(email) {
		t.Error("user should have been deleted from DB")
	}
}

func accessUsername(t *testing.T, env *testEnv, accessToken string) string {
	t.Helper()
	claims, err := env.jwtManager.VerifyAccess(accessToken)
	if err != nil {
		t.Fatalf("VerifyAccess: %v", err)
	}
	return claims.Username
}

func TestCallback_UsernameFromUsersTable(t *testing.T) {
	const email = "named@example.com"

	stub := newOAuthStub("u1", email, "Provider Name")
	defer stub.Close()

	db := newMockDB()
	db.seedUsername(email, "Stored Name")
	env := newTestEnv(stub, db)

	stateVal := loginState(t, env)
	w := env.do(httptest.NewRequest(http.MethodGet, "/callback?code=testcode&state="+stateVal, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Message struct {
			AccessToken string `json:"access_token"`
		} `json:"message"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got := accessUsername(t, env, resp.Message.AccessToken); got != "Stored Name" {
		t.Errorf("username claim: want %q, got %q", "Stored Name", got)
	}
}

func TestRefresh_UsernameFromUsersTable(t *testing.T) {
	const email = "renamed@example.com"

	stub := newOAuthStub("", "", "")
	defer stub.Close()

	db := newMockDB()
	db.seedUsername(email, "Old Name")
	env := newTestEnv(stub, db)

	refresh, _ := env.issueTokens(email, "device")
	db.seedUsername(email, "New Name")

	req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: refresh})
	w := env.do(req)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Message struct {
			AccessToken string `json:"access_token"`
		} `json:"message"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got := accessUsername(t, env, resp.Message.AccessToken); got != "New Name" {
		t.Errorf("username claim: want %q, got %q", "New Name", got)
	}
}

func TestRefresh_UnknownUserKeepsSession(t *testing.T) {
	const email = "vanished@example.com"

	stub := newOAuthStub("", "", "")
	defer stub.Close()

	db := newMockDB()
	db.seedUser(email)
	env := newTestEnv(stub, db)

	refresh, _ := env.issueTokens(email, "device")
	db.mu.Lock()
	delete(db.users, email)
	db.mu.Unlock()

	if code, _ := refreshWithCookie(env, refresh); code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", code)
	}
	if _, err := env.jwtManager.VerifyRefresh(refresh); err != nil {
		t.Errorf("refresh token must not be rotated when the username lookup fails: %v", err)
	}
}

func TestRefreshAccess_UsernameFromUsersTable(t *testing.T) {
	const email = "access-name@example.com"

	stub := newOAuthStub("", "", "")
	defer stub.Close()

	db := newMockDB()
	db.seedUsername(email, "Access Name")
	env := newTestEnv(stub, db)

	refresh, _ := env.issueTokens(email, "device")

	body, _ := json.Marshal(map[string]string{"refresh_token": refresh})
	req := httptest.NewRequest(http.MethodPost, "/refresh-access", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := env.do(req)
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got := accessUsername(t, env, resp.Message); got != "Access Name" {
		t.Errorf("username claim: want %q, got %q", "Access Name", got)
	}
}

func TestVerify_ForwardsUsername(t *testing.T) {
	const email = "verify-name@example.com"

	stub := newOAuthStub("", "", "")
	defer stub.Close()

	db := newMockDB()
	db.seedUser(email)
	env := newTestEnv(stub, db)

	refresh, _ := env.issueTokens(email, "device")
	access, err := env.jwtManager.GenerateAccess(refresh, "Verify Name")
	if err != nil {
		t.Fatalf("GenerateAccess: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/verify", nil)
	req.Header.Set("Authorization", "Bearer "+access)
	w := env.do(req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d: %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("X-Forwarded-Username"); got != "Verify Name" {
		t.Errorf("X-Forwarded-Username: want %q, got %q", "Verify Name", got)
	}
}

func TestCallback_UsernameLookupFailureCreatesNoDevice(t *testing.T) {
	const email = "lookup-fail@example.com"

	stub := newOAuthStub("u1", email, "Lookup Fail")
	defer stub.Close()

	db := newMockDB()
	db.seedUser(email)
	db.usernameErr = errors.New("connection reset")
	env := newTestEnv(stub, db)

	stateVal := loginState(t, env)
	w := env.do(httptest.NewRequest(http.MethodGet, "/callback?code=testcode&state="+stateVal, nil))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d: %s", w.Code, w.Body.String())
	}
	if n := db.deviceCount(); n != 0 {
		t.Errorf("no device session should be created, got %d", n)
	}
}

func TestCallback_RegistrationUsesProviderName(t *testing.T) {
	const email = "register-name@example.com"

	stub := newOAuthStub("u1", email, "Provider Name")
	defer stub.Close()

	db := newMockDB()
	env := newTestEnv(stub, db, options.WithAllowRegistration(true))

	stateVal := loginState(t, env)
	w := env.do(httptest.NewRequest(http.MethodGet, "/callback?code=testcode&state="+stateVal, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Message struct {
			AccessToken string `json:"access_token"`
		} `json:"message"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got := accessUsername(t, env, resp.Message.AccessToken); got != "Provider Name" {
		t.Errorf("username claim: want %q, got %q", "Provider Name", got)
	}
}
