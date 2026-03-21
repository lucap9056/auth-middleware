package jwt

import (
	"testing"
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

func TestJWTManager_InvalidTokens(t *testing.T) {
	db := NewMockDatabase()
	manager := NewJWTManager(db)

	t.Run("invalid token string", func(t *testing.T) {
		_, err := manager.VerifyAccess("invalid-token")
		if err == nil {
			t.Fatal("expected error for invalid token")
		}
	})

	t.Run("non-existent device secret", func(t *testing.T) {
		// Use a token that has a device ID but the secret isn't in the DB
		// This is tricky because we need a signed token.
		// Let's just manually forge a token if needed, or use a valid one and clear DB.
		userID := "user123"
		deviceID := "device456"
		token, _ := manager.GenerateRefresh(userID, deviceID)

		delete(db.secrets, deviceID)

		_, err := manager.VerifyRefresh(token)
		if err == nil {
			t.Fatal("expected error when device secret is missing")
		}
	})
}
