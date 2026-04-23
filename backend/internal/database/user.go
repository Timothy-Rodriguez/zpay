package database

import (
	"context"
	"fmt"
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
