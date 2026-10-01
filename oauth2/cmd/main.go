package main

import (
	"context"
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
	"github.com/lucap9056/auth-middleware/oauth2/internal/config"
	"github.com/lucap9056/auth-middleware/oauth2/internal/flight"
	"github.com/lucap9056/auth-middleware/oauth2/internal/handlers"
	"github.com/lucap9056/auth-middleware/oauth2/internal/handlers/login"
	"github.com/lucap9056/auth-middleware/oauth2/internal/handlers/options"
	"github.com/lucap9056/auth-middleware/oauth2/internal/oauthclient"
	"github.com/lucap9056/auth-middleware/oauth2/internal/providers"
	"github.com/lucap9056/go-lifecycle/v2/lifecycle"
	"github.com/lucap9056/go-lifecycle/v2/runner"
)

func main() {
	if err := runner.Run(run); err != nil {
		log.Fatalln(err)
	}
}

func run(life *lifecycle.Coordinator) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	var db *database.Database
	if cfg.Database != nil {
		db, err = database.NewDatabase(cfg.Database.URL,
			database.WithMaxOpenConns(cfg.Database.MaxOpenConns),
			database.WithMaxIdleConns(cfg.Database.MaxIdleConns),
			database.WithConnMaxLifetime(cfg.Database.ConnMaxLifetime),
			database.WithConnMaxIdleTime(cfg.Database.ConnMaxIdleTime),
			database.WithCleanupInterval(cfg.Database.CleanupInterval),
		)
		if err != nil {
			return fmt.Errorf("failed to open database: %w", err)
		}
		life.OnExit(func() { db.Close() })
	}

	var redisClient *cache.RedisClient
	var flightOptions []flight.Option
	if cfg.Redis != nil {
		redisClient, err = cache.NewRedisClient(cfg.Redis.URL)
		if err != nil {
			return fmt.Errorf("failed to connect to Redis: %w", err)
		}
		life.OnExit(redisClient.Close)
		flightOptions = append(flightOptions, flight.WithRedis(cfg.Redis.URL, 0))
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

	jwtManager := jwt.NewJWTManager(jwtDB,
		jwt.WithAccessTokenDuration(cfg.JWT.AccessTokenDuration),
		jwt.WithRefreshTokenDuration(cfg.JWT.RefreshTokenDuration),
	)

	oauth2Client, err := newOAuth2Client(cfg)
	if err != nil {
		return err
	}

	flightGroup, err := flight.New(flightOptions...)
	if err != nil {
		return fmt.Errorf("failed to create flight group: %w", err)
	}
	life.OnExit(flightGroup.Close)

	stateCache, err := state.NewCache(redisClient)
	if err != nil {
		return fmt.Errorf("failed to create state cache: %w", err)
	}

	authOptions := []options.Option{
		options.WithDevMode(cfg.HTTP.DevMode()),
		options.WithAllowRegistration(cfg.Auth.AllowRegistration),
		options.WithPassOAuthToken(cfg.Auth.PassOAuthToken),
		options.WithClientPKCE(cfg.OAuth2 != nil && cfg.OAuth2.Client.PKCE),
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

	listener, err := createListener(cfg.HTTP.Address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", cfg.HTTP.Address, err)
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

func newOAuth2Client(cfg *config.Config) (login.OAuth2Client, error) {
	oauth2Config := cfg.OAuth2
	if oauth2Config == nil {
		return nil, nil
	}

	providerOptions := []providers.Option{
		providers.WithAllowUnverifiedEmail(cfg.Auth.AllowUnverifiedEmail),
	}

	if oauth2Config.OIDC != nil {
		discoveryCtx, discoveryCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer discoveryCancel()

		oidcClient, err := oauthclient.NewOIDC(discoveryCtx, oauthclient.OIDCConfig{
			IssuerURL:    oauth2Config.OIDC.IssuerURL,
			ClientID:     oauth2Config.Client.ID,
			ClientSecret: oauth2Config.Client.Secret,
			RedirectURL:  oauth2Config.Client.RedirectURL,
			Scopes:       oauth2Config.Scopes,
		}, providerOptions...)
		if err != nil {
			return nil, fmt.Errorf("OIDC setup failed: %w", err)
		}
		log.Printf("Starting OIDC server (Issuer: %s) on %s (Mode: %s)", oauth2Config.OIDC.IssuerURL, cfg.HTTP.Address, cfg.HTTP.Mode)
		return oidcClient, nil
	}

	provider := oauth2Config.Provider
	oauth2Client := oauthclient.New(oauthclient.Config{
		Provider:     provider.Name,
		ClientID:     oauth2Config.Client.ID,
		ClientSecret: oauth2Config.Client.Secret,
		RedirectURL:  oauth2Config.Client.RedirectURL,
		AuthURL:      provider.AuthURL,
		TokenURL:     provider.TokenURL,
		UserinfoURL:  provider.UserinfoURL,
		RevokeURL:    provider.RevokeURL,
		Scopes:       oauth2Config.Scopes,
	}, providerOptions...)
	log.Printf("Starting OAuth2 server (Provider: %s) on %s (Mode: %s)", provider.Name, cfg.HTTP.Address, cfg.HTTP.Mode)
	return oauth2Client, nil
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
