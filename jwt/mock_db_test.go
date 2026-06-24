package jwt

import "errors"

type MockDatabase struct {
	secrets map[string]string
}

func NewMockDatabase() *MockDatabase {
	return &MockDatabase{
		secrets: make(map[string]string),
	}
}

func (db *MockDatabase) UpdateDeviceSecret(deviceID, secret string) error {
	db.secrets[deviceID] = secret
	return nil
}

func (db *MockDatabase) GetDeviceSecret(deviceID string) (string, error) {
	secret, ok := db.secrets[deviceID]
	if !ok {
		return "", errors.New("device secret not found")
	}
	return secret, nil
}

type MockErrDatabase struct {
	updateErr error
	getErr    error
}

func (db *MockErrDatabase) UpdateDeviceSecret(_, _ string) error { return db.updateErr }
func (db *MockErrDatabase) GetDeviceSecret(_ string) (string, error) {
	return "", db.getErr
}
