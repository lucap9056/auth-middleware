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
	"github.com/lucap9056/auth-middleware/oauth2/internal/cache/token"
)

const (
	CookieOAuthState   = "oauth_state"
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
	Verifier  string `json:"verifier"`
	Challenge string `json:"challenge"`
	URL       string `json:"url"`
}

type AuthHandler struct {
	db            DB
	jwtManager    *jwt.JWTManager
	refreshCache  token.Cache
	oauth2Handler *OAuth2Handler
	config        *AuthConfig
}

func NewAuthHandler(db DB, jwtManager *jwt.JWTManager, refreshCache token.Cache, oauth2Handler *OAuth2Handler, opts ...AuthOption) *AuthHandler {
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

	verifier, challenge := generatePKCE()

	url := h.oauth2Handler.AuthURL("", challenge)
	sendJSONResponse(w, true, &LoginResponse{
		Verifier:  verifier,
		Challenge: challenge,
		URL:       url,
	}, http.StatusOK, nil)
}

func (h *AuthHandler) Callback(w http.ResponseWriter, r *http.Request) {

	if h.oauth2Handler == nil {
		sendJSONResponse(w, false, "OAuth2 callback is not available", http.StatusServiceUnavailable, nil)
		return
	}

	code := r.FormValue("code")
	codeVerifier := r.URL.Query().Get("code_verifier")
	if codeVerifier == "" {
		sendJSONResponse(w, false, "PKCE verifier missing", http.StatusBadRequest, nil)
		return
	}

	deviceName := r.Header.Get("X-Device-Name")
	if deviceName == "" {
		deviceName = DefaultDeviceName
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	oauthToken, err := h.oauth2Handler.Exchange(ctx, code, codeVerifier)
	if err != nil {
		sendJSONResponse(w, false, "Code exchange failed", http.StatusInternalServerError, err)
		return
	}

	if h.db != nil {
		user, err := h.oauth2Handler.GetUser(ctx, oauthToken)
		if err != nil {
			sendJSONResponse(w, false, "Failed to fetch user info", http.StatusInternalServerError, err)
			return
		}

		dbUser, err := h.db.GetUserFromEmail(user.Email)
		if err != nil {
			sendJSONResponse(w, false, "Database error", http.StatusInternalServerError, err)
			return
		}

		if dbUser == nil {
			if h.config.AllowRegistration {
				dbUser, err = h.db.CreateUser(user.Name, user.Email)
				if err != nil {
					sendJSONResponse(w, false, "Failed to create user", http.StatusInternalServerError, err)
					return
				}
			} else {
				sendJSONResponse(w, false, "User not found", http.StatusUnauthorized, nil)
				return
			}
		}

		userID := fmt.Sprint(dbUser.UserID)
		secret := h.jwtManager.GenerateRandomSecret()

		deviceID, err := h.db.SaveDeviceSecret(userID, deviceName, secret)
		if err != nil {
			sendJSONResponse(w, false, "Failed to register device session", http.StatusInternalServerError, err)
			return
		}

		refreshToken, err := h.jwtManager.GenerateRefresh(userID, deviceID, secret)
		if err != nil {
			sendJSONResponse(w, false, "Refresh token generation failed", http.StatusInternalServerError, err)
			return
		}

		accessToken, err := h.jwtManager.GenerateAccess(refreshToken, dbUser.Username)
		if err != nil {
			sendJSONResponse(w, false, "Access token generation failed", http.StatusInternalServerError, err)
			return
		}

		h.setRefreshCookie(w, refreshToken)

		if h.config.PassOAuthToken {
			w.Header().Set("X-Forwarded-Refresh-Token", oauthToken.RefreshToken)
			w.Header().Set("X-Forwarded-Access-Token", oauthToken.AccessToken)
		} else {
			h.oauth2Handler.Revoke(r.Context(), oauthToken)
		}

		sendJSONResponse(w, true, &token.TokenPair{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		}, http.StatusOK, nil)
		return
	}

	if h.config.PassOAuthToken {
		w.Header().Set("X-Forwarded-Refresh-Token", oauthToken.RefreshToken)
		w.Header().Set("X-Forwarded-Access-Token", oauthToken.AccessToken)
		sendJSONResponse(w, true, "Logged in", http.StatusOK, nil)
		return
	} else {
		sendJSONResponse(w, true, &token.TokenPair{
			AccessToken:  oauthToken.AccessToken,
			RefreshToken: oauthToken.RefreshToken,
		}, http.StatusOK, nil)
		return
	}

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
	w.WriteHeader(http.StatusOK)
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
