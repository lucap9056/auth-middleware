package database

import (
	"database/sql"
)

type User struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

func (d *Database) CreateUser(username, email string) (*User, error) {
	query := `
	INSERT INTO users (username, email)
	VALUES ($1, $2)
	RETURNING user_id;
	`
	var userID string
	err := d.db.QueryRow(query, username, email).Scan(&userID)
	if err != nil {
		return nil, err
	}
	return &User{UserID: userID, Username: username, Email: email}, nil
}

func (d *Database) GetUserFromEmail(email string) (*User, error) {
	var user User
	query := `
	SELECT user_id, username, email
	FROM users
	WHERE email = $1;
	`
	err := d.db.QueryRow(query, email).Scan(&user.UserID, &user.Username, &user.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (d *Database) GetUserFromID(userID string) (*User, error) {
	var user User
	query := `
	SELECT user_id, username, email
	FROM users
	WHERE user_id = $1;
	`
	err := d.db.QueryRow(query, userID).Scan(&user.UserID, &user.Username, &user.Email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (d *Database) UpdateUsername(userID string, newUsername string) error {
	query := `
	UPDATE users
	SET username = $1
	WHERE user_id = $2;
	`
	_, err := d.db.Exec(query, newUsername, userID)
	return err
}

func (d *Database) DeleteUser(userID string) error {
	query := `
	DELETE FROM users
	WHERE user_id = $1;
	`
	_, err := d.db.Exec(query, userID)
	return err
}
