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
