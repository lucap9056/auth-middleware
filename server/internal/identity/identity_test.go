package identity

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "0123456789abcdef0123456789abcdef"

func parse(t *testing.T, tokenStr string, opts ...jwt.ParserOption) (*jwt.Token, *Claims) {
	t.Helper()
	claims := &Claims{}
	opts = append(opts, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(*jwt.Token) (any, error) {
		return []byte(testSecret), nil
	}, opts...)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return token, claims
}

func TestSign_Claims(t *testing.T) {
	signer := NewSigner(testSecret, "https://auth.example.com", "api")

	tokenStr, err := signer.Sign("user@example.com", "User", "device-1", jwt.NewNumericDate(time.Now().Add(time.Hour)))
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	token, claims := parse(t, tokenStr, jwt.WithIssuer("https://auth.example.com"), jwt.WithAudience("api"))
	if typ := token.Header["typ"]; typ != TokenType {
		t.Errorf("typ: got %v, want %q", typ, TokenType)
	}
	if claims.Subject != "user@example.com" || claims.Username != "User" || claims.DeviceID != "device-1" {
		t.Errorf("claims: got %+v", claims)
	}
	if lifetime := claims.ExpiresAt.Sub(claims.IssuedAt.Time); lifetime != TokenDuration {
		t.Errorf("lifetime: got %v, want %v", lifetime, TokenDuration)
	}
}

func TestSign_ExpiryCappedByAccessToken(t *testing.T) {
	signer := NewSigner(testSecret, "", "")
	accessExpiresAt := time.Now().Add(20 * time.Second).Truncate(time.Second)

	tokenStr, err := signer.Sign("user@example.com", "", "device-1", jwt.NewNumericDate(accessExpiresAt))
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	_, claims := parse(t, tokenStr)
	if !claims.ExpiresAt.Equal(accessExpiresAt) {
		t.Errorf("exp: got %v, want %v", claims.ExpiresAt.Time, accessExpiresAt)
	}
}

func TestSign_OmitsEmptyOptionalClaims(t *testing.T) {
	signer := NewSigner(testSecret, "", "")

	tokenStr, err := signer.Sign("user@example.com", "", "device-1", nil)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	raw := jwt.MapClaims{}
	if _, _, err := jwt.NewParser().ParseUnverified(tokenStr, raw); err != nil {
		t.Fatalf("ParseUnverified: %v", err)
	}
	for _, key := range []string{"username", "iss", "aud"} {
		if _, ok := raw[key]; ok {
			t.Errorf("claim %q: want omitted, got %v", key, raw[key])
		}
	}
}

func TestSign_WrongSecretRejected(t *testing.T) {
	tokenStr, err := NewSigner(testSecret, "", "").Sign("user@example.com", "", "device-1", nil)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}

	_, err = jwt.Parse(tokenStr, func(*jwt.Token) (any, error) {
		return []byte("another-secret-another-secret-xx"), nil
	})
	if err == nil {
		t.Error("want signature error with wrong secret")
	}
}
