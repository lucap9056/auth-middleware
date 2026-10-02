package refresh

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/lucap9056/auth-middleware/jwt"
	"github.com/lucap9056/auth-middleware/oauth2/internal/flight"
	"github.com/lucap9056/auth-middleware/oauth2/internal/handlers/options"
	"github.com/lucap9056/auth-middleware/oauth2/internal/handlers/refreshtoken"
	"github.com/lucap9056/auth-middleware/oauth2/internal/handlers/response"
)

const (
	flightKeyPrefix = "refresh:"
	rotateTimeout   = 10 * time.Second
)

type rotateError struct {
	message string
	status  int
	err     error
}

func (e *rotateError) Error() string {
	return fmt.Sprintf("%s: %v", e.message, e.err)
}

func (e *rotateError) Unwrap() error {
	return e.err
}

type Handler struct {
	db           options.DB
	jwtManager   *jwt.JWTManager
	flight       *flight.Group
	secureCookie bool
}

func New(db options.DB, jwtManager *jwt.JWTManager, flightGroup *flight.Group, secureCookie bool) *Handler {
	return &Handler{
		db:           db,
		jwtManager:   jwtManager,
		flight:       flightGroup,
		secureCookie: secureCookie,
	}
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := refreshtoken.FromRequest(r)
	if err != nil {
		response.Unauthorized(w, response.BearerChallenge, "Invalid refresh token", err)
		return
	}

	// The rotation is shared with concurrent callers, so one client disconnecting must not abort it for the rest.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), rotateTimeout)
	defer cancel()

	tokens, err := flight.Do(ctx, h.flight, flightKeyPrefix+refreshToken, func(ctx context.Context) (response.TokenPair, error) {
		return h.rotate(refreshToken)
	})
	if err != nil {
		var rotateErr *rotateError
		if errors.As(err, &rotateErr) {
			if rotateErr.status == http.StatusUnauthorized {
				w.Header().Set("WWW-Authenticate", response.InvalidTokenChallenge)
			}
			response.JSON(w, false, rotateErr.message, rotateErr.status, rotateErr.err)
			return
		}
		response.JSON(w, false, "Failed to rotate refresh token", http.StatusInternalServerError, err)
		return
	}

	refreshtoken.SetCookie(w, tokens.RefreshToken, h.secureCookie)
	response.JSON(w, true, tokens, http.StatusOK, nil)
}

func (h *Handler) rotate(refreshToken string) (response.TokenPair, error) {
	claims, err := h.jwtManager.VerifyRefresh(refreshToken)
	if err != nil {
		return response.TokenPair{}, &rotateError{"Invalid session or expired refresh token", http.StatusUnauthorized, err}
	}

	userID := claims.Subject

	user, err := h.db.GetUserFromID(userID)
	if err != nil {
		return response.TokenPair{}, &rotateError{"User not found", http.StatusUnauthorized, err}
	}

	newRefreshToken, err := h.jwtManager.GenerateRefresh(userID, claims.DeviceID)
	if err != nil {
		return response.TokenPair{}, &rotateError{"Failed to rotate refresh token", http.StatusInternalServerError, err}
	}

	accessToken, err := h.jwtManager.GenerateAccess(newRefreshToken, user.Username)
	if err != nil {
		return response.TokenPair{}, &rotateError{"Access token generation failed", http.StatusInternalServerError, err}
	}

	return response.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (h *Handler) RefreshAccess(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := refreshtoken.FromRequest(r)
	if err != nil {
		response.Unauthorized(w, response.BearerChallenge, "Invalid refresh token", err)
		return
	}

	claims, err := h.jwtManager.VerifyRefresh(refreshToken)
	if err != nil {
		response.Unauthorized(w, response.InvalidTokenChallenge, "Invalid session or expired refresh token", err)
		return
	}

	user, err := h.db.GetUserFromID(claims.Subject)
	if err != nil {
		response.Unauthorized(w, response.InvalidTokenChallenge, "User not found", err)
		return
	}

	accessToken, err := h.jwtManager.GenerateAccess(refreshToken, user.Username)
	if err != nil {
		response.JSON(w, false, "Refresh failed", http.StatusInternalServerError, err)
		return
	}

	response.JSON(w, true, accessToken, http.StatusOK, nil)
}
