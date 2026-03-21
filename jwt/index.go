package jwt

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken            = errors.New("invalid or expired token")
	ErrUnexpectedSigningMethod = errors.New("unexpected token signing method")
	ErrTypeAssertionFailed     = errors.New("failed to assert token claims")
)

type Database interface {
	UpdateDeviceSecret(deviceID, secret string) error
	GetDeviceSecret(deviceID string) (string, error)
}

type AccessClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	DeviceID string `json:"device_id"`
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	DeviceID string `json:"device_id"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	db     Database
	config *jwtOptions
}

func NewJWTManager(db Database, opts ...JWTOption) *JWTManager {
	cfg := defaultOptions()

	for _, opt := range opts {
		opt(cfg)
	}

	return &JWTManager{
		db:     db,
		config: cfg,
	}
}

func (m *JWTManager) GenerateRandomSecret() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (m *JWTManager) GenerateRefresh(userID, deviceID string, providedSecret ...string) (string, error) {
	var finalSecret string

	if len(providedSecret) > 0 && providedSecret[0] != "" {
		finalSecret = providedSecret[0]
	} else {
		finalSecret = m.GenerateRandomSecret()

		if err := m.db.UpdateDeviceSecret(deviceID, finalSecret); err != nil {
			return "", err
		}
	}

	claims := RefreshClaims{
		DeviceID: deviceID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.config.RefreshTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(finalSecret))
}

func (m *JWTManager) GenerateAccess(refreshToken, username string) (string, error) {
	parser := jwt.NewParser()
	unverifiedToken, _, err := parser.ParseUnverified(refreshToken, &RefreshClaims{})
	if err != nil {
		return "", ErrInvalidToken
	}

	claims, ok := unverifiedToken.Claims.(*RefreshClaims)
	if !ok {
		return "", ErrInvalidToken
	}

	storedSecret, err := m.db.GetDeviceSecret(claims.DeviceID)
	if err != nil {
		return "", ErrInvalidToken
	}

	token, err := jwt.ParseWithClaims(refreshToken, &RefreshClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrUnexpectedSigningMethod
		}
		return []byte(storedSecret), nil
	})

	if err != nil || !token.Valid {
		return "", ErrInvalidToken
	}

	accessClaims := AccessClaims{
		UserID:   claims.Subject,
		Username: username,
		DeviceID: claims.DeviceID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.config.AccessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(storedSecret))
}
func (m *JWTManager) VerifyAccess(accessToken string) (*AccessClaims, error) {
	return verifyToken(m, accessToken, &AccessClaims{})
}

func (m *JWTManager) VerifyRefresh(refreshToken string) (*RefreshClaims, error) {
	return verifyToken(m, refreshToken, &RefreshClaims{})
}

func verifyToken[T jwt.Claims](m *JWTManager, tokenStr string, claims T) (T, error) {
	parser := jwt.NewParser()

	_, _, err := parser.ParseUnverified(tokenStr, claims)
	if err != nil {
		return claims, ErrInvalidToken
	}

	var deviceID string
	switch c := any(claims).(type) {
	case *AccessClaims:
		deviceID = c.DeviceID
	case *RefreshClaims:
		deviceID = c.DeviceID
	default:
		return claims, ErrInvalidToken
	}

	storedSecret, err := m.db.GetDeviceSecret(deviceID)
	if err != nil {
		return claims, ErrInvalidToken
	}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrUnexpectedSigningMethod
		}
		return []byte(storedSecret), nil
	})

	if err != nil || !token.Valid {
		return claims, ErrInvalidToken
	}

	return claims, nil
}
