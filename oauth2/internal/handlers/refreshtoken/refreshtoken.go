package refreshtoken

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const CookieName = "refresh_token"

type Request struct {
	RefreshToken string `json:"refresh_token"`
}

func FromRequest(r *http.Request) (string, error) {
	cookie, err := r.Cookie(CookieName)
	if err == nil {
		return cookie.Value, nil
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return "", err
	}
	return req.RefreshToken, nil
}

func SetCookie(w http.ResponseWriter, token string, secure bool) {
	cookie := &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
	if expiresAt, ok := expiryOf(token); ok {
		cookie.Expires = expiresAt
	}
	http.SetCookie(w, cookie)
}

func expiryOf(token string) (time.Time, bool) {
	claims := &jwt.RegisteredClaims{}
	if _, _, err := jwt.NewParser().ParseUnverified(token, claims); err != nil || claims.ExpiresAt == nil {
		return time.Time{}, false
	}
	return claims.ExpiresAt.Time, true
}

func ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}
