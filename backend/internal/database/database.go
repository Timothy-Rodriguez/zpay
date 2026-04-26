package database

import (
	"context"
	"fmt"
	"zpay/internal/pkg"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type DB struct {
	Pool   *pgxpool.Pool
	Logger *pkg.Logger
}

type DatabaseClient interface {
	InitializeTables(ctx context.Context) error
	Close()

	// Users table functions
	CreateUser(email, password string) error
	CheckLoginAndStoreRefreshToken(ctx context.Context, email string, password string, refreshToken string) (bool, error)

	// Payments table functions
	UpdateBalace(email string, amount decimal.Decimal) error
	ProcessTransaction(ctx context.Context, fromEmail string, toEmail string, amount decimal.Decimal) error

	// Outbox table functions
	GetPendingOutboxRecords(ctx context.Context, batchSize int) ([]*PaymentOutboxEvent, error)
	MarkOutboxPublished(ctx context.Context, outboxID int) error
}

func NewDB(cfg *pkg.DatabaseConfig) (*DB, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DBName,
		cfg.SSLMode,
	)

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse db config: %w", err)
	}

	config.MaxConns = int32(cfg.MaxConns)

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{Pool: pool}

	// Initialize tables
	if err := db.InitializeTables(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	return db, nil
}

func (db *DB) InitializeTables(ctx context.Context) error {
	createTablesSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		email VARCHAR(255) UNIQUE NOT NULL,
		password VARCHAR(255) NOT NULL,
		refresh_token VARCHAR(255),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS accounts (
		id SERIAL PRIMARY KEY,
		email VARCHAR(255) UNIQUE NOT NULL,
		balance DECIMAL(10, 2) DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS payments (
		id SERIAL PRIMARY KEY,
		from_account INT REFERENCES accounts(id) ON DELETE CASCADE,
		to_account INT REFERENCES accounts(id) ON DELETE CASCADE,
		amount DECIMAL(10, 2) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS payment_outbox (
        id SERIAL PRIMARY KEY,
        transaction_id INT NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
        from_account INT NOT NULL REFERENCES accounts(id),
        to_account INT NOT NULL REFERENCES accounts(id),
        amount DECIMAL(10, 2) NOT NULL,
        event_type VARCHAR(50) NOT NULL,
        status VARCHAR(50) DEFAULT 'PENDING',
        payload JSONB NOT NULL,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        processed_at TIMESTAMP,
        retry_count INT DEFAULT 0,
        last_error VARCHAR(500)
    );

	CREATE TABLE IF NOT EXISTS payment_audit (
        id SERIAL PRIMARY KEY,
        transaction_id INT NOT NULL,
        outbox_id INT NOT NULL REFERENCES payment_outbox(id),
        from_email VARCHAR(255) NOT NULL,
        to_email VARCHAR(255) NOT NULL,
        amount DECIMAL(10, 2) NOT NULL,
        status VARCHAR(50) NOT NULL,
        email_sent BOOLEAN DEFAULT FALSE,
        error_message VARCHAR(500),
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        processed_at TIMESTAMP
    );

	CREATE INDEX IF NOT EXISTS idx_payment_outbox_status ON payment_outbox(status);
    CREATE INDEX IF NOT EXISTS idx_payment_outbox_created_at ON payment_outbox(created_at);
    CREATE INDEX IF NOT EXISTS idx_payment_audit_transaction ON payment_audit(transaction_id);
    CREATE INDEX IF NOT EXISTS idx_payment_audit_status ON payment_audit(status);
	`

	if _, err := db.Pool.Exec(ctx, createTablesSQL); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	return nil
}

func (db *DB) Close() {
	db.Pool.Close()
}
