package database

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestSaveDeviceSecret_Success(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectQuery(`INSERT INTO user_devices`).
		WithArgs("My Phone", "uid-1", "secret-abc").
		WillReturnRows(sqlmock.NewRows([]string{"device_id"}).AddRow("dev-1"))

	deviceID, err := d.SaveDeviceSecret("uid-1", "My Phone", "secret-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deviceID != "dev-1" {
		t.Errorf("deviceID: got %q, want %q", deviceID, "dev-1")
	}
}

func TestSaveDeviceSecret_Error(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectQuery(`INSERT INTO user_devices`).
		WithArgs("My Phone", "uid-1", "secret-abc").
		WillReturnError(errors.New("insert failed"))

	if _, err := d.SaveDeviceSecret("uid-1", "My Phone", "secret-abc"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUpdateDeviceSecret_Success(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectExec(`UPDATE user_devices`).
		WithArgs("new-secret", "dev-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := d.UpdateDeviceSecret("dev-1", "new-secret"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateDeviceSecret_Error(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectExec(`UPDATE user_devices`).
		WithArgs("new-secret", "dev-1").
		WillReturnError(errors.New("update failed"))

	if err := d.UpdateDeviceSecret("dev-1", "new-secret"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetDeviceSecret_Success(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectQuery(`SELECT secret FROM user_devices`).
		WithArgs("dev-1").
		WillReturnRows(sqlmock.NewRows([]string{"secret"}).AddRow("secret-abc"))

	secret, err := d.GetDeviceSecret("dev-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if secret != "secret-abc" {
		t.Errorf("secret: got %q, want %q", secret, "secret-abc")
	}
}

func TestGetDeviceSecret_NotFound(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectQuery(`SELECT secret FROM user_devices`).
		WithArgs("dev-nonexistent").
		WillReturnRows(sqlmock.NewRows([]string{"secret"}))

	if _, err := d.GetDeviceSecret("dev-nonexistent"); err == nil {
		t.Fatal("expected error for missing device, got nil")
	}
}

func TestGetDeviceSecret_Error(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectQuery(`SELECT secret FROM user_devices`).
		WithArgs("dev-1").
		WillReturnError(errors.New("db error"))

	if _, err := d.GetDeviceSecret("dev-1"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteDevice_Success(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectExec(`DELETE FROM user_devices`).
		WithArgs("uid-1", "dev-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := d.DeleteDevice("uid-1", "dev-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteDevice_Error(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectExec(`DELETE FROM user_devices`).
		WithArgs("uid-1", "dev-1").
		WillReturnError(errors.New("delete failed"))

	if err := d.DeleteDevice("uid-1", "dev-1"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteAllDevices_Success(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectExec(`DELETE FROM user_devices`).
		WithArgs("uid-1").
		WillReturnResult(sqlmock.NewResult(0, 3))

	if err := d.DeleteAllDevices("uid-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteAllDevices_Error(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectExec(`DELETE FROM user_devices`).
		WithArgs("uid-1").
		WillReturnError(errors.New("delete failed"))

	if err := d.DeleteAllDevices("uid-1"); err == nil {
		t.Fatal("expected error, got nil")
	}
}
