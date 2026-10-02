package jwt

import "errors"

var errDeviceNotFound = errors.New("device not found")

type mockDevice struct {
	secret     string
	generation int
}

type MockDatabase struct {
	devices   map[string]*mockDevice
	updateErr error
}

func NewMockDatabase() *MockDatabase {
	return &MockDatabase{
		devices: make(map[string]*mockDevice),
	}
}

func (db *MockDatabase) AddDevice(deviceID, secret string) {
	db.devices[deviceID] = &mockDevice{secret: secret, generation: 1}
}

func (db *MockDatabase) UpdateDeviceSecret(deviceID string) (int, error) {
	if db.updateErr != nil {
		return 0, db.updateErr
	}
	device, ok := db.devices[deviceID]
	if !ok {
		return 0, errDeviceNotFound
	}
	device.generation++
	return device.generation, nil
}

func (db *MockDatabase) GetDeviceSecret(deviceID string) (string, int, error) {
	device, ok := db.devices[deviceID]
	if !ok {
		return "", 0, errDeviceNotFound
	}
	return device.secret, device.generation, nil
}
