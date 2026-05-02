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
	logger := pkg.NewStructuredLogger("kafka-producer")

	logger.Info("Starting Payment Outbox Producer", zap.String("env", cfg.Server.Env))

	// Initialize database
	db, err := database.NewDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	db.Logger = logger
	defer db.Close()

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

	// Configuration
	pollInterval := 5 * time.Second
	batchSize := 100

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start outbox polling goroutine
	go func() {
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				logger.Info("Producer shutting down")
				return
			case <-ticker.C:
				// Fetch PENDING records from outbox
				records, err := db.GetPendingOutboxRecords(ctx, batchSize)
				if err != nil {
					logger.Error("failed to fetch pending records", zap.Error(err))
					continue
				}

				if len(records) == 0 {
					logger.Debug("No pending records")
					continue
				}

				logger.Info("Fetched pending records", zap.Int("count", len(records)))

				// Publish to Kafka
				for _, record := range records {
					// Parse payload
					var payload PaymentEventPayload
					if err := json.Unmarshal([]byte(record.Payload), &payload); err != nil {
						logger.Error("failed to unmarshal payload", zap.Error(err), zap.Int("outboxID", record.ID))
						continue
					}

					// Add outbox_id to payload
					payload.OutboxID = record.ID

					// Serialize updated payload
					payloadBytes, err := json.Marshal(payload)
					if err != nil {
						logger.Error("failed to marshal payload", zap.Error(err), zap.Int("outboxID", record.ID))
						continue
					}

					// Publish to Kafka
					kafkaRecord := &kgo.Record{
						Topic: cfg.Kafka.Producer.Topic,
						Key:   []byte(fmt.Sprintf("%d", record.TransactionID)),
						Value: payloadBytes,
					}

					err = kafkaClient.Produce(ctx, kafkaRecord, func(record *kgo.Record, err error) {
						if err != nil {
							logger.Error("failed to produce to Kafka", zap.Error(err))
						} else {
							logger.Info("Published to Kafka", zap.String("topic", record.Topic))
						}
					})

					if err != nil {
						logger.Error("failed to produce record", zap.Error(err), zap.Int("outboxID", record.ID))
						continue
					}

					// Mark as PUBLISHED
					if err := db.MarkOutboxPublished(ctx, record.ID); err != nil {
						logger.Error("failed to mark as published", zap.Error(err), zap.Int("outboxID", record.ID))
						continue
					}

					logger.Info("Marked outbox as published", zap.Int("outboxID", record.ID), zap.Int("txID", record.TransactionID))
				}
			}
		}
	}()

	logger.Info("Producer started", zap.Duration("pollInterval", pollInterval), zap.Int("batchSize", batchSize))

	// Wait for shutdown signal
	<-sigChan
	logger.Info("Shutdown signal received, gracefully stopping...")
	cancel()

	// Give goroutines time to finish
	time.Sleep(2 * time.Second)
	logger.Info("Producer stopped")
}
