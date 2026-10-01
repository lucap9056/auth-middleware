package refreshtoken

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func signedToken(t *testing.T, expiresAt time.Time) string {
	t.Helper()
	claims := jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(expiresAt)}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return token
}

func TestSetCookie_ExpiresWithToken(t *testing.T) {
	expiresAt := time.Now().Add(30 * 24 * time.Hour).Truncate(time.Second)
	w := httptest.NewRecorder()

	SetCookie(w, signedToken(t, expiresAt), true)

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	if !cookies[0].Expires.Equal(expiresAt) {
		t.Errorf("cookie expires at %v, want %v", cookies[0].Expires, expiresAt)
	}
}

func TestSetCookie_UnparseableTokenBecomesSessionCookie(t *testing.T) {
	w := httptest.NewRecorder()

	SetCookie(w, "not-a-jwt", true)

	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}
	if !cookies[0].Expires.IsZero() {
		t.Errorf("expected session cookie without expiry, got %v", cookies[0].Expires)
	}
}
