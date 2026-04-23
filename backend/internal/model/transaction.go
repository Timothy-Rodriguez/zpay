package model

import "github.com/shopspring/decimal"

type TranactionRequest struct {
	FromEmail string          `json:"from_email" binding:"required,email"`
	ToEmail   string          `json:"to_email" binding:"required,email"`
	Amount    decimal.Decimal `json:"amount" binding:"required,email"`
}

type TransactionResponse struct {
	Message        string `json:"message"`
	From           string `json:"from"`
	To             string `json:"to"`
	Amount         string `json:"amount"`
	IdempotencyKey string `json:"idempotency_key"`
	Timestamp      string `json:"timestamp"`
}
