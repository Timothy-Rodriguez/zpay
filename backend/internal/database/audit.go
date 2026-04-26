package database

import (
	"context"
)

type PaymentAudit struct {
	ID            int     `db:"id"`
	TransactionID int     `db:"transaction_id"`
	OutboxID      int     `db:"outbox_id"`
	FromEmail     string  `db:"from_email"`
	ToEmail       string  `db:"to_email"`
	Amount        string  `db:"amount"`
	Status        string  `db:"status"`
	EmailSent     bool    `db:"email_sent"`
	ErrorMessage  *string `db:"error_message"`
}

// CreateAuditRecord creates an audit entry for payment processing
func (db *DB) CreateAuditRecord(ctx context.Context, audit *PaymentAudit) (int, error) {
	query := `
        INSERT INTO payment_audit (transaction_id, outbox_id, from_email, to_email, amount, status, email_sent, error_message)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        RETURNING id
    `
	var id int
	err := db.Pool.QueryRow(
		ctx, query,
		audit.TransactionID, audit.OutboxID, audit.FromEmail, audit.ToEmail,
		audit.Amount, audit.Status, audit.EmailSent, audit.ErrorMessage,
	).Scan(&id)
	return id, err
}

// MarkAuditProcessed updates audit record as PROCESSED
func (db *DB) MarkAuditProcessed(ctx context.Context, auditID int) error {
	query := `
        UPDATE payment_audit
        SET status = 'PROCESSED', processed_at = CURRENT_TIMESTAMP
        WHERE id = $1
    `
	_, err := db.Pool.Exec(ctx, query, auditID)
	return err
}

// MarkAuditFailed updates audit record as FAILED
func (db *DB) MarkAuditFailed(ctx context.Context, auditID int, errorMsg string) error {
	query := `
        UPDATE payment_audit
        SET status = 'FAILED', error_message = $1, processed_at = CURRENT_TIMESTAMP
        WHERE id = $2
    `
	_, err := db.Pool.Exec(ctx, query, errorMsg, auditID)
	return err
}

// MarkOutboxProcessed marks outbox as PROCESSED by consumer
func (db *DB) MarkOutboxProcessed(ctx context.Context, outboxID int) error {
	query := `
        UPDATE payment_outbox
        SET status = 'PROCESSED'
        WHERE id = $1
    `
	_, err := db.Pool.Exec(ctx, query, outboxID)
	return err
}

// MarkOutboxFailed marks outbox as FAILED
func (db *DB) MarkOutboxFailed(ctx context.Context, outboxID int, errorMsg string) error {
	query := `
        UPDATE payment_outbox
        SET status = 'FAILED', last_error = $1
        WHERE id = $1
    `
	_, err := db.Pool.Exec(ctx, query, errorMsg, outboxID)
	return err
}
