package refreshtoken

import (
	"encoding/json"
	"net/http"
	"time"
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
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		Expires:  time.Now().Add(7 * 24 * time.Hour),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
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
