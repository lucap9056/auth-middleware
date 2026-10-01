package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

var (
	issuer     = getenv("ISSUER_URL", "http://test-provider:5556")
	listenAddr = getenv("LISTEN_ADDR", ":5556")

	userSub   = getenv("USER_SUB", "test-user-001")
	userEmail = getenv("USER_EMAIL", "user@example.com")
	userName  = getenv("USER_NAME", "Test User")
)

type pendingCode struct {
	challenge string
	expiresAt time.Time
}

var (
	codeMu    sync.Mutex
	codeStore = map[string]pendingCode{}
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/openid-configuration", handleDiscovery)
	mux.HandleFunc("GET /auth", handleAuth)
	mux.HandleFunc("POST /token", handleToken)
	mux.HandleFunc("GET /userinfo", handleUserinfo)

	log.Printf("test OAuth2 provider  issuer=%s  addr=%s", issuer, listenAddr)

	listener, err := newListener(listenAddr)
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(http.Serve(listener, mux))
}

func newListener(addr string) (net.Listener, error) {
	if after, ok := strings.CutPrefix(addr, "unix://"); ok {
		if err := os.MkdirAll(filepath.Dir(after), 0777); err != nil {
			return nil, err
		}
		temp := after + ".temp"
		os.Remove(temp)
		os.Remove(after)
		l, err := net.Listen("unix", temp)
		if err != nil {
			return nil, err
		}
		if err := os.Chmod(temp, 0666); err != nil {
			l.Close()
			os.Remove(temp)
			return nil, err
		}
		if err := os.Rename(temp, after); err != nil {
			l.Close()
			os.Remove(temp)
			return nil, err
		}
		return l, nil
	}
	return net.Listen("tcp", addr)
}

func handleDiscovery(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{
		"issuer":                                issuer,
		"authorization_endpoint":                issuer + "/auth",
		"token_endpoint":                        issuer + "/token",
		"userinfo_endpoint":                     issuer + "/userinfo",
		"response_types_supported":              []string{"code"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"none"},
		"scopes_supported":                      []string{"openid", "profile", "email"},
		"code_challenge_methods_supported":      []string{"S256"},
	})
}

func handleAuth(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	redirectURI := q.Get("redirect_uri")
	state := q.Get("state")
	challenge := q.Get("code_challenge")

	if redirectURI == "" {
		http.Error(w, "missing redirect_uri", http.StatusBadRequest)
		return
	}

	code := randomToken()
	codeMu.Lock()
	codeStore[code] = pendingCode{challenge: challenge, expiresAt: time.Now().Add(5 * time.Minute)}
	codeMu.Unlock()

	u, err := url.Parse(redirectURI)
	if err != nil {
		http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
		return
	}
	params := u.Query()
	params.Set("code", code)
	if state != "" {
		params.Set("state", state)
	}
	u.RawQuery = params.Encode()

	http.Redirect(w, r, u.String(), http.StatusFound)
}

func handleToken(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, `{"error":"invalid_request"}`, http.StatusBadRequest)
		return
	}
	code := r.FormValue("code")
	verifier := r.FormValue("code_verifier")

	codeMu.Lock()
	pending, ok := codeStore[code]
	if ok {
		delete(codeStore, code)
	}
	codeMu.Unlock()

	if !ok || time.Now().After(pending.expiresAt) {
		http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
		return
	}
	if pending.challenge != "" && !pkceOK(verifier, pending.challenge) {
		http.Error(w, `{"error":"invalid_grant","error_description":"pkce verification failed"}`, http.StatusBadRequest)
		return
	}

	writeJSON(w, map[string]any{
		"access_token": randomToken(),
		"token_type":   "Bearer",
		"expires_in":   3600,
		"id_token":     buildIDToken(),
	})
}

func handleUserinfo(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{
		"sub":            userSub,
		"email":          userEmail,
		"email_verified": true,
		"name":           userName,
	})
}

func buildIDToken() string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none","typ":"JWT"}`))
	payload, _ := json.Marshal(map[string]any{
		"iss":            issuer,
		"sub":            userSub,
		"email":          userEmail,
		"email_verified": true,
		"name":           userName,
		"aud":            "oauth2-middleware",
		"iat":            time.Now().Unix(),
		"exp":            time.Now().Add(time.Hour).Unix(),
	})
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + "."
}

func pkceOK(verifier, challenge string) bool {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:]) == challenge
}

func randomToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
