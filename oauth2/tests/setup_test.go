package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/lucap9056/auth-middleware/database"
	"github.com/lucap9056/auth-middleware/jwt"
	"github.com/lucap9056/auth-middleware/oauth2/internal/cache/state"
	"github.com/lucap9056/auth-middleware/oauth2/internal/flight"
	"github.com/lucap9056/auth-middleware/oauth2/internal/handlers"
	"github.com/lucap9056/auth-middleware/oauth2/internal/handlers/login"
	"github.com/lucap9056/auth-middleware/oauth2/internal/handlers/options"
	"github.com/lucap9056/auth-middleware/oauth2/internal/oauthclient"
)

// oauthStub is a minimal in-process OAuth2 provider for testing.
type oauthStub struct {
	*httptest.Server
	UserID string
	Email  string
	Name   string
}

func newOAuthStub(userID, email, name string) *oauthStub {
	s := &oauthStub{UserID: userID, Email: email, Name: name}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /token", s.handleToken)
	mux.HandleFunc("GET /userinfo", s.handleUserInfo)
	mux.HandleFunc("POST /revoke", s.handleRevoke)
	mux.HandleFunc("GET /.well-known/openid-configuration", s.handleDiscovery)
	s.Server = httptest.NewServer(mux)
	return s
}

func (s *oauthStub) handleDiscovery(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"issuer":                 s.URL,
		"authorization_endpoint": s.URL + "/authorize",
		"token_endpoint":         s.URL + "/token",
		"userinfo_endpoint":      s.URL + "/userinfo",
		"revocation_endpoint":    s.URL + "/revoke",
		"authorization_response_iss_parameter_supported": true,
	})
}

func (s *oauthStub) handleToken(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"access_token":  "stub-access-token",
		"refresh_token": "stub-refresh-token",
		"token_type":    "Bearer",
		"expires_in":    3600,
	})
}

func (s *oauthStub) handleUserInfo(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"id":             s.UserID,
		"email":          s.Email,
		"email_verified": true,
		"name":           s.Name,
	})
}

func (s *oauthStub) handleRevoke(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// mockDB implements options.DB and jwt.Database using in-memory maps.
type mockDB struct {
	mu           sync.Mutex
	users        map[string]*database.User // email -> user
	usersID      map[string]*database.User // userID -> user
	deviceOwner  map[string]string         // deviceID -> userID
	deviceSecret map[string]string         // deviceID -> secret
	idCounter    int
}

func newMockDB() *mockDB {
	return &mockDB{
		users:        make(map[string]*database.User),
		usersID:      make(map[string]*database.User),
		deviceOwner:  make(map[string]string),
		deviceSecret: make(map[string]string),
	}
}

func (m *mockDB) seedUser(u *database.User) {
	m.users[u.Email] = u
	m.usersID[u.UserID] = u
}

func (m *mockDB) GetUserFromEmail(email string) (*database.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.users[email], nil
}

func (m *mockDB) GetUserFromID(id string) (*database.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.usersID[id], nil
}

func (m *mockDB) CreateUser(username, email string) (*database.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.idCounter++
	u := &database.User{
		UserID:   fmt.Sprintf("user-%d", m.idCounter),
		Username: username,
		Email:    email,
	}
	m.users[email] = u
	m.usersID[u.UserID] = u
	return u, nil
}

func (m *mockDB) SaveDeviceSecret(userID, _, secret string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.idCounter++
	deviceID := fmt.Sprintf("device-%d", m.idCounter)
	m.deviceOwner[deviceID] = userID
	m.deviceSecret[deviceID] = secret
	return deviceID, nil
}

func (m *mockDB) UpdateDeviceSecret(deviceID, secret string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.deviceSecret[deviceID]; !ok {
		return errors.New("device not found")
	}
	m.deviceSecret[deviceID] = secret
	return nil
}

func (m *mockDB) GetDeviceSecret(deviceID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.deviceSecret[deviceID]
	if !ok {
		return "", fmt.Errorf("device %s not found", deviceID)
	}
	return s, nil
}

func (m *mockDB) DeleteDevice(_, deviceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.deviceOwner, deviceID)
	delete(m.deviceSecret, deviceID)
	return nil
}

func (m *mockDB) DeleteAllDevices(userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, owner := range m.deviceOwner {
		if owner == userID {
			delete(m.deviceOwner, id)
			delete(m.deviceSecret, id)
		}
	}
	return nil
}

func (m *mockDB) DeleteUser(userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, ok := m.usersID[userID]; ok {
		delete(m.users, u.Email)
		delete(m.usersID, userID)
	}
	return nil
}

// testEnv holds all wired-up dependencies for a single test scenario.
type testEnv struct {
	stub       *oauthStub
	db         *mockDB
	jwtManager *jwt.JWTManager
	mux        *http.ServeMux
}

// newTestEnv builds a handler stack backed by the given stub and mock DB.
// Pass db=nil to simulate a no-database deployment.
func newTestEnv(stub *oauthStub, db *mockDB, opts ...options.Option) *testEnv {
	oauth2Client := oauthclient.New(oauthclient.Config{
		Provider:     "generic",
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost/callback",
		AuthURL:      stub.URL + "/authorize",
		TokenURL:     stub.URL + "/token",
		UserinfoURL:  stub.URL + "/userinfo",
		RevokeURL:    stub.URL + "/revoke",
		Scopes:       []string{"email"},
	})
	return newTestEnvWithClient(stub, db, oauth2Client, opts...)
}

func newOIDCTestEnv(t *testing.T, stub *oauthStub, db *mockDB, opts ...options.Option) *testEnv {
	t.Helper()
	oauth2Client, err := oauthclient.NewOIDC(context.Background(), oauthclient.OIDCConfig{
		IssuerURL:    stub.URL,
		ClientID:     "test-client",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost/callback",
		Scopes:       []string{"email"},
	})
	if err != nil {
		t.Fatalf("NewOIDC: %v", err)
	}
	return newTestEnvWithClient(stub, db, oauth2Client, opts...)
}

func newTestEnvWithClient(stub *oauthStub, db *mockDB, oauth2Client login.OAuth2Client, opts ...options.Option) *testEnv {
	var jwtDB jwt.Database
	var handlerDB options.DB
	if db != nil {
		jwtDB = db
		handlerDB = db
	}

	jwtManager := jwt.NewJWTManager(jwtDB)
	flightGroup, err := flight.New()
	if err != nil {
		panic(err)
	}
	stateCache, err := state.NewCache(nil)
	if err != nil {
		panic(err)
	}
	mux := http.NewServeMux()
	handlers.RegisterRoutes(mux, handlers.Dependencies{
		DB:           handlerDB,
		JWTManager:   jwtManager,
		Flight:       flightGroup,
		StateCache:   stateCache,
		OAuth2Client: oauth2Client,
		Options:      opts,
	})

	return &testEnv{stub: stub, db: db, jwtManager: jwtManager, mux: mux}
}

func (e *testEnv) do(r *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	e.mux.ServeHTTP(w, r)
	return w
}

// issueTokens seeds a device in the mock DB and returns a valid (refresh, access) pair.
func (e *testEnv) issueTokens(userID, username, deviceName string) (refresh, access string) {
	secret := e.jwtManager.GenerateRandomSecret()
	deviceID, _ := e.db.SaveDeviceSecret(userID, deviceName, secret)
	refresh, _ = e.jwtManager.GenerateRefresh(userID, deviceID, secret)
	access, _ = e.jwtManager.GenerateAccess(refresh, username)
	return
}
