package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lucap9056/auth-middleware/database"
	"github.com/lucap9056/auth-middleware/jwt"
	"github.com/lucap9056/auth-middleware/oauth2/internal/cache"
	"github.com/lucap9056/auth-middleware/oauth2/internal/cache/device"
	"github.com/lucap9056/auth-middleware/oauth2/internal/cache/state"
	"github.com/lucap9056/auth-middleware/oauth2/internal/flight"
	"github.com/lucap9056/auth-middleware/oauth2/internal/handlers"
	"github.com/lucap9056/auth-middleware/oauth2/internal/handlers/login"
	"github.com/lucap9056/auth-middleware/oauth2/internal/handlers/options"
	"github.com/lucap9056/auth-middleware/oauth2/internal/oauthclient"
	"github.com/lucap9056/auth-middleware/oauth2/internal/providers"
	"github.com/lucap9056/go-lifecycle/v2/lifecycle"
	"github.com/lucap9056/go-lifecycle/v2/runner"
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
	EnvOAuth2UserinfoURL  = "OAUTH2_USERINFO_URL"
	EnvOAuth2RevokeURL    = "OAUTH2_REVOKE_URL"
	EnvOAuth2Scopes       = "OAUTH2_SCOPES"
	EnvOAuth2ClientPKCE   = "OAUTH2_CLIENT_PKCE"
	EnvOIDCIssuerURL      = "OIDC_ISSUER_URL"
	EnvHTTPMode           = "HTTP_MODE"
	EnvAllowRegistration  = "ALLOW_REGISTRATION"
	EnvPassOAuthToken     = "PASS_OAUTH_TOKEN"
	EnvRedisURL           = "REDIS_URL"

	DefaultHTTPAddress = ":80"
	ModeDevelopment    = "development"
)

var mode = ModeDevelopment

func main() {
	if err := runner.Run(run); err != nil {
		log.Fatalln(err)
	}
}

func run(life *lifecycle.Coordinator) error {
	var db *database.Database
	databaseUrl := os.Getenv(EnvDatabaseURL)
	if databaseUrl != "" {
		dbOptions := database.FromEnv()
		var err error
		db, err = database.NewDatabase(databaseUrl, dbOptions)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		life.OnExit(func() { db.Close() })
	}

	jwtOptions := jwt.FromEnv()

	redisURL := os.Getenv(EnvRedisURL)

	redisClient, err := cache.NewRedisClient(redisURL)
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}
	if redisClient != nil {
		life.OnExit(redisClient.Close)
	}

	deviceCache, err := device.NewSecretCache(redisClient)
	if err != nil {
		return fmt.Errorf("failed to create device secret cache: %w", err)
	}
	life.OnExit(func() { deviceCache.Close() })

	var jwtDB jwt.Database
	var handlerDB options.DB
	if db != nil {
		cachedDB := device.NewCachedDB(db, deviceCache)
		jwtDB = cachedDB
		handlerDB = cachedDB
	}

	jwtManager := jwt.NewJWTManager(jwtDB, jwtOptions)

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
	oidcIssuer := os.Getenv(EnvOIDCIssuerURL)

	var oauth2Client login.OAuth2Client

	switch {
	case oidcIssuer != "" && !enableOAuth2:
		return errors.New("OIDC_ISSUER_URL requires OAUTH2_CLIENT_ID, OAUTH2_CLIENT_SECRET, and OAUTH2_REDIRECT_URL")
	case oidcIssuer != "":
		scopesStr := os.Getenv(EnvOAuth2Scopes)
		scopes := strings.Split(scopesStr, ",")
		for i := range scopes {
			scopes[i] = strings.TrimSpace(scopes[i])
		}

		discoveryCtx, discoveryCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer discoveryCancel()

		oidcClient, err := oauthclient.NewOIDC(discoveryCtx, oauthclient.OIDCConfig{
			IssuerURL:    oidcIssuer,
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       scopes,
		})
		if err != nil {
			return fmt.Errorf("OIDC setup failed: %w", err)
		}
		oauth2Client = oidcClient
		log.Printf("Starting OIDC server (Issuer: %s) on %s (Mode: %s)", oidcIssuer, httpAddress, mode)

	case enableOAuth2:
		authURL := os.Getenv(EnvOAuth2AuthURL)
		tokenURL := os.Getenv(EnvOAuth2TokenURL)
		userinfoURL := os.Getenv(EnvOAuth2UserinfoURL)
		revokeURL := os.Getenv(EnvOAuth2RevokeURL)
		scopesStr := os.Getenv(EnvOAuth2Scopes)
		provider := os.Getenv(EnvOAuth2Provider)

		if !providers.IsBuiltin(provider) && (authURL == "" || tokenURL == "" || userinfoURL == "") {
			return errors.New("generic OAuth2 provider requires AUTH_URL, TOKEN_URL, and USERINFO_URL")
		}

		scopes := strings.Split(scopesStr, ",")
		for i := range scopes {
			scopes[i] = strings.TrimSpace(scopes[i])
		}
		oauth2Client = oauthclient.New(oauthclient.Config{
			Provider:     provider,
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			AuthURL:      authURL,
			TokenURL:     tokenURL,
			UserinfoURL:  userinfoURL,
			RevokeURL:    revokeURL,
			Scopes:       scopes,
		})

		log.Printf("Starting OAuth2 server (Provider: %s) on %s (Mode: %s)", provider, httpAddress, mode)
	}

	flightGroup, err := flight.New(flight.WithRedis(redisURL, 0))
	if err != nil {
		return fmt.Errorf("failed to create flight group: %w", err)
	}
	life.OnExit(flightGroup.Close)

	stateCache, err := state.NewCache(redisClient)
	if err != nil {
		return fmt.Errorf("failed to create state cache: %w", err)
	}

	var authOptions []options.Option
	if devMode {
		authOptions = append(authOptions, options.WithDevMode(true))
	}
	if os.Getenv(EnvAllowRegistration) == "true" {
		authOptions = append(authOptions, options.WithAllowRegistration(true))
	}
	if os.Getenv(EnvPassOAuthToken) == "true" {
		authOptions = append(authOptions, options.WithPassOAuthToken(true))
	}

	if os.Getenv(EnvOAuth2ClientPKCE) == "true" {
		authOptions = append(authOptions, options.WithClientPKCE(true))
	}

	mux := http.NewServeMux()

	handlers.RegisterRoutes(mux, handlers.Dependencies{
		DB:           handlerDB,
		JWTManager:   jwtManager,
		Flight:       flightGroup,
		StateCache:   stateCache,
		OAuth2Client: oauth2Client,
		Options:      authOptions,
	})

	server := &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	listener, err := createListener(httpAddress)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", httpAddress, err)
	}
	life.OnExit(func() { listener.Close() })

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

	return nil
}

func createListener(addr string) (net.Listener, error) {
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
