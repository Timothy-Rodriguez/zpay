package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

func (db *DB) UpdateBalace(email string, amount decimal.Decimal) error {
	query := `
        INSERT INTO accounts (email, balance)
        VALUES ($1, $2)
        ON CONFLICT (email)
        DO UPDATE SET
            balance = EXCLUDED.balance,
            updated_at = CURRENT_TIMESTAMP
    `

	if _, err := db.Pool.Exec(context.Background(), query, email, amount.String()); err != nil {
		return fmt.Errorf("failed to update balance for %s: %w", email, err)
	}

	return nil
}

func (db *DB) ProcessTransaction(
	ctx context.Context,
	fromEmail string,
	toEmail string,
	amount decimal.Decimal,
) error {
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get from account ID
	var fromAccountID int
	err = tx.QueryRow(
		ctx,
		`SELECT id FROM accounts WHERE email = $1`,
		fromEmail,
	).Scan(&fromAccountID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("from account not found")
		}
		return fmt.Errorf("failed to fetch from account: %w", err)
	}

	// Get to account ID
	var toAccountID int
	err = tx.QueryRow(
		ctx,
		`SELECT id FROM accounts WHERE email = $1`,
		toEmail,
	).Scan(&toAccountID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("to account not found")
		}
		return fmt.Errorf("failed to fetch to account: %w", err)
	}

	// Check if from account has sufficient balance (with lock)
	var fromBalance decimal.Decimal
	err = tx.QueryRow(
		ctx,
		`SELECT balance FROM accounts WHERE id = $1 FOR UPDATE`,
		fromAccountID,
	).Scan(&fromBalance)
	if err != nil {
		return fmt.Errorf("failed to fetch from account balance: %w", err)
	}

	if fromBalance.LessThan(amount) {
		return fmt.Errorf("insufficient balance")
	}

	// Debit from account
	_, err = tx.Exec(
		ctx,
		`UPDATE accounts SET balance = balance - $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`,
		amount.String(),
		fromAccountID,
	)
	if err != nil {
		return fmt.Errorf("failed to debit from account: %w", err)
	}

	// Credit to account
	_, err = tx.Exec(
		ctx,
		`UPDATE accounts SET balance = balance + $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`,
		amount.String(),
		toAccountID,
	)
	if err != nil {
		return fmt.Errorf("failed to credit to account: %w", err)
	}

	// Record transaction
	_, err = tx.Exec(
		ctx,
		`INSERT INTO transactions (from_account, to_account, amount) VALUES ($1, $2, $3)`,
		fromAccountID,
		toAccountID,
		amount.String(),
	)
	if err != nil {
		return fmt.Errorf("failed to record transaction: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
