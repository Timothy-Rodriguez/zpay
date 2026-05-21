package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"zpay/internal/constants"
	"zpay/internal/model"
	"zpay/internal/pkg"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

type TranactionHandler struct {
	App *model.App
}

func NewTranactionHandler(app *model.App) *TranactionHandler {
	return &TranactionHandler{
		App: app,
	}
}

func (t *TranactionHandler) UpdateBalace(c *gin.Context) {
	ctx, span := t.App.Tracer.Start(c.Request.Context(), "update.balance")
	defer span.End()

	email := strings.TrimSpace(c.Query("email"))
	if email == "" {
		span.RecordError(fmt.Errorf("empty email address"))
		span.SetStatus(codes.Error, "email is required")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "email is required",
		})
		return
	}

	updatedBalance, err := decimal.NewFromString(c.Query("balance"))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid balance")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid balance",
		})
		return
	}

	if err := t.App.DB.UpdateBalace(ctx, email, updatedBalance); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to update balance")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to update balance",
		})
		return
	}

	span.SetStatus(codes.Ok, "balance updated")
	c.JSON(http.StatusOK, gin.H{
		"data": "balance updated",
	})
}

func (t *TranactionHandler) ProcessTransaction(c *gin.Context) {
	ctx, span := t.App.Tracer.Start(c.Request.Context(), "payment.process")
	defer span.End()

	userEmail, _ := c.Get(constants.ClaimsEmail)

	idempotencyKey := c.GetHeader("X-IDEMPOTENCY-KEY")
	span.SetAttributes(
		attribute.String("payment.idempotency_key", idempotencyKey),
	)
	if idempotencyKey == "" {
		span.SetStatus(codes.Error, "missing idempotency key")
		t.App.Logger.Warn("payment_missing_idempotency_key", "email", userEmail)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "missing idempotency key",
		})
		return
	}

	// Check if idempotency key exists in Redis
	redisKey := fmt.Sprintf("idempotency:%s", idempotencyKey)
	var transactionStatus model.TransactionStatus

	result, err := t.App.Redis.Get(ctx, redisKey).Result()
	if err == nil {
		span.SetStatus(codes.Ok, "idempotency hit")
		span.End()

		// Key exists, return stored transaction
		if err := json.Unmarshal([]byte(result), &transactionStatus); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "cached transaction unmarshal failed")
			t.App.Logger.Error("payment_cached_transaction_unmarshal_failed",
				"idempotency_key", idempotencyKey,
				"error", err.Error(),
			)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to deserialize cached transaction",
			})
			return
		}

		t.App.Logger.Info("payment_idempotency_hit",
			"idempotency_key", idempotencyKey,
			"from", transactionStatus.From,
			"to", transactionStatus.To,
		)
		c.JSON(http.StatusOK, transactionStatus)
		return
	}

	var transactionReq model.TranactionRequest
	if err := c.ShouldBindJSON(&transactionReq); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid request body")
		t.App.Logger.Warn("payment_invalid_body",
			"idempotency_key", idempotencyKey,
			"error", err.Error(),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "bad request body",
		})
		return
	}

	// Validate emails
	fromEmail, exists := c.Get(constants.ClaimsEmail)
	if !exists {
		span.SetStatus(codes.Error, "missing claims")
		t.App.Logger.Error("payment_missing_claims",
			"idempotency_key", idempotencyKey,
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "missing claims, please raise issue with this response",
		})
		return
	}

	transactionReq.ToEmail = strings.TrimSpace(transactionReq.ToEmail)
	span.SetAttributes(
		attribute.String("payment.from", fromEmail.(string)),
		attribute.String("payment.to", transactionReq.ToEmail),
		attribute.String("payment.amount", transactionReq.Amount.String()),
	)

	if fromEmail.(string) == transactionReq.ToEmail {
		span.RecordError(err)
		span.SetStatus(codes.Error, "same sender and receiver")
		t.App.Logger.Error("payment_logic_error",
			"idempotency_key", idempotencyKey,
			"from", fromEmail.(string),
			"to", transactionReq.ToEmail,
			"amount", transactionReq.Amount.String(),
			"error", fmt.Errorf("sender and receiver can't be same"),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "sender and receiver can't be same",
		})
		return
	}

	if transactionReq.Amount.LessThanOrEqual(decimal.Zero) {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid amount")
		t.App.Logger.Error("payment_logic_error",
			"idempotency_key", idempotencyKey,
			"from", fromEmail.(string),
			"to", transactionReq.ToEmail,
			"amount", transactionReq.Amount.String(),
			"error", fmt.Errorf("invalid amount"),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "amount must be greater than zero",
		})
		return
	}

	// Process transaction with DB transaction
	if err := t.App.DB.ProcessTransaction(
		ctx,
		fromEmail.(string),
		transactionReq.ToEmail,
		transactionReq.Amount,
	); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "db transaction failed")
		span.End()

		span.RecordError(err)
		span.SetStatus(codes.Error, "db transaction failed")
		t.App.Logger.Error("payment_db_failed",
			"idempotency_key", idempotencyKey,
			"from", fromEmail.(string),
			"to", transactionReq.ToEmail,
			"amount", transactionReq.Amount.String(),
			"error", err.Error(),
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	transactionStatus = model.TransactionStatus{
		Message:        "transaction completed successfully",
		From:           fromEmail.(string),
		To:             transactionReq.ToEmail,
		Amount:         transactionReq.Amount.String(),
		IdempotencyKey: idempotencyKey,
		Timestamp:      time.Now().String(),
	}
	var transactionStatusBytes []byte
	if transactionStatusBytes, err = json.Marshal(transactionStatus); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "marshal transaction status failed")
		t.App.Logger.Error("payment_status_marshal_failed",
			"idempotency_key", idempotencyKey,
			"error", err.Error(),
		)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Store idempotency key in Redis with 24-hour TTL
	err = t.App.Redis.Set(
		ctx,
		redisKey,
		string(transactionStatusBytes),
		24*time.Hour,
	).Err()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "redis set failed")

		t.App.Logger.Error("payment_idempotency_store_failed",
			"idempotency_key", idempotencyKey,
			"error", err.Error(),
		)
		t.App.Logger.Error(fmt.Sprintf("failed to store idempotency key in redis: %v", err))
	}
	span.SetStatus(codes.Ok, "redis set success")

	t.App.Logger.Info("payment_success",
		"idempotency_key", idempotencyKey,
		"from", fromEmail.(string),
		"to", transactionReq.ToEmail,
		"amount", transactionReq.Amount.String(),
	)
	c.JSON(http.StatusOK, transactionStatus)
}

func (t *TranactionHandler) GetBalance(c *gin.Context) {
	emailVal, exists := c.Get(constants.ClaimsEmail)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing claims"})
		return
	}
	email, ok := emailVal.(string)
	if !ok || email == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid claims"})
		return
	}

	balance, err := t.App.DB.GetBalance(c.Request.Context(), email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"email":   email,
		"balance": balance.String(),
	})
}

func (t *TranactionHandler) GetTransactions(c *gin.Context) {
	emailVal, exists := c.Get(constants.ClaimsEmail)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing claims"})
		return
	}
	email, ok := emailVal.(string)
	if !ok || email == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid claims"})
		return
	}

	transactions, err := t.App.DB.GetTransactions(c.Request.Context(), email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"email":        email,
		"count":        len(transactions),
		"transactions": transactions,
	})
}

func (t *TranactionHandler) GetDashboard(c *gin.Context) {
	start := time.Now()
	ctx, span := t.App.Tracer.Start(c.Request.Context(), "dashboard.get")
	defer span.End()

	emailVal, exists := c.Get(constants.ClaimsEmail)
	if !exists {
		span.SetStatus(codes.Error, "missing claims")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing claims"})
		return
	}
	email, ok := emailVal.(string)
	if !ok || email == "" {
		span.SetStatus(codes.Error, "invalid claims")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid claims"})
		return
	}
	span.SetAttributes(attribute.String("user.email", email))

	balance, err := t.App.DB.GetBalance(ctx, email)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get_balance failed")
		t.App.Logger.Error("dashboard_balance_failed", "email", email, "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch balance"})
		return
	}

	txns, err := t.App.DB.GetTransactions(ctx, email)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "get_transactions failed")
		t.App.Logger.Error("dashboard_transactions_failed", "email", email, "error", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch transactions"})
		return
	}

	// Aggregate totals
	var totalCredit, totalDebit decimal.Decimal
	for _, tx := range txns {
		if tx.Direction == "credit" {
			totalCredit = totalCredit.Add(tx.Amount)
		} else {
			totalDebit = totalDebit.Add(tx.Amount)
		}
	}

	// Build 7-day activity window (today and previous 6 days)
	now := time.Now()
	dayNames := [7]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	activity := make([]model.DailyActivity, 7)
	for i := range activity {
		d := now.AddDate(0, 0, -(6 - i))
		activity[i] = model.DailyActivity{
			Day:  dayNames[d.Weekday()],
			Date: d.Format("2006-01-02"),
		}
	}
	for _, tx := range txns {
		txDate := tx.CreatedAt.Format("2006-01-02")
		for j := range activity {
			if activity[j].Date == txDate {
				if tx.Direction == "credit" {
					activity[j].Credit = activity[j].Credit.Add(tx.Amount)
				} else {
					activity[j].Debit = activity[j].Debit.Add(tx.Amount)
				}
				break
			}
		}
	}

	// Last 5 transactions (already DESC by created_at from DB)
	recentCount := min(5, len(txns))
	recent := make([]model.Transaction, recentCount)
	for i := range recent {
		tx := txns[i]
		recent[i] = model.Transaction{
			ID:        tx.ID,
			FromEmail: tx.FromEmail,
			ToEmail:   tx.ToEmail,
			Amount:    tx.Amount,
			Direction: tx.Direction,
			CreatedAt: tx.CreatedAt,
		}
	}

	pkg.HTTPRequestDurationSeconds.WithLabelValues("GET", "/dashboard").Observe(time.Since(start).Seconds())
	pkg.HTTPRequestsTotal.WithLabelValues("GET", "/dashboard", "200").Inc()

	t.App.Logger.Info("dashboard_fetched",
		"email", email,
		"transaction_count", len(txns),
		"balance", balance.String(),
	)
	span.SetAttributes(attribute.Int("dashboard.transaction_count", len(txns)))
	span.SetStatus(codes.Ok, "dashboard fetched")

	c.JSON(http.StatusOK, model.DashboardResponse{
		Balance:          balance.String(),
		TotalCredited:    totalCredit.String(),
		TotalDebited:     totalDebit.String(),
		NetFlow:          totalCredit.Sub(totalDebit).String(),
		TransactionCount: len(txns),
		Activity:         activity,
		Recent:           recent,
	})
}
