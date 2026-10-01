package refresh

import (
	"net/http"

	"github.com/lucap9056/auth-middleware/jwt"
	"github.com/lucap9056/auth-middleware/oauth2/internal/cache/token"
	"github.com/lucap9056/auth-middleware/oauth2/internal/handlers/options"
	"github.com/lucap9056/auth-middleware/oauth2/internal/handlers/refreshtoken"
	"github.com/lucap9056/auth-middleware/oauth2/internal/handlers/response"
)

type Handler struct {
	db           options.DB
	jwtManager   *jwt.JWTManager
	refreshCache token.Cache
	secureCookie bool
}

func New(db options.DB, jwtManager *jwt.JWTManager, refreshCache token.Cache, secureCookie bool) *Handler {
	return &Handler{
		db:           db,
		jwtManager:   jwtManager,
		refreshCache: refreshCache,
		secureCookie: secureCookie,
	}
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := refreshtoken.FromRequest(r)
	if err != nil {
		response.JSON(w, false, "Invalid refresh token", http.StatusUnauthorized, err)
		return
	}

	claims, err := h.jwtManager.VerifyRefresh(refreshToken)
	if err != nil {
		response.JSON(w, false, "Invalid session or expired refresh token", http.StatusUnauthorized, err)
		return
	}

	cachedToken, err := h.refreshCache.Get(r.Context(), refreshToken)
	if err == nil && cachedToken != nil {
		response.JSON(w, true, cachedToken, http.StatusOK, nil)
		return
	}

	userID := claims.Subject

	user, err := h.db.GetUserFromID(userID)
	if err != nil {
		response.JSON(w, false, "User not found", http.StatusUnauthorized, err)
		return
	}

	newRefreshToken, err := h.jwtManager.GenerateRefresh(userID, claims.DeviceID)
	if err != nil {
		response.JSON(w, false, "Failed to rotate refresh token", http.StatusInternalServerError, err)
		return
	}

	accessToken, err := h.jwtManager.GenerateAccess(newRefreshToken, user.Username)
	if err != nil {
		response.JSON(w, false, "Access token generation failed", http.StatusInternalServerError, err)
		return
	}

	token := token.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}
	h.refreshCache.Set(r.Context(), refreshToken, token)

	refreshtoken.SetCookie(w, newRefreshToken, h.secureCookie)
	response.JSON(w, true, token, http.StatusOK, nil)

}

func (h *Handler) RefreshAccess(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := refreshtoken.FromRequest(r)
	if err != nil {
		response.JSON(w, false, "Invalid refresh token", http.StatusUnauthorized, err)
		return
	}

	claims, err := h.jwtManager.VerifyRefresh(refreshToken)
	if err != nil {
		response.JSON(w, false, "Invalid session or expired refresh token", http.StatusUnauthorized, err)
		return
	}

	user, err := h.db.GetUserFromID(claims.Subject)
	if err != nil {
		response.JSON(w, false, "User not found", http.StatusUnauthorized, err)
		return
	}

	accessToken, err := h.jwtManager.GenerateAccess(refreshToken, user.Username)
	if err != nil {
		response.JSON(w, false, "Refresh failed", http.StatusInternalServerError, err)
		return
	}

	response.JSON(w, true, accessToken, http.StatusOK, nil)
}
