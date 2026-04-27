package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"zpay/internal/constants"
	"zpay/internal/model"

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
	email := strings.TrimSpace(c.Query("email"))
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "email is required",
		})
		return
	}

	updatedBalance, err := decimal.NewFromString(c.Query("balance"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid balance",
		})
		return
	}

	if err := t.App.DB.UpdateBalace(email, updatedBalance); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "failed to update balance",
		})
		return
	}

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

	redisGetCtx, redisGetSpan := t.App.Tracer.Start(ctx, "redis.get_idempotency")
	result, err := t.App.Redis.Get(context.Background(), redisKey).Result()
	if err == nil {
		redisGetSpan.SetStatus(codes.Ok, "idempotency hit")
		redisGetSpan.End()

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
	redisGetSpan.End()

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
	dbCtx, dbSpan := t.App.Tracer.Start(redisGetCtx, "db.process_transaction")
	if err := t.App.DB.ProcessTransaction(
		context.Background(),
		fromEmail.(string),
		transactionReq.ToEmail,
		transactionReq.Amount,
	); err != nil {
		dbSpan.RecordError(err)
		dbSpan.SetStatus(codes.Error, "db transaction failed")
		dbSpan.End()

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
	dbSpan.End()

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
	_, redisSetSpan := t.App.Tracer.Start(dbCtx, "redis.set_idempotency")
	err = t.App.Redis.Set(
		ctx,
		redisKey,
		string(transactionStatusBytes),
		24*time.Hour,
	).Err()
	if err != nil {
		redisSetSpan.RecordError(err)
		redisSetSpan.SetStatus(codes.Error, "redis set failed")
		redisSetSpan.End()

		t.App.Logger.Error("payment_idempotency_store_failed",
			"idempotency_key", idempotencyKey,
			"error", err.Error(),
		)
		t.App.Logger.Error(fmt.Sprintf("failed to store idempotency key in redis: %v", err))
	}
	redisSetSpan.SetStatus(codes.Ok, "redis set success")
	redisSetSpan.End()

	t.App.Logger.Info("payment_success",
		"idempotency_key", idempotencyKey,
		"from", fromEmail.(string),
		"to", transactionReq.ToEmail,
		"amount", transactionReq.Amount.String(),
	)
	c.JSON(http.StatusOK, transactionStatus)
}
