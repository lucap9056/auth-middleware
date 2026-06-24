package jwt

import (
	"errors"
	"testing"
	"time"
)

func TestJWTManager_GenerateRandomSecret(t *testing.T) {
	db := NewMockDatabase()
	manager := NewJWTManager(db)

	secret1 := manager.GenerateRandomSecret()
	secret2 := manager.GenerateRandomSecret()

	if secret1 == "" || secret2 == "" {
		t.Fatal("expected non-empty secrets")
	}

	if secret1 == secret2 {
		t.Fatal("expected random secrets to be different")
	}

	if len(secret1) != 64 { // 32 bytes hex encoded = 64 chars
		t.Errorf("expected secret length 64, got %d", len(secret1))
	}
}

func TestJWTManager_GenerateRefresh(t *testing.T) {
	db := NewMockDatabase()
	manager := NewJWTManager(db)

	userID := "user123"
	deviceID := "device456"

	token, err := manager.GenerateRefresh(userID, deviceID)
	if err != nil {
		t.Fatalf("failed to generate refresh token: %v", err)
	}

	if token == "" {
		t.Fatal("expected non-empty token")
	}

	secret, err := db.GetDeviceSecret(deviceID)
	if err != nil {
		t.Fatalf("failed to get device secret: %v", err)
	}
	if secret == "" {
		t.Fatal("expected non-empty device secret")
	}
}

func TestJWTManager_GenerateAccess(t *testing.T) {
	db := NewMockDatabase()
	manager := NewJWTManager(db)

	userID := "user123"
	deviceID := "device456"
	username := "testuser"

	refreshToken, err := manager.GenerateRefresh(userID, deviceID)
	if err != nil {
		t.Fatalf("failed to generate refresh token: %v", err)
	}

	accessToken, err := manager.GenerateAccess(refreshToken, username)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	if accessToken == "" {
		t.Fatal("expected non-empty access token")
	}
}

func TestJWTManager_VerifyAccess(t *testing.T) {
	db := NewMockDatabase()
	manager := NewJWTManager(db)

	userID := "user123"
	deviceID := "device456"
	username := "testuser"

	refreshToken, err := manager.GenerateRefresh(userID, deviceID)
	if err != nil {
		t.Fatalf("failed to generate refresh token: %v", err)
	}

	accessToken, err := manager.GenerateAccess(refreshToken, username)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	claims, err := manager.VerifyAccess(accessToken)
	if err != nil {
		t.Fatalf("failed to verify access token: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("expected user ID %s, got %s", userID, claims.UserID)
	}
	if claims.Username != username {
		t.Errorf("expected username %s, got %s", username, claims.Username)
	}
	if claims.DeviceID != deviceID {
		t.Errorf("expected device ID %s, got %s", deviceID, claims.DeviceID)
	}
}

func TestJWTManager_VerifyRefresh(t *testing.T) {
	db := NewMockDatabase()
	manager := NewJWTManager(db)

	userID := "user123"
	deviceID := "device456"

	refreshToken, err := manager.GenerateRefresh(userID, deviceID)
	if err != nil {
		t.Fatalf("failed to generate refresh token: %v", err)
	}

	claims, err := manager.VerifyRefresh(refreshToken)
	if err != nil {
		t.Fatalf("failed to verify refresh token: %v", err)
	}

	if claims.DeviceID != deviceID {
		t.Errorf("expected device ID %s, got %s", deviceID, claims.DeviceID)
	}
	if claims.Subject != userID {
		t.Errorf("expected subject %s, got %s", userID, claims.Subject)
	}
}

func TestJWTManager_GenerateRefresh_WithProvidedSecret(t *testing.T) {
	db := NewMockDatabase()
	manager := NewJWTManager(db)

	userID := "user1"
	deviceID := "dev1"
	secret := "my-fixed-secret"

	token, err := manager.GenerateRefresh(userID, deviceID, secret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if _, exists := db.secrets[deviceID]; exists {
		t.Error("provided secret path must not call UpdateDeviceSecret")
	}
}

func TestJWTManager_GenerateRefresh_UpdateSecretError(t *testing.T) {
	db := &MockErrDatabase{updateErr: errors.New("db down")}
	manager := NewJWTManager(db)

	_, err := manager.GenerateRefresh("user1", "dev1")
	if err == nil {
		t.Fatal("expected error when UpdateDeviceSecret fails")
	}
}

func TestJWTManager_GenerateAccess_InvalidRefreshToken(t *testing.T) {
	db := NewMockDatabase()
	manager := NewJWTManager(db)

	_, err := manager.GenerateAccess("not-a-jwt", "username")
	if err == nil {
		t.Fatal("expected error for unparseable refresh token")
	}
}

func TestJWTManager_GenerateAccess_DBGetError(t *testing.T) {
	db := &MockErrDatabase{getErr: errors.New("db down")}
	manager := NewJWTManager(db)

	// Build a structurally valid refresh token signed with a known secret so
	// ParseUnverified succeeds and we reach the GetDeviceSecret call.
	realDB := NewMockDatabase()
	realManager := NewJWTManager(realDB)
	refreshToken, _ := realManager.GenerateRefresh("user1", "dev1")

	_, err := manager.GenerateAccess(refreshToken, "username")
	if err == nil {
		t.Fatal("expected error when GetDeviceSecret fails")
	}
}

func TestJWTManager_VerifyAccess_TamperedToken(t *testing.T) {
	db := NewMockDatabase()
	manager := NewJWTManager(db)

	refreshToken, _ := manager.GenerateRefresh("user1", "dev1")
	accessToken, _ := manager.GenerateAccess(refreshToken, "username")

	db.secrets["dev1"] = "different-secret"

	_, err := manager.VerifyAccess(accessToken)
	if err == nil {
		t.Fatal("expected error when secret has changed (tampered)")
	}
}

func TestJWTManager_VerifyRefresh_TamperedToken(t *testing.T) {
	db := NewMockDatabase()
	manager := NewJWTManager(db)

	refreshToken, _ := manager.GenerateRefresh("user1", "dev1")

	db.secrets["dev1"] = "different-secret"

	_, err := manager.VerifyRefresh(refreshToken)
	if err == nil {
		t.Fatal("expected error when secret has changed (tampered)")
	}
}

func TestJWTManager_VerifyRefresh_ExpiredToken(t *testing.T) {
	db := NewMockDatabase()
	manager := NewJWTManager(db,
		WithRefreshTokenDuration(-time.Second),
	)

	token, err := manager.GenerateRefresh("user1", "dev1", "secret")
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}

	// secret is not in db yet — seed it so GetDeviceSecret succeeds
	db.secrets["dev1"] = "secret"

	_, err = manager.VerifyRefresh(token)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestJWTManager_VerifyAccess_InvalidString(t *testing.T) {
	db := NewMockDatabase()
	manager := NewJWTManager(db)

	_, err := manager.VerifyAccess("invalid-token")
	if err == nil {
		t.Fatal("expected error for invalid token string")
	}
}

func TestJWTManager_VerifyRefresh_MissingDeviceSecret(t *testing.T) {
	db := NewMockDatabase()
	manager := NewJWTManager(db)

	userID := "user123"
	deviceID := "device456"
	token, _ := manager.GenerateRefresh(userID, deviceID)

	delete(db.secrets, deviceID)

	_, err := manager.VerifyRefresh(token)
	if err == nil {
		t.Fatal("expected error when device secret is missing")
	}
}
