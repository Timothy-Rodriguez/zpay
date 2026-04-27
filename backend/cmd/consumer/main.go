package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
	"zpay/internal/database"
	"zpay/internal/pkg"
	"zpay/internal/service"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

type PaymentEventPayload struct {
	OutboxID      int    `json:"outboxId"`
	TransactionID int    `json:"transactionId"`
	FromAccount   int    `json:"fromAccount"`
	ToAccount     int    `json:"toAccount"`
	Amount        string `json:"amount"`
	FromEmail     string `json:"fromEmail"`
	ToEmail       string `json:"toEmail"`
	EventType     string `json:"eventType"`
	Timestamp     string `json:"timestamp"`
}

func main() {
	// Load config
	cfg, err := pkg.LoadConfig("./config")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize logger
	logger := pkg.NewStructuredLogger("kafka-consumer")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	// defer func() {
	// 	if err := logger.Sync(); err != nil {
	// 		logger.Error("Error syncing logger", zap.Error(err))
	// 	}
	// }()

	logger.Info("Starting Payment Consumer", zap.String("env", cfg.Server.Env))

	// Initialize database
	db, err := database.NewDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	db.Logger = logger
	defer db.Close()

	// Initialize email client
	emailClient, err := service.NewEmailClient(&cfg.SMTP, logger)
	if err != nil {
		log.Fatal("Failed to initialize email client", zap.Error(err))
	}
	defer emailClient.Close()

	logger.Info("Email client initialized", zap.Bool("enabled", cfg.SMTP.Enabled))

	// Initialize Kafka client
	kafkaClient, err := pkg.NewKafkaClient(
		"PLAINTEXT", "", "", "", "", "", "",
		cfg.Kafka.Brokers,
		cfg.Kafka.Consumer.GroupID,
		cfg.Kafka.Producer.Topic,
		cfg.Kafka.Producer.Topic+"-dlq",
		cfg.Kafka.Consumer.Topic,
	)
	if err != nil {
		log.Fatal("Failed to initialize Kafka client", zap.Error(err))
	}
	defer kafkaClient.Close()

	logger.Info("Kafka client initialized", zap.Strings("brokers", cfg.Kafka.Brokers))

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start consuming
	err = kafkaClient.Consume(ctx, []string{cfg.Kafka.Consumer.Topic}, func(record *kgo.Record) {
		// Parse payload
		var payload PaymentEventPayload
		if err := json.Unmarshal(record.Value, &payload); err != nil {
			logger.Error("failed to unmarshal payload", zap.Error(err))
			return
		}

		logger.Info("Received message from Kafka",
			zap.Int("txID", payload.TransactionID),
			zap.Int("outboxID", payload.OutboxID),
			zap.String("from", payload.FromEmail))

		// Create audit record with outbox_id
		audit := &database.PaymentAudit{
			TransactionID: payload.TransactionID,
			OutboxID:      payload.OutboxID, // ← Include this!
			FromEmail:     payload.FromEmail,
			ToEmail:       payload.ToEmail,
			Amount:        payload.Amount,
			Status:        "PENDING",
			EmailSent:     false,
		}

		auditID, err := db.CreateAuditRecord(ctx, audit)
		if err != nil {
			logger.Error("failed to create audit record", zap.Error(err), zap.Int("outboxID", payload.OutboxID))
			return
		}

		logger.Info("Created audit record", zap.Int("auditID", auditID))

		// Send email
		emailErr := emailClient.SendTransactionEmail(
			payload.FromEmail,
			payload.FromEmail,
			payload.ToEmail,
			fmt.Sprintf("%d", payload.TransactionID),
			payload.Amount,
		)

		if emailErr != nil {
			logger.Error("failed to send email", zap.Error(emailErr), zap.Int("auditID", auditID))
			if err := db.MarkAuditFailed(ctx, auditID, fmt.Sprintf("email error: %v", emailErr)); err != nil {
				logger.Error("failed to mark audit as failed", zap.Error(err))
			}
			return
		}

		logger.Info("Email sent successfully", zap.Int("auditID", auditID))

		// Mark audit as PROCESSED
		if err := db.MarkAuditProcessed(ctx, auditID); err != nil {
			logger.Error("failed to mark audit as processed", zap.Error(err), zap.Int("auditID", auditID))
			return
		}

		// Mark outbox as PROCESSED
		if err := db.MarkOutboxProcessed(ctx, payload.OutboxID); err != nil {
			logger.Error("failed to mark outbox as processed", zap.Error(err), zap.Int("outboxID", payload.OutboxID))
			return
		}

		logger.Info("Processed message successfully",
			zap.Int("auditID", auditID),
			zap.Int("txID", payload.TransactionID),
			zap.Int("outboxID", payload.OutboxID))

		// Commit offset for this record after successful processing
		if err := kafkaClient.MarkRecordsProcessed(ctx, []*kgo.Record{record}); err != nil {
			logger.Error("failed to commit offset", zap.Error(err))
		}
	})

	if err != nil {
		log.Fatal("failed to consume from Kafka", zap.Error(err))
	}

	logger.Info("Consumer started, listening to topic", zap.String("topic", cfg.Kafka.Consumer.Topic))

	// Wait for shutdown signal
	<-sigChan
	logger.Info("Shutdown signal received, gracefully stopping...")
	cancel()

	// Give goroutines time to finish
	time.Sleep(2 * time.Second)
	logger.Info("Consumer stopped")
}
