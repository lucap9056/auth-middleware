package database

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
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

func TestCreateUser_Success(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectQuery(`INSERT INTO users`).
		WithArgs("alice", "alice@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow("uid-1"))

	user, err := d.CreateUser("alice", "alice@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.UserID != "uid-1" || user.Username != "alice" || user.Email != "alice@example.com" {
		t.Errorf("unexpected user: %+v", user)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestCreateUser_Error(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectQuery(`INSERT INTO users`).
		WithArgs("alice", "alice@example.com").
		WillReturnError(errors.New("insert failed"))

	if _, err := d.CreateUser("alice", "alice@example.com"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetUserFromEmail_Found(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectQuery(`SELECT user_id, username, email`).
		WithArgs("alice@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "username", "email"}).
			AddRow("uid-1", "alice", "alice@example.com"))

	user, err := d.GetUserFromEmail("alice@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user == nil || user.UserID != "uid-1" {
		t.Errorf("unexpected user: %+v", user)
	}
}

func TestGetUserFromEmail_NotFound(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectQuery(`SELECT user_id, username, email`).
		WithArgs("nobody@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "username", "email"}))

	user, err := d.GetUserFromEmail("nobody@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user != nil {
		t.Errorf("expected nil user, got %+v", user)
	}
}

func TestGetUserFromEmail_Error(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectQuery(`SELECT user_id, username, email`).
		WithArgs("alice@example.com").
		WillReturnError(errors.New("db error"))

	if _, err := d.GetUserFromEmail("alice@example.com"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetUserFromID_Found(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectQuery(`SELECT user_id, username, email`).
		WithArgs("uid-1").
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "username", "email"}).
			AddRow("uid-1", "alice", "alice@example.com"))

	user, err := d.GetUserFromID("uid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user == nil || user.UserID != "uid-1" {
		t.Errorf("unexpected user: %+v", user)
	}
}

func TestGetUserFromID_NotFound(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectQuery(`SELECT user_id, username, email`).
		WithArgs("nonexistent").
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "username", "email"}))

	user, err := d.GetUserFromID("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user != nil {
		t.Errorf("expected nil user, got %+v", user)
	}
}

func TestGetUserFromID_Error(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectQuery(`SELECT user_id, username, email`).
		WithArgs("uid-1").
		WillReturnError(errors.New("db error"))

	if _, err := d.GetUserFromID("uid-1"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUpdateUsername_Success(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectExec(`UPDATE users`).
		WithArgs("bob", "uid-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := d.UpdateUsername("uid-1", "bob"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateUsername_Error(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectExec(`UPDATE users`).
		WithArgs("bob", "uid-1").
		WillReturnError(errors.New("update failed"))

	if err := d.UpdateUsername("uid-1", "bob"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDeleteUser_Success(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectExec(`DELETE FROM users`).
		WithArgs("uid-1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := d.DeleteUser("uid-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteUser_Error(t *testing.T) {
	d, mock := newMockDB(t)

	mock.ExpectExec(`DELETE FROM users`).
		WithArgs("uid-1").
		WillReturnError(errors.New("delete failed"))

	if err := d.DeleteUser("uid-1"); err == nil {
		t.Fatal("expected error, got nil")
	}
}
