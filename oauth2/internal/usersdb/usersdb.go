package usersdb

import (
	"database/sql"
	_ "embed"
	"errors"
	"strings"
	"text/template"

	"github.com/lucap9056/auth-middleware/database/v2"
	"github.com/lucap9056/auth-middleware/database/v2/schema"
)

const schemaAdvisoryLockKey int64 = 0x6175746875736572

//go:embed schema.sql
var schemaSQL string

var schemaTemplate = template.Must(template.New("schema.sql").Parse(schemaSQL))

func createSchema(db *sql.DB) error {
	params := schema.DefaultParams()
	var s strings.Builder
	if err := schemaTemplate.Execute(&s, params); err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("SELECT pg_advisory_xact_lock($1)", schemaAdvisoryLockKey); err != nil {
		return err
	}
	if _, err := tx.Exec(s.String()); err != nil {
		return err
	}
	return tx.Commit()
}

type Store struct {
	*database.Database
	db *sql.DB
}

func New(db *sql.DB, opts ...Option) (*Store, error) {
	cfg := newOptions(opts)

	store, err := newStore(db, cfg)
	if err != nil {
		return nil, err
	}

	databaseOptions := append(cfg.databaseOptions, database.WithAutoCreateSchema(cfg.autoCreateSchema))

	store.Database, err = database.New(db, databaseOptions...)
	if err != nil {
		return nil, err
	}
	return store, nil
}

func newStore(db *sql.DB, cfg *options) (*Store, error) {
	if cfg.autoCreateSchema {
		if err := createSchema(db); err != nil {
			return nil, err
		}
	} else if err := probe(db, "SELECT user_id, username, email FROM users LIMIT 0"); err != nil {
		return nil, err
	}

	return &Store{db: db}, nil
}

func probe(db *sql.DB, query string) error {
	rows, err := db.Query(query)
	if err != nil {
		return err
	}
	return rows.Close()
}

type User struct {
	ID       string
	Username string
	Email    string
}

func (s *Store) CreateUser(username, email string) (*User, error) {
	var user User
	err := s.db.QueryRow(
		"INSERT INTO users (username, email) VALUES ($1, $2) ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email RETURNING user_id, username, email",
		username, email,
	).Scan(&user.ID, &user.Username, &user.Email)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) GetUser(email string) (*User, error) {
	var user User
	err := s.db.QueryRow(
		"SELECT user_id, username, email FROM users WHERE email = $1",
		email,
	).Scan(&user.ID, &user.Username, &user.Email)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Store) GetUsername(email string) (string, error) {
	var username sql.NullString
	err := s.db.QueryRow("SELECT username FROM users WHERE email = $1", email).Scan(&username)
	if errors.Is(err, sql.ErrNoRows) {
		return "", database.ErrUserNotFound
	}
	if err != nil {
		return "", err
	}
	return username.String, nil
}

func (s *Store) DeleteUser(email string) error {
	_, err := s.db.Exec(
		"DELETE FROM users WHERE email = $1",
		email,
	)
	return err
}
