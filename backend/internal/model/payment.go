package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type TranactionRequest struct {
	ToEmail string          `json:"to_email" binding:"required,email"`
	Amount  decimal.Decimal `json:"amount" binding:"required"`
}

type TransactionResponse struct {
	Message        string `json:"message"`
	From           string `json:"from"`
	To             string `json:"to"`
	Amount         string `json:"amount"`
	IdempotencyKey string `json:"idempotency_key"`
	Timestamp      string `json:"timestamp"`
}

type TransactionStatus struct {
	Message        string `json:"message"`
	From           string `json:"from"`
	To             string `json:"to"`
	Amount         string `json:"amount"`
	IdempotencyKey string `json:"key"`
	Timestamp      string `json:"timestamp"`
}

type Transaction struct {
	ID        int             `json:"id"`
	FromEmail string          `json:"from_email"`
	ToEmail   string          `json:"to_email"`
	Amount    decimal.Decimal `json:"amount"`
	Direction string          `json:"direction"` // "credit" or "debit"
	CreatedAt time.Time       `json:"created_at"`
}

// DailyActivity holds aggregated credit/debit amounts for a single calendar day.
type DailyActivity struct {
	Day    string          `json:"day"`    // e.g. "Mon"
	Date   string          `json:"date"`   // e.g. "2026-05-19"
	Credit decimal.Decimal `json:"credit"` // total credits on this day
	Debit  decimal.Decimal `json:"debit"`  // total debits on this day
}

// DashboardResponse is the response shape for GET /dashboard.
type DashboardResponse struct {
	Balance          string          `json:"balance"`
	TotalCredited    string          `json:"total_credited"`
	TotalDebited     string          `json:"total_debited"`
	NetFlow          string          `json:"net_flow"`
	TransactionCount int             `json:"transaction_count"`
	Activity         []DailyActivity `json:"activity"` // last 7 calendar days
	Recent           []Transaction   `json:"recent"`   // last 5 transactions
}
