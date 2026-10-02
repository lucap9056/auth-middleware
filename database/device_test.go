package database

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
)

func newMockDB(t *testing.T) (*Database, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		_ = db.Close()
		cancel()
	})
	return &Database{db: db, ctx: ctx, cancel: cancel}, mock
}

func TestSaveDeviceSecret_Success(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectQuery(`INSERT INTO auth_user_devices`).
		WithArgs("My Phone", "user@example.com", "secret-abc").
		WillReturnRows(sqlmock.NewRows([]string{"device_id"}).AddRow("dev-1"))

	deviceID, err := d.SaveDeviceSecret("user@example.com", "My Phone", "secret-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deviceID != "dev-1" {
		t.Errorf("deviceID: got %q, want %q", deviceID, "dev-1")
	}
}

func TestSaveDeviceSecret_UnknownUser(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectQuery(`INSERT INTO auth_user_devices`).
		WithArgs("My Phone", "nobody@example.com", "secret-abc").
		WillReturnError(&pgconn.PgError{Code: "23503", ConstraintName: "fk_auth_user_devices"})

	if _, err := d.SaveDeviceSecret("nobody@example.com", "My Phone", "secret-abc"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("err: got %v, want ErrUserNotFound", err)
	}
}

func TestUpdateDeviceSecret_Success(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectExec(`UPDATE auth_user_devices`).
		WithArgs("new-secret", "dev-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := d.UpdateDeviceSecret("dev-1", "new-secret"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetDeviceSecret_Success(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectQuery(`SELECT secret FROM auth_user_devices`).
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

	mock.ExpectQuery(`SELECT secret FROM auth_user_devices`).
		WithArgs("dev-nonexistent").
		WillReturnRows(sqlmock.NewRows([]string{"secret"}))

	if _, err := d.GetDeviceSecret("dev-nonexistent"); err == nil {
		t.Fatal("expected error for missing device, got nil")
	}
}

func TestDeleteDevice_Success(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectExec(`DELETE FROM auth_user_devices`).
		WithArgs("user@example.com", "dev-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := d.DeleteDevice("user@example.com", "dev-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteAllDevices_Success(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectExec(`DELETE FROM auth_user_devices`).
		WithArgs("user@example.com").
		WillReturnResult(sqlmock.NewResult(0, 3))

	if err := d.DeleteAllDevices("user@example.com"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteAllDevicesReturningIDs_Success(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectQuery(`DELETE FROM auth_user_devices WHERE user_email = \$1 RETURNING device_id`).
		WithArgs("user@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"device_id"}).AddRow("dev-1").AddRow("dev-2"))

	deviceIDs, err := d.DeleteAllDevicesReturningIDs("user@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deviceIDs) != 2 || deviceIDs[0] != "dev-1" || deviceIDs[1] != "dev-2" {
		t.Errorf("deviceIDs: got %v, want [dev-1 dev-2]", deviceIDs)
	}
}

func TestDeleteAllDevicesReturningIDs_NoDevices(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectQuery(`DELETE FROM auth_user_devices`).
		WithArgs("user@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"device_id"}))

	deviceIDs, err := d.DeleteAllDevicesReturningIDs("user@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deviceIDs) != 0 {
		t.Errorf("deviceIDs: got %v, want empty", deviceIDs)
	}
}

func TestDeleteAllDevicesReturningIDs_RowError(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectQuery(`DELETE FROM auth_user_devices`).
		WithArgs("user@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"device_id"}).
			AddRow("dev-1").
			AddRow("dev-2").
			RowError(1, errors.New("row failed")))

	if _, err := d.DeleteAllDevicesReturningIDs("user@example.com"); err == nil {
		t.Fatal("expected error, got nil")
	}
}
