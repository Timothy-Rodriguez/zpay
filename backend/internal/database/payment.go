package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

func (db *DB) UpdateBalace(ctx context.Context, email string, amount decimal.Decimal) error {
	_, dbSpan := db.Tracer.Start(ctx, "db.init_user_balance")
	defer dbSpan.End()

	dbSpan.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "UPDATE"),
	)

	query := `
        INSERT INTO accounts (email, balance)
        VALUES ($1, $2)
        ON CONFLICT (email)
        DO UPDATE SET
            balance = EXCLUDED.balance,
            updated_at = CURRENT_TIMESTAMP
    `

	if _, err := db.Pool.Exec(context.Background(), query, email, amount.String()); err != nil {
		dbSpan.RecordError(err)
		dbSpan.SetStatus(codes.Error, "update balance failed")
		return fmt.Errorf("failed to update balance for %s: %w", email, err)
	}

	dbSpan.SetStatus(codes.Ok, "update balance success")
	return nil
}

func (db *DB) ProcessTransaction(
	ctx context.Context,
	fromEmail string,
	toEmail string,
	amount decimal.Decimal,
) error {
	_, dbSpan := db.Tracer.Start(ctx, "db.init_user_transaction")
	defer dbSpan.End()

	dbSpan.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "UPDATE"),
	)

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		dbSpan.RecordError(err)
		dbSpan.SetStatus(codes.Error, "failed to begin transaction")
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
			dbSpan.RecordError(err)
			dbSpan.SetStatus(codes.Error, "from account not found")
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
			dbSpan.RecordError(err)
			dbSpan.SetStatus(codes.Error, "to account not found")
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
		dbSpan.RecordError(err)
		dbSpan.SetStatus(codes.Error, "failed to fetch first account balance")
		return fmt.Errorf("failed to fetch first account balance: %w", err)
	}

	err = tx.QueryRow(
		ctx,
		`SELECT balance FROM accounts WHERE id = $1 FOR UPDATE`,
		secondAccountID,
	).Scan(&toBalance)
	if err != nil {
		dbSpan.RecordError(err)
		dbSpan.SetStatus(codes.Error, "failed to fetch second account balance")
		return fmt.Errorf("failed to fetch second account balance: %w", err)
	}

	// Reassign based on original from/to relationship
	if fromAccountID > toAccountID {
		fromBalance, toBalance = toBalance, fromBalance
	}

	// Check if from account has sufficient balance
	if fromBalance.LessThan(amount) {
		dbSpan.RecordError(fmt.Errorf("insufficient balance"))
		dbSpan.SetStatus(codes.Error, "insufficient balance")
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
		dbSpan.RecordError(err)
		dbSpan.SetStatus(codes.Error, "failed to debit from account")
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
		dbSpan.RecordError(err)
		dbSpan.SetStatus(codes.Error, "failed to credit to account")
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
		dbSpan.RecordError(err)
		dbSpan.SetStatus(codes.Error, "failed to record transaction")
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
		dbSpan.RecordError(err)
		dbSpan.SetStatus(codes.Error, "failed to insert outbox record")
		return fmt.Errorf("failed to insert outbox record: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		dbSpan.RecordError(err)
		dbSpan.SetStatus(codes.Error, "failed to commit transaction")
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	dbSpan.SetStatus(codes.Ok, "payment success")
	return nil
}

// GetBalance returns the account balance for the given email.
func (db *DB) GetBalance(ctx context.Context, email string) (decimal.Decimal, error) {
	var balance decimal.Decimal
	query := `SELECT balance FROM accounts WHERE email = $1`
	err := db.Pool.QueryRow(ctx, query, email).Scan(&balance)
	if err != nil {
		if err == pgx.ErrNoRows {
			return decimal.Zero, fmt.Errorf("account not found for email %s", email)
		}
		return decimal.Zero, fmt.Errorf("failed to get balance: %w", err)
	}
	return balance, nil
}

// AccountSummary represents a user account with its balance.
type AccountSummary struct {
	Email   string          `json:"email"`
	Balance decimal.Decimal `json:"balance"`
}

// GetAccounts returns up to limit accounts ordered by creation time.
func (db *DB) GetAccounts(ctx context.Context, limit int) ([]AccountSummary, error) {
	_, dbSpan := db.Tracer.Start(ctx, "db.get_accounts")
	defer dbSpan.End()

	dbSpan.SetAttributes(
		attribute.String("db.system", "postgresql"),
		attribute.String("db.operation", "SELECT"),
	)

	query := `SELECT email, balance FROM accounts ORDER BY created_at ASC LIMIT $1`
	rows, err := db.Pool.Query(ctx, query, limit)
	if err != nil {
		dbSpan.RecordError(err)
		dbSpan.SetStatus(codes.Error, "get_accounts query failed")
		return nil, fmt.Errorf("failed to get accounts: %w", err)
	}
	defer rows.Close()

	accounts := make([]AccountSummary, 0, limit)
	for rows.Next() {
		var a AccountSummary
		if err := rows.Scan(&a.Email, &a.Balance); err != nil {
			return nil, fmt.Errorf("failed to scan account: %w", err)
		}
		accounts = append(accounts, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	dbSpan.SetStatus(codes.Ok, "get_accounts success")
	return accounts, nil
}

// Transaction represents one payment row joined to account emails.
type Transaction struct {
	ID        int             `json:"id"`
	FromEmail string          `json:"from_email"`
	ToEmail   string          `json:"to_email"`
	Amount    decimal.Decimal `json:"amount"`
	Direction string          `json:"direction"` // "credit" or "debit"
	CreatedAt time.Time       `json:"created_at"`
}

// GetTransactions returns all payments where the email is sender or receiver.
func (db *DB) GetTransactions(ctx context.Context, email string) ([]Transaction, error) {
	query := `
        SELECT p.id,
               fa.email AS from_email,
               ta.email AS to_email,
               p.amount,
               p.created_at
        FROM payments p
        JOIN accounts fa ON fa.id = p.from_account
        JOIN accounts ta ON ta.id = p.to_account
        WHERE fa.email = $1 OR ta.email = $1
        ORDER BY p.created_at DESC
    `

	rows, err := db.Pool.Query(ctx, query, email)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()

	transactions := make([]Transaction, 0)
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.FromEmail, &t.ToEmail, &t.Amount, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}
		if t.FromEmail == email {
			t.Direction = "debit"
		} else {
			t.Direction = "credit"
		}
		transactions = append(transactions, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return transactions, nil
}
