package integration

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/lucap9056/auth-middleware/database"
	"github.com/lucap9056/auth-middleware/jwt"
	"github.com/lucap9056/auth-middleware/oauth2/internal/cache/token"
	"github.com/lucap9056/auth-middleware/oauth2/internal/handlers"
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
	s.Server = httptest.NewServer(mux)
	return s
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
		"id":    s.UserID,
		"email": s.Email,
		"name":  s.Name,
	})
}

func (s *oauthStub) handleRevoke(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// mockDB implements handlers.DB and jwt.Database using in-memory maps.
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
func newTestEnv(stub *oauthStub, db *mockDB, opts ...handlers.AuthOption) *testEnv {
	var jwtDB jwt.Database
	var handlerDB handlers.DB
	if db != nil {
		jwtDB = db
		handlerDB = db
	}

	oauth2Handler := handlers.NewOAuth2Handler(
		"generic",
		"test-client",
		"test-secret",
		"http://localhost/callback",
		stub.URL+"/authorize",
		stub.URL+"/token",
		[]string{"email"},
		stub.URL+"/userinfo",
		stub.URL+"/revoke",
	)

	jwtManager := jwt.NewJWTManager(jwtDB)
	refreshCache := token.NewCache(nil)
	authHandler := handlers.NewAuthHandler(handlerDB, jwtManager, refreshCache, oauth2Handler, opts...)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", authHandler.Health)
	mux.HandleFunc("GET /login", authHandler.Login)
	mux.HandleFunc("GET /callback", authHandler.Callback)
	mux.HandleFunc("POST /refresh", authHandler.Refresh)
	mux.HandleFunc("POST /refresh-access", authHandler.RefreshAccess)
	mux.HandleFunc("GET /verify", authHandler.Verify)
	mux.HandleFunc("POST /logout", authHandler.Logout)
	mux.HandleFunc("DELETE /users/me", authHandler.DeleteMe)

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
