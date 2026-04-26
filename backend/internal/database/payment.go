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

	// Get both account IDs
	var fromAccountID, toAccountID int
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

	// Lock both accounts in deterministic order (by ID)
	// Always lock the lower ID first
	firstAccountID := fromAccountID
	secondAccountID := toAccountID
	if firstAccountID > secondAccountID {
		firstAccountID, secondAccountID = secondAccountID, firstAccountID
	}

	// Fetch both balances with locks in deterministic order
	var fromBalance, toBalance decimal.Decimal

	err = tx.QueryRow(
		ctx,
		`SELECT balance FROM accounts WHERE id = $1 FOR UPDATE`,
		firstAccountID,
	).Scan(&fromBalance)
	if err != nil {
		return fmt.Errorf("failed to fetch first account balance: %w", err)
	}

	err = tx.QueryRow(
		ctx,
		`SELECT balance FROM accounts WHERE id = $1 FOR UPDATE`,
		secondAccountID,
	).Scan(&toBalance)
	if err != nil {
		return fmt.Errorf("failed to fetch second account balance: %w", err)
	}

	// Reassign based on original from/to relationship
	if fromAccountID > toAccountID {
		fromBalance, toBalance = toBalance, fromBalance
	}

	// Check if from account has sufficient balance
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
	var transactionID int
	err = tx.QueryRow(
		ctx,
		`INSERT INTO payments (from_account, to_account, amount) 
         VALUES ($1, $2, $3) RETURNING id`,
		fromAccountID,
		toAccountID,
		amount.String(),
	).Scan(&transactionID)
	if err != nil {
		return fmt.Errorf("failed to record transaction: %w", err)
	}

	// Insert into payment outbox
	payload := fmt.Sprintf(
		`{"transactionId": %d, "fromAccount": %d, "toAccount": %d, "amount": "%s", "fromEmail": "%s", "toEmail": "%s"}`,
		transactionID, fromAccountID, toAccountID, amount.String(), fromEmail, toEmail,
	)

	_, err = tx.Exec(
		ctx,
		`INSERT INTO payment_outbox (transaction_id, from_account, to_account, amount, event_type, status, payload)
         VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		transactionID,
		fromAccountID,
		toAccountID,
		amount.String(),
		"PAYMENT_COMPLETED",
		"PENDING",
		payload,
	)
	if err != nil {
		return fmt.Errorf("failed to insert outbox record: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
