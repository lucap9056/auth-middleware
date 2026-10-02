package jwt

import (
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
	UpdateDeviceSecret(deviceID string) (int, error)
	GetDeviceSecret(deviceID string) (string, int, error)
}

type AccessClaims struct {
	Username   string `json:"username"`
	UserEmail  string `json:"user_email"`
	DeviceID   string `json:"device_id"`
	Generation int    `json:"generation"`
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	DeviceID   string `json:"device_id"`
	Generation int    `json:"generation"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	db     Database
	config *options
}

func NewJWTManager(db Database, opts ...Option) *JWTManager {
	cfg := defaultOptions()

	for _, opt := range opts {
		opt(cfg)
	}

	return &JWTManager{
		db:     db,
		config: cfg,
	}
}

func (m *JWTManager) GenerateRefresh(userEmail, deviceID, secret string, gen int) (string, error) {
	return jwt.NewWithClaims(jwt.SigningMethodHS256, m.newRefreshClaims(userEmail, deviceID, gen)).SignedString([]byte(secret))
}

func (m *JWTManager) newRefreshClaims(userEmail, deviceID string, gen int) *RefreshClaims {
	return &RefreshClaims{
		DeviceID:   deviceID,
		Generation: gen,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.config.Issuer,
			Audience:  jwt.ClaimStrings{m.config.Audience},
			Subject:   userEmail,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.config.RefreshTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
}

func (m *JWTManager) GenerateAccess(refreshToken, username string) (string, error) {
	claims, secret, err := verifyToken(m, refreshToken, &RefreshClaims{})
	if err != nil {
		return "", err
	}

	accessClaims := AccessClaims{
		Username:   username,
		UserEmail:  claims.Subject,
		DeviceID:   claims.DeviceID,
		Generation: claims.Generation,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.config.Issuer,
			Audience:  jwt.ClaimStrings{m.config.Audience},
			Subject:   claims.Subject,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.config.AccessTokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(secret))
}

func (m *JWTManager) VerifyAccess(accessToken string) (*AccessClaims, error) {
	claims, _, err := verifyToken(m, accessToken, &AccessClaims{})
	return claims, err
}

func (m *JWTManager) VerifyRefresh(refreshToken string) (*RefreshClaims, error) {
	claims, _, err := verifyToken(m, refreshToken, &RefreshClaims{})
	return claims, err
}

func (m *JWTManager) RotateRefresh(refreshToken string) (string, *RefreshClaims, error) {
	claims, secret, err := verifyToken(m, refreshToken, &RefreshClaims{})
	if err != nil {
		return "", claims, err
	}

	gen, err := m.db.UpdateDeviceSecret(claims.DeviceID)
	if err != nil {
		return "", claims, err
	}

	newClaims := m.newRefreshClaims(claims.Subject, claims.DeviceID, gen)
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims).SignedString([]byte(secret))
	return token, newClaims, err
}

func (m *JWTManager) parserOptions() []jwt.ParserOption {
	var opts []jwt.ParserOption
	if m.config.Issuer != "" {
		opts = append(opts, jwt.WithIssuer(m.config.Issuer))
	}
	if m.config.Audience != "" {
		opts = append(opts, jwt.WithAudience(m.config.Audience))
	}
	return opts
}

func verifyToken[T jwt.Claims](m *JWTManager, tokenStr string, claims T) (T, string, error) {
	parser := jwt.NewParser()

	_, _, err := parser.ParseUnverified(tokenStr, claims)
	if err != nil {
		return claims, "", ErrInvalidToken
	}

	var deviceID string
	var generation int
	switch c := any(claims).(type) {
	case *AccessClaims:
		deviceID = c.DeviceID
		generation = c.Generation
	case *RefreshClaims:
		deviceID = c.DeviceID
		generation = c.Generation
	default:
		return claims, "", ErrInvalidToken
	}

	secret, gen, err := m.db.GetDeviceSecret(deviceID)
	if err != nil {
		return claims, "", ErrInvalidToken
	}

	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrUnexpectedSigningMethod
		}
		return []byte(secret), nil
	}, m.parserOptions()...)

	if err != nil || !token.Valid {
		return claims, "", ErrInvalidToken
	}

	if generation != gen {
		return claims, "", ErrInvalidToken
	}

	return claims, secret, nil
}
