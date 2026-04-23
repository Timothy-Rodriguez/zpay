package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"zpay/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
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

	idempotencyKey := c.GetHeader("X-IDEMPOTENCY-KEY")
	if idempotencyKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "missing idempotency key",
		})
		return
	}

	// Check if idempotency key exists in Redis
	ctx := context.Background()
	redisKey := fmt.Sprintf("idempotency:%s", idempotencyKey)

	// Check if idempotency key exists in Redis
	result, err := t.App.Redis.Get(ctx, redisKey).Result()
	if err == nil {
		// Key exists, return stored transaction
		var txnResponse model.TranactionRequest
		if err := json.Unmarshal([]byte(result), &txnResponse); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to deserialize cached transaction",
			})
			return
		}
		c.JSON(http.StatusOK, txnResponse)
		return
	}

	var transactionReq model.TranactionRequest
	if err := c.ShouldBindJSON(&transactionReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "bad request body",
		})
		return
	}

	// Validate emails
	transactionReq.FromEmail = strings.TrimSpace(transactionReq.FromEmail)
	transactionReq.ToEmail = strings.TrimSpace(transactionReq.ToEmail)

	if transactionReq.FromEmail == transactionReq.ToEmail {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "from and to emails cannot be the same",
		})
		return
	}

	if transactionReq.Amount.LessThanOrEqual(decimal.Zero) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "amount must be greater than zero",
		})
		return
	}

	// Process transaction with DB transaction
	if err := t.App.DB.ProcessTransaction(
		context.Background(),
		transactionReq.FromEmail,
		transactionReq.ToEmail,
		transactionReq.Amount,
	); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Store idempotency key in Redis with 24-hour TTL
	err = t.App.Redis.Set(
		ctx,
		redisKey,
		"processed",
		24*time.Hour,
	).Err()
	if err != nil {
		t.App.Logger.Error(fmt.Sprintf("failed to store idempotency key in redis: %v", err))
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "transaction completed successfully",
		"from":            transactionReq.FromEmail,
		"to":              transactionReq.ToEmail,
		"amount":          transactionReq.Amount.String(),
		"idempotency_key": idempotencyKey,
	})
}
