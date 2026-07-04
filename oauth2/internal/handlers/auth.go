package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/lucap9056/auth-middleware/jwt"
	"github.com/lucap9056/auth-middleware/oauth2/internal/cache/state"
	"github.com/lucap9056/auth-middleware/oauth2/internal/cache/token"
	"golang.org/x/oauth2"
	"golang.org/x/sync/singleflight"
)

const (
	CookieRefreshToken = "refresh_token"
	DefaultDeviceName  = "Unknown Device"
)

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type Response[T any] struct {
	Success bool `json:"success"`
	Message T    `json:"message"`
}

type LoginResponse struct {
	Verifier string `json:"verifier,omitempty"`
	URL      string `json:"url"`
}

type ExchangeResponse struct {
	Success     bool
	Message     string
	State       int
	OAuth2Token *token.TokenPair
	AuthToken   *token.TokenPair
}

type AuthHandler struct {
	db                   DB
	jwtManager           *jwt.JWTManager
	refreshCache         token.Cache
	stateCache           state.Cache
	oauth2Handler        *OAuth2Handler
	config               *AuthConfig
	exchangeSingleflight singleflight.Group
}

func NewAuthHandler(db DB, jwtManager *jwt.JWTManager, refreshCache token.Cache, stateCache state.Cache, oauth2Handler *OAuth2Handler, opts ...AuthOption) *AuthHandler {
	config := &AuthConfig{
		DevMode:           false,
		AllowRegistration: false,
		PassOAuthToken:    false,
	}
	for _, opt := range opts {
		opt(config)
	}

	return &AuthHandler{
		db:            db,
		jwtManager:    jwtManager,
		refreshCache:  refreshCache,
		stateCache:    stateCache,
		oauth2Handler: oauth2Handler,
		config:        config,
	}
}

func (h *AuthHandler) Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {

	if h.oauth2Handler == nil {
		sendJSONResponse(w, false, "OAuth2 login is not available", http.StatusServiceUnavailable, nil)
		return
	}

	verifier := oauth2.GenerateVerifier()
	stateVal := generateState()

	if err := h.stateCache.Set(r.Context(), stateVal, verifier); err != nil {
		sendJSONResponse(w, false, "Failed to initialize login session", http.StatusInternalServerError, err)
		return
	}

	url := h.oauth2Handler.AuthURL(stateVal, verifier)
	resp := &LoginResponse{URL: url}
	if h.config.ClientPKCE {
		resp.Verifier = verifier
	}
	sendJSONResponse(w, true, resp, http.StatusOK, nil)
}

func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {

	if h.oauth2Handler == nil {
		sendJSONResponse(w, false, "OAuth2 callback is not available", http.StatusServiceUnavailable, nil)
		return
	}

	code := r.FormValue("code")
	state := r.FormValue("state")
	if state == "" {
		sendJSONResponse(w, false, "Missing state parameter", http.StatusBadRequest, nil)
		return
	}

	deviceName := r.Header.Get("X-Device-Name")
	if deviceName == "" {
		deviceName = DefaultDeviceName
	}

	headerVerifier := r.Header.Get("X-PKCE-Verifier")

	res, err := h.exchangeSingleflightDo(code, state, deviceName, headerVerifier)
	if err != nil || !res.Success {
		sendJSONResponse(w, res.Success, res.Message, res.State, err)
		return
	}

	if res.AuthToken != nil {

		h.setRefreshCookie(w, res.AuthToken.RefreshToken)

		if h.config.PassOAuthToken && res.OAuth2Token != nil {
			w.Header().Set("X-Forwarded-Refresh-Token", res.OAuth2Token.RefreshToken)
			w.Header().Set("X-Forwarded-Access-Token", res.OAuth2Token.AccessToken)
		}

		sendJSONResponse(w, res.Success, res.AuthToken, res.State, nil)
		return
	}

	if res.OAuth2Token != nil {

		if h.config.PassOAuthToken {
			w.Header().Set("X-Forwarded-Refresh-Token", res.OAuth2Token.RefreshToken)
			w.Header().Set("X-Forwarded-Access-Token", res.OAuth2Token.AccessToken)
			sendJSONResponse(w, res.Success, "Logged in", res.State, nil)
			return
		}

		sendJSONResponse(w, res.Success, res.OAuth2Token, res.State, nil)
	}

	sendJSONResponse(w, false, "", http.StatusInternalServerError, nil)
}

func (h *AuthHandler) exchangeSingleflightDo(code, state, device, headerVerifier string) (*ExchangeResponse, error) {
	v, err, _ := h.exchangeSingleflight.Do(code+"|"+state+"|"+headerVerifier, func() (any, error) {

		stateCtx, stateCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stateCancel()

		verifier, err := h.stateCache.Get(stateCtx, state)
		if err != nil {
			return &ExchangeResponse{
				Success: false,
				Message: "Failed to retrieve login session",
				State:   http.StatusInternalServerError,
			}, err
		}
		if verifier == "" {
			return &ExchangeResponse{
				Success: false,
				Message: "Invalid or expired state",
				State:   http.StatusBadRequest,
			}, nil
		}

		if h.config.ClientPKCE {
			if headerVerifier != verifier {
				return &ExchangeResponse{
					Success: false,
					Message: "Invalid PKCE verifier",
					State:   http.StatusBadRequest,
				}, nil
			}
		}

		exchangeCtx, exchangeCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer exchangeCancel()
		oauth2Token, err := h.oauth2Handler.Exchange(exchangeCtx, code, verifier)
		if err != nil {
			if err == context.DeadlineExceeded {
				return &ExchangeResponse{
					Success: false,
					Message: "OAuth2 provider timeout",
					State:   http.StatusGatewayTimeout,
				}, err
			}
			return &ExchangeResponse{
				Success: false,
				Message: "Code exchange failed",
				State:   http.StatusInternalServerError,
			}, err
		}

		defer func() {
			deferCtx, deferCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer deferCancel()
			h.stateCache.Delete(deferCtx, state)
		}()

		if h.db != nil {
			getUserCtx, getUserCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer getUserCancel()
			user, err := h.oauth2Handler.GetUser(getUserCtx, oauth2Token)
			if err != nil {
				return &ExchangeResponse{
					Success: false,
					Message: "Failed to fetch user info",
					State:   http.StatusInternalServerError,
				}, err
			}

			dbUser, err := h.db.GetUserFromEmail(user.Email)
			if err != nil {
				return &ExchangeResponse{
					Success: false,
					Message: "Database error",
					State:   http.StatusInternalServerError,
				}, err
			}

			if dbUser == nil {
				if h.config.AllowRegistration {
					dbUser, err = h.db.CreateUser(user.Name, user.Email)
					if err != nil {
						return &ExchangeResponse{
							Success: false,
							Message: "Failed to create user",
							State:   http.StatusInternalServerError,
						}, err
					}
				} else {
					return &ExchangeResponse{
						Success: false,
						Message: "User not found",
						State:   http.StatusUnauthorized,
					}, nil
				}
			}

			userID := fmt.Sprint(dbUser.UserID)
			secret := h.jwtManager.GenerateRandomSecret()

			deviceID, err := h.db.SaveDeviceSecret(userID, device, secret)
			if err != nil {
				return &ExchangeResponse{
					Success: false,
					Message: "Failed to register device session",
					State:   http.StatusInternalServerError,
				}, err
			}

			refreshToken, err := h.jwtManager.GenerateRefresh(userID, deviceID, secret)
			if err != nil {
				return &ExchangeResponse{
					Success: false,
					Message: "Refresh token generation failed",
					State:   http.StatusInternalServerError,
				}, err
			}

			accessToken, err := h.jwtManager.GenerateAccess(refreshToken, dbUser.Username)
			if err != nil {
				return &ExchangeResponse{
					Success: false,
					Message: "Access token generation failed",
					State:   http.StatusInternalServerError,
				}, err
			}

			if !h.config.PassOAuthToken {
				revokeCtx, revokeCancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer revokeCancel()
				h.oauth2Handler.Revoke(revokeCtx, oauth2Token)
			}

			return &ExchangeResponse{
				Success: true,
				Message: "Login successful",
				State:   http.StatusOK,
				OAuth2Token: &token.TokenPair{
					AccessToken:  oauth2Token.AccessToken,
					RefreshToken: oauth2Token.RefreshToken,
				},
				AuthToken: &token.TokenPair{
					AccessToken:  accessToken,
					RefreshToken: refreshToken,
				},
			}, nil
		}

		return &ExchangeResponse{
			Success: true,
			Message: "Login successful",
			State:   http.StatusOK,
			OAuth2Token: &token.TokenPair{
				AccessToken:  oauth2Token.AccessToken,
				RefreshToken: oauth2Token.RefreshToken,
			},
		}, nil
	})

	return v.(*ExchangeResponse), err
}

func (h *AuthHandler) getRefreshToken(r *http.Request) (string, error) {

	cookie, err := r.Cookie(CookieRefreshToken)
	if err == nil {
		return cookie.Value, nil
	}

	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return "", err
	}
	return req.RefreshToken, nil
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := h.getRefreshToken(r)
	if err != nil {
		sendJSONResponse(w, false, "Invalid refresh token", http.StatusUnauthorized, err)
		return
	}

	claims, err := h.jwtManager.VerifyRefresh(refreshToken)
	if err != nil {
		sendJSONResponse(w, false, "Invalid session or expired refresh token", http.StatusUnauthorized, err)
		return
	}

	cachedToken, err := h.refreshCache.Get(r.Context(), refreshToken)
	if err == nil && cachedToken != nil {
		sendJSONResponse(w, true, cachedToken, http.StatusOK, nil)
		return
	}

	userID := claims.Subject

	user, err := h.db.GetUserFromID(userID)
	if err != nil {
		sendJSONResponse(w, false, "User not found", http.StatusUnauthorized, err)
		return
	}

	newRefreshToken, err := h.jwtManager.GenerateRefresh(userID, claims.DeviceID)
	if err != nil {
		sendJSONResponse(w, false, "Failed to rotate refresh token", http.StatusInternalServerError, err)
		return
	}

	accessToken, err := h.jwtManager.GenerateAccess(newRefreshToken, user.Username)
	if err != nil {
		sendJSONResponse(w, false, "Access token generation failed", http.StatusInternalServerError, err)
		return
	}

	token := token.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}
	h.refreshCache.Set(r.Context(), refreshToken, token)

	h.setRefreshCookie(w, newRefreshToken)
	sendJSONResponse(w, true, token, http.StatusOK, nil)

}

func (h *AuthHandler) RefreshAccess(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := h.getRefreshToken(r)
	if err != nil {
		sendJSONResponse(w, false, "Invalid refresh token", http.StatusUnauthorized, err)
		return
	}

	claims, err := h.jwtManager.VerifyRefresh(refreshToken)
	if err != nil {
		sendJSONResponse(w, false, "Invalid session or expired refresh token", http.StatusUnauthorized, err)
		return
	}

	user, err := h.db.GetUserFromID(claims.Subject)
	if err != nil {
		sendJSONResponse(w, false, "User not found", http.StatusUnauthorized, err)
		return
	}

	accessToken, err := h.jwtManager.GenerateAccess(refreshToken, user.Username)
	if err != nil {
		sendJSONResponse(w, false, "Refresh failed", http.StatusInternalServerError, err)
		return
	}

	sendJSONResponse(w, true, accessToken, http.StatusOK, nil)
}

func (h *AuthHandler) Verify(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		sendJSONResponse(w, false, "Missing Bearer token", http.StatusUnauthorized, nil)
		return
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := h.jwtManager.VerifyAccess(tokenStr)
	if err != nil {
		sendJSONResponse(w, false, "Invalid access token", http.StatusUnauthorized, err)
		return
	}

	w.Header().Set("X-Forwarded-User-ID", claims.UserID)
	w.Header().Set("X-Forwarded-Device-ID", claims.DeviceID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := h.getRefreshToken(r)
	if err == nil {
		claims, pErr := h.jwtManager.VerifyRefresh(refreshToken)
		if pErr == nil {
			err = h.db.DeleteDevice(claims.Subject, claims.DeviceID)
			if err != nil {
				log.Printf("[WARN] Failed to delete device from DB on logout: %v", err)
			}
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     CookieRefreshToken,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	sendJSONResponse(w, true, "Logged out and device session revoked", http.StatusOK, nil)
}

func (h *AuthHandler) DeleteMe(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		sendJSONResponse(w, false, "Missing Bearer token", http.StatusUnauthorized, nil)
		return
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := h.jwtManager.VerifyAccess(tokenStr)
	if err != nil {
		sendJSONResponse(w, false, "Invalid access token", http.StatusUnauthorized, err)
		return
	}

	userID := claims.UserID

	if err := h.db.DeleteAllDevices(userID); err != nil {
		sendJSONResponse(w, false, "Failed to remove device sessions", http.StatusInternalServerError, err)
		return
	}

	if err := h.db.DeleteUser(userID); err != nil {
		sendJSONResponse(w, false, "Failed to delete account", http.StatusInternalServerError, err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     CookieRefreshToken,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})

	sendJSONResponse(w, true, "Account deleted", http.StatusOK, nil)
}

func (h *AuthHandler) setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieRefreshToken,
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HttpOnly: true,
		Secure:   !h.config.DevMode,
		SameSite: http.SameSiteLaxMode,
	})
}

func sendJSONResponse[T any](w http.ResponseWriter, success bool, message T, code int, internalErr error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	if internalErr != nil {
		log.Printf("[ERROR] Status: %d, Msg: %v, Err: %v", code, message, internalErr)
	}

	json.NewEncoder(w).Encode(Response[T]{
		Success: success,
		Message: message,
	})
}
