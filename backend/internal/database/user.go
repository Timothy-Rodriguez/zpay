package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

func (db *DB) CreateUser(email, password string) error {
	query := `
        INSERT INTO users (email, password)
        VALUES ($1, $2)
        ON CONFLICT (email) DO NOTHING
    `

	result, err := db.Pool.Exec(context.Background(), query, email, password)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user with email %s already exists", email)
	}

	return nil
}

func (db *DB) CheckLoginAndStoreRefreshToken(ctx context.Context, email string, password string, refreshToken string) (bool, error) {
	// Step 1: Get stored password
	getQuery := `
        SELECT password FROM users WHERE email = $1
    `

	var storedPassword string
	err := db.Pool.QueryRow(ctx, getQuery, email).Scan(&storedPassword)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to get password: %w", err)
	}

	// Step 2: Verify password matches
	if err := bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password)); err != nil {
		return false, nil // Wrong password, don't update token
	}

	// Step 3: Update refresh token ONLY if password was correct
	updateQuery := `
        UPDATE users SET refresh_token = $1, updated_at = CURRENT_TIMESTAMP WHERE email = $2
    `

	result, err := db.Pool.Exec(ctx, updateQuery, refreshToken, email)
	if err != nil {
		return false, fmt.Errorf("failed to update refresh token: %w", err)
	}

	if result.RowsAffected() == 0 {
		return false, fmt.Errorf("user not found")
	}

	return true, nil
}
