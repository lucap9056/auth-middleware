package session

import (
	"log"
	"net/http"
	"strings"

	"github.com/lucap9056/auth-middleware/jwt"
	"github.com/lucap9056/auth-middleware/oauth2/internal/handlers/options"
	"github.com/lucap9056/auth-middleware/oauth2/internal/handlers/refreshtoken"
	"github.com/lucap9056/auth-middleware/oauth2/internal/handlers/response"
)

type Handler struct {
	db         options.DB
	jwtManager *jwt.JWTManager
}

func New(db options.DB, jwtManager *jwt.JWTManager) *Handler {
	return &Handler{
		db:         db,
		jwtManager: jwtManager,
	}
}

func (h *Handler) Verify(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		response.Unauthorized(w, response.BearerChallenge, "Missing Bearer token", nil)
		return
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := h.jwtManager.VerifyAccess(tokenStr)
	if err != nil {
		response.Unauthorized(w, response.InvalidTokenChallenge, "Invalid access token", err)
		return
	}

	w.Header().Set("X-Forwarded-User-ID", claims.UserID)
	w.Header().Set("X-Forwarded-Device-ID", claims.DeviceID)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := refreshtoken.FromRequest(r)
	if err == nil {
		claims, pErr := h.jwtManager.VerifyRefresh(refreshToken)
		if pErr == nil {
			err = h.db.DeleteDevice(claims.Subject, claims.DeviceID)
			if err != nil {
				log.Printf("[WARN] Failed to delete device from DB on logout: %v", err)
			}
		}
	}

	refreshtoken.ClearCookie(w)

	response.JSON(w, true, "Logged out and device session revoked", http.StatusOK, nil)
}

func (h *Handler) DeleteMe(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		response.Unauthorized(w, response.BearerChallenge, "Missing Bearer token", nil)
		return
	}

	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := h.jwtManager.VerifyAccess(tokenStr)
	if err != nil {
		response.Unauthorized(w, response.InvalidTokenChallenge, "Invalid access token", err)
		return
	}

	userID := claims.UserID

	if err := h.db.DeleteAllDevices(userID); err != nil {
		response.JSON(w, false, "Failed to remove device sessions", http.StatusInternalServerError, err)
		return
	}

	if err := h.db.DeleteUser(userID); err != nil {
		response.JSON(w, false, "Failed to delete account", http.StatusInternalServerError, err)
		return
	}

	refreshtoken.ClearCookie(w)

	response.JSON(w, true, "Account deleted", http.StatusOK, nil)
}
