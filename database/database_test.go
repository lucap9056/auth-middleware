package database

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lucap9056/auth-middleware/database/v2/schema"
)

func TestCreateSchema_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec(`SELECT pg_advisory_xact_lock`).
		WithArgs(schemaAdvisoryLockKey).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS auth_user_devices`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	if err := createSchema(db, schema.DefaultParams()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestCreateSchema_RollbackOnFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec(`SELECT pg_advisory_xact_lock`).
		WithArgs(schemaAdvisoryLockKey).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS auth_user_devices`).
		WillReturnError(errors.New("relation users does not exist"))
	mock.ExpectRollback()

	if err := createSchema(db, schema.DefaultParams()); err == nil {
		t.Fatal("expected error, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestCreateSchema_InvalidParams(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := createSchema(db, nil); err == nil {
		t.Fatal("expected error, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}
