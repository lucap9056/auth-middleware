package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/lucap9056/auth-middleware/database"
	"github.com/lucap9056/auth-middleware/jwt"
	"github.com/lucap9056/auth-middleware/oauth2/internal/cache"
	"github.com/lucap9056/auth-middleware/oauth2/internal/handlers"
	"github.com/lucap9056/go-envfile/envfile"
	"github.com/lucap9056/go-lifecycle/lifecycle"
)

const (
	EnvDatabaseURL        = "DATABASE_URL"
	EnvHTTPAddress        = "HTTP_ADDRESS"
	EnvOAuth2Provider     = "OAUTH2_PROVIDER"
	EnvOAuth2ClientID     = "OAUTH2_CLIENT_ID"
	EnvOAuth2ClientSecret = "OAUTH2_CLIENT_SECRET"
	EnvOAuth2RedirectURL  = "OAUTH2_REDIRECT_URL"
	EnvOAuth2AuthURL      = "OAUTH2_AUTH_URL"
	EnvOAuth2TokenURL     = "OAUTH2_TOKEN_URL"
	EnvOAuth2UserInfoURL  = "OAUTH2_USER_INFO_URL"
	EnvOAuth2RevokeURL    = "OAUTH2_REVOKE_URL"
	EnvOAuth2Scopes       = "OAUTH2_SCOPES"
	EnvHTTPMode           = "HTTP_MODE"
	EnvAllowRegistration  = "ALLOW_REGISTRATION"
	EnvPassOAuthToken     = "PASS_OAUTH_TOKEN"
	EnvRedisURL           = "REDIS_URL"
	EnvRefreshTokenTTL    = "REFRESH_TOKEN_TTL"

	DefaultHTTPAddress = ":80"
	ModeDevelopment    = "development"
)

var mode = ModeDevelopment

func main() {
	envfile.Load()
	life := lifecycle.New()

	var db *database.Database
	databaseUrl := os.Getenv(EnvDatabaseURL)
	if databaseUrl != "" {

		dbOptions := database.FromEnv()
		db, err := database.NewDatabase(databaseUrl, dbOptions)
		if err != nil {
			log.Fatalf("Failed to open database: %v", err)
		}
		defer db.Close()

	}

	jwtOptions := jwt.FromEnv()
	jwtManager := jwt.NewJWTManager(db, jwtOptions)

	httpAddress := os.Getenv(EnvHTTPAddress)
	if httpAddress == "" {
		httpAddress = DefaultHTTPAddress
	}

	clientID := os.Getenv(EnvOAuth2ClientID)
	clientSecret := os.Getenv(EnvOAuth2ClientSecret)
	redirectURL := os.Getenv(EnvOAuth2RedirectURL)
	httpMode := os.Getenv(EnvHTTPMode)

	if httpMode != "" {
		mode = httpMode
	}

	devMode := (mode == ModeDevelopment)

	enableOAuth2 := clientID != "" && clientSecret != "" && redirectURL != ""

	var oauth2Handler *handlers.OAuth2Handler

	if enableOAuth2 {

		authURL := os.Getenv(EnvOAuth2AuthURL)
		tokenURL := os.Getenv(EnvOAuth2TokenURL)
		userInfoURL := os.Getenv(EnvOAuth2UserInfoURL)
		revokeURL := os.Getenv(EnvOAuth2RevokeURL)
		scopesStr := os.Getenv(EnvOAuth2Scopes)
		provider := os.Getenv(EnvOAuth2Provider)

		isGeneric := (provider != handlers.ProviderDiscordName && provider != handlers.ProviderGoogleName)
		if isGeneric && (authURL == "" || tokenURL == "" || userInfoURL == "") {
			log.Fatalln("Generic OAuth2 provider requires AUTH_URL, TOKEN_URL, and USER_INFO_URL")
		}

		scopes := strings.Split(scopesStr, ",")
		for i := range scopes {
			scopes[i] = strings.TrimSpace(scopes[i])
		}
		oauth2Handler = handlers.NewOAuth2Handler(provider, clientID, clientSecret, redirectURL, authURL, tokenURL, scopes, userInfoURL, revokeURL)

		log.Printf("Starting OAuth2 server (Provider: %s) on %s (Mode: %s)", provider, httpAddress, mode)
	}

	var cacaheOptions []cache.CacheOption
	redisURL := os.Getenv(EnvRedisURL)
	refreshTokenTTL := os.Getenv(EnvRefreshTokenTTL)

	if refreshTokenTTL != "" {
		duration, err := time.ParseDuration(refreshTokenTTL)
		if err != nil {
			log.Fatalf("Invalid REFRESH_TOKEN_TTL: %v", err)
		}
		cacaheOptions = append(cacaheOptions, cache.WithTTL(duration))

	}
	refreshCache, err := cache.NewCache(redisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	var authOptions []handlers.AuthOption
	if devMode {
		authOptions = append(authOptions, handlers.WithDevMode(true))
	}
	if os.Getenv(EnvAllowRegistration) == "true" {
		authOptions = append(authOptions, handlers.WithAllowRegistration(true))
	}
	if os.Getenv(EnvPassOAuthToken) == "true" {
		authOptions = append(authOptions, handlers.WithPassOAuthToken(true))
	}

	authHandler := handlers.NewAuthHandler(db, jwtManager, refreshCache, oauth2Handler, authOptions...)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", authHandler.Health)
	if db != nil {
		mux.HandleFunc("POST /refresh", authHandler.Refresh)
		mux.HandleFunc("POST /refresh-access", authHandler.RefreshAccess)
		mux.HandleFunc("POST /logout", authHandler.Logout)
		mux.HandleFunc("GET /verify", authHandler.Verify)
	}

	if enableOAuth2 {
		mux.HandleFunc("GET /login", authHandler.Login)
		mux.HandleFunc("GET /callback", authHandler.Callback)
		log.Println("OAuth2 is enabled")
	}

	server := &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	listener, err := createListener(httpAddress)
	if err != nil {
		log.Fatalln("")
	}
	defer listener.Close()

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			life.Exitln(err.Error())
		}
	}()

	life.OnExit(func() {
		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	})

	life.Wait()
}

func createListener(addr string) (net.Listener, error) {
	if path, ok := strings.CutPrefix(addr, "unix://"); ok {
		return net.Listen("unix", path)
	}
	return net.Listen("tcp", addr)
}
