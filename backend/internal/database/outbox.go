package database

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

type PaymentOutboxEvent struct {
	ID            int        `db:"id"`
	TransactionID int        `db:"transaction_id"`
	FromAccount   int        `db:"from_account"`
	ToAccount     int        `db:"to_account"`
	Amount        string     `db:"amount"`
	EventType     string     `db:"event_type"`
	Status        string     `db:"status"`
	Payload       string     `db:"payload"`
	CreatedAt     time.Time  `db:"created_at"`
	ProcessedAt   *time.Time `db:"processed_at"`
	RetryCount    int        `db:"retry_count"`
	LastError     *string    `db:"last_error"`
}

// GetPendingOutboxRecords fetches PENDING records from outbox
func (db *DB) GetPendingOutboxRecords(ctx context.Context, batchSize int) ([]*PaymentOutboxEvent, error) {
	query := `
        SELECT id, transaction_id, from_account, to_account, amount, event_type, status, payload, created_at, processed_at, retry_count, last_error
        FROM payment_outbox
        WHERE status = 'PENDING'
        ORDER BY created_at ASC
        LIMIT $1
        FOR UPDATE SKIP LOCKED
    `

	rows, err := db.Pool.Query(ctx, query, batchSize)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pending outbox records: %w", err)
	}
	defer rows.Close()

	var records []*PaymentOutboxEvent
	for rows.Next() {
		var rec PaymentOutboxEvent
		if err := rows.Scan(
			&rec.ID, &rec.TransactionID, &rec.FromAccount, &rec.ToAccount,
			&rec.Amount, &rec.EventType, &rec.Status, &rec.Payload,
			&rec.CreatedAt, &rec.ProcessedAt, &rec.RetryCount, &rec.LastError,
		); err != nil {
			db.Logger.Error("failed to scan outbox row", zap.String("err", err.Error()))
			continue
		}
		records = append(records, &rec)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating outbox records: %w", err)
	}

	return records, nil
}

// MarkOutboxPublished marks outbox record as PUBLISHED
func (db *DB) MarkOutboxPublished(ctx context.Context, outboxID int) error {
	query := `
        UPDATE payment_outbox
        SET status = 'PUBLISHED', processed_at = CURRENT_TIMESTAMP
        WHERE id = $1
    `
	_, err := db.Pool.Exec(ctx, query, outboxID)
	return err
}
