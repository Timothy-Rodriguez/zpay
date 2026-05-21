package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"golang.org/x/crypto/bcrypt"
)

func (db *DB) CreateUser(ctx context.Context, email, password string) error {
	_, dbSpan := db.Tracer.Start(ctx, "db.create_user")
	defer dbSpan.End()

	dbSpan.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "INSERT"),
	)

	query := `
        INSERT INTO users (email, password)
        VALUES ($1, $2)
        ON CONFLICT (email) DO NOTHING
    `

	result, err := db.Pool.Exec(context.Background(), query, email, password)
	if err != nil {
		dbSpan.RecordError(err)
		dbSpan.SetStatus(codes.Error, "insert user failed")
		return fmt.Errorf("failed to create user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("user with email %s already exists", email)
	}

	dbSpan.SetStatus(codes.Ok, "insert user success")
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

// Get refresh token from database
func (db *DB) GetRefreshToken(ctx context.Context, email string) (string, error) {
	var token string
	query := `SELECT refresh_token FROM users WHERE email = $1`
	err := db.Pool.QueryRow(ctx, query, email).Scan(&token)
	if err != nil {
		return "", fmt.Errorf("failed to get refresh token: %w", err)
	}
	return token, nil
}

// Clear refresh token (logout)
func (db *DB) ClearRefreshToken(ctx context.Context, email string) error {
	query := `UPDATE users SET refresh_token = NULL WHERE email = $1`
	_, err := db.Pool.Exec(ctx, query, email)
	return err
}

// UserExists checks whether a user with the given email exists in the database.
func (db *DB) UserExists(ctx context.Context, email string) (bool, error) {
	_, dbSpan := db.Tracer.Start(ctx, "db.user_exists")
	defer dbSpan.End()

	dbSpan.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "SELECT"),
		attribute.String("user.email", email),
	)

	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
	err := db.Pool.QueryRow(ctx, query, email).Scan(&exists)
	if err != nil {
		dbSpan.RecordError(err)
		dbSpan.SetStatus(codes.Error, "user_exists query failed")
		return false, fmt.Errorf("failed to check user existence: %w", err)
	}

	dbSpan.SetStatus(codes.Ok, "user_exists query success")
	return exists, nil
}
