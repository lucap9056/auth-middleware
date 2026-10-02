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

func TestUpdateDeviceSecret_Success(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectExec(`UPDATE user_devices`).
		WithArgs("new-secret", "dev-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := d.UpdateDeviceSecret("dev-1", "new-secret"); err != nil {
		t.Fatalf("unexpected error: %v", err)
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

func TestDeleteDevice_Success(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectExec(`DELETE FROM user_devices`).
		WithArgs("uid-1", "dev-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := d.DeleteDevice("uid-1", "dev-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
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

func TestDeleteAllDevicesReturningIDs_Success(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectQuery(`DELETE FROM user_devices WHERE user_id = \$1 RETURNING device_id`).
		WithArgs("uid-1").
		WillReturnRows(sqlmock.NewRows([]string{"device_id"}).AddRow("dev-1").AddRow("dev-2"))

	deviceIDs, err := d.DeleteAllDevicesReturningIDs("uid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deviceIDs) != 2 || deviceIDs[0] != "dev-1" || deviceIDs[1] != "dev-2" {
		t.Errorf("deviceIDs: got %v, want [dev-1 dev-2]", deviceIDs)
	}
}

func TestDeleteAllDevicesReturningIDs_NoDevices(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectQuery(`DELETE FROM user_devices`).
		WithArgs("uid-1").
		WillReturnRows(sqlmock.NewRows([]string{"device_id"}))

	deviceIDs, err := d.DeleteAllDevicesReturningIDs("uid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deviceIDs) != 0 {
		t.Errorf("deviceIDs: got %v, want empty", deviceIDs)
	}
}

func TestDeleteAllDevicesReturningIDs_RowError(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectQuery(`DELETE FROM user_devices`).
		WithArgs("uid-1").
		WillReturnRows(sqlmock.NewRows([]string{"device_id"}).
			AddRow("dev-1").
			AddRow("dev-2").
			RowError(1, errors.New("row failed")))

	if _, err := d.DeleteAllDevicesReturningIDs("uid-1"); err == nil {
		t.Fatal("expected error, got nil")
	}
}
