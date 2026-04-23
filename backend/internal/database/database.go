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

	// Transactions table functions
	UpdateBalace(email string, amount decimal.Decimal) error
	ProcessTransaction(ctx context.Context, fromEmail string, toEmail string, amount decimal.Decimal) error
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

	CREATE TABLE IF NOT EXISTS transactions (
		id SERIAL PRIMARY KEY,
		from_account INT REFERENCES accounts(id) ON DELETE CASCADE,
		to_account INT REFERENCES accounts(id) ON DELETE CASCADE,
		amount DECIMAL(10, 2) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`

	if _, err := db.Pool.Exec(ctx, createTablesSQL); err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	return nil
}

func (db *DB) Close() {
	db.Pool.Close()
}
