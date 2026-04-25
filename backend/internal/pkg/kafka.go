package pkg

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/twmb/franz-go/pkg/kerr"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
	"github.com/twmb/franz-go/pkg/sasl/plain"
)

// KafkaClient defines the interface for a Kafka client.
type KafkaClient interface {
	Produce(ctx context.Context, record *kgo.Record, callback func(*kgo.Record, error)) error
	Consume(ctx context.Context, topics []string, handler func(*kgo.Record)) error
	MarkRecordsProcessed(ctx context.Context, records []*kgo.Record) error
	Ping(ctx context.Context) error
	IsConnected() bool
	Close()
}

// kafkaClient is a struct that implements the KafkaClient interface.
type kafkaClient struct {
	client  *kgo.Client
	groupID string
	// logger  zerolog.Logger
}

// NewKafkaClient creates and initializes a new KafkaClient instance.
func NewKafkaClient(
	securityProtocol, caCert, cert, key, mechanism, username, password string,
	brokers []string,
	groupID string,
	producerTopic string,
	producerDLQTopic string,
	consumerTopic string,
	// logger zerolog.Logger,
) (*kafkaClient, error) {
	// Validate configuration
	if len(brokers) == 0 {
		return nil, fmt.Errorf("kafka brokers cannot be empty")
	}
	if groupID == "" {
		return nil, fmt.Errorf("kafka consumer group ID cannot be empty")
	}

	// Set up Kafka client options
	opts := []kgo.Opt{
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup(groupID),
		kgo.AutoCommitMarks(), // Commit offsets only after they are processed
	}

	// Configure SASL if mechanism is provided
	if mechanism == "PLAIN" {
		opts = append(opts, kgo.SASL(plain.Auth{
			User: username,
			Pass: password,
		}.AsMechanism()))
	}

	// Configure TLS if security protocol includes SSL
	if securityProtocol == "SSL" || securityProtocol == "SASL_SSL" {
		tlsConfig, err := newTLSConfig(caCert, cert, key)
		if err != nil {
			return nil, fmt.Errorf("failed to configure TLS: %w", err)
		}
		opts = append(opts, kgo.DialTLSConfig(tlsConfig))
	}

	// Create Kafka client
	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka client: %w", err)
	}

	// Verify connection with retries
	kc := &kafkaClient{
		client:  client,
		groupID: groupID,
		// logger:  logger,
	}
	if err := kc.connectWithRetry(context.Background()); err != nil {
		kc.Close()
		return nil, fmt.Errorf("failed to initialize Kafka client: %w", err)
	}

	// Create topics if they don't exist
	if err := kc.createTopics(context.Background(), []string{producerTopic, producerDLQTopic, consumerTopic}); err != nil {
		kc.Close()
		return nil, fmt.Errorf("failed to create topics: %w", err)
	}

	return kc, nil
}

// newTLSConfig creates a TLS configuration from certificate files
func newTLSConfig(caCert, cert, key string) (*tls.Config, error) {
	tlsConfig := &tls.Config{}
	if caCert != "" {
		caCertBytes, err := os.ReadFile(caCert)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA cert: %w", err)
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCertBytes) {
			return nil, fmt.Errorf("failed to parse CA cert")
		}
		tlsConfig.RootCAs = caCertPool
	}

	if cert != "" && key != "" {
		cert, err := tls.LoadX509KeyPair(cert, key)
		if err != nil {
			return nil, fmt.Errorf("failed to load client cert and key: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// connectWithRetry attempts to connect to the Kafka broker with exponential backoff.
func (c *kafkaClient) connectWithRetry(ctx context.Context) error {
	const retries = 5
	const initialDelay = 2 * time.Second

	var lastErr error
	var err error
	delay := initialDelay
	for i := 0; i < retries; i++ {
		if err = c.Ping(ctx); err == nil {
			return nil
		}
		lastErr = err
		// c.logger.Warn().
		// 	Err(err).
		// 	Int("attempt", i+1).
		// 	Msg("Failed to connect to Kafka, retrying")

		select {
		case <-ctx.Done():
			return fmt.Errorf("context canceled during connection retries: %w", ctx.Err())
		case <-time.After(delay):
			delay = time.Duration(float64(delay) * 1.5) // Exponential backoff
		}
	}

	return fmt.Errorf("failed to connect to Kafka broker after %d attempts: %w", retries, lastErr)
}

// createTopics creates the specified topics if they don't exist
func (c *kafkaClient) createTopics(ctx context.Context, topics []string) error {
	for _, topic := range topics {
		if topic == "" {
			// c.logger.Warn().Msg("Skipping empty topic name")
			continue
		}

		req := kmsg.NewCreateTopicsRequest()
		topicReq := kmsg.NewCreateTopicsRequestTopic()
		topicReq.Topic = topic
		topicReq.NumPartitions = 1     // Single partition for simplicity
		topicReq.ReplicationFactor = 1 // Single broker
		req.Topics = append(req.Topics, topicReq)

		resp, err := req.RequestWith(ctx, c.client)
		if err != nil {
			// c.logger.Error().
			// 	Err(err).
			// 	Str("topic", topic).
			// 	Msg("Failed to create topic")
			return fmt.Errorf("failed to create topic %s: %w", topic, err)
		}

		for _, tr := range resp.Topics {
			if tr.ErrorCode != 0 {
				err := kerr.ErrorForCode(tr.ErrorCode)
				if err == kerr.TopicAlreadyExists {
					// c.logger.Info().
					// 	Str("topic", tr.Topic).
					// 	Msg("Topic already exists, skipping creation")
					continue
				}
				// c.logger.Error().
				// 	Err(err).
				// 	Str("topic", tr.Topic).
				// 	Msg("Failed to create topic")
				return fmt.Errorf("failed to create topic %s: %w", tr.Topic, err)
			}
			// c.logger.Info().
			// 	Str("topic", tr.Topic).
			// 	Msg("Successfully created topic")
		}
	}
	return nil
}

// Produce sends a record to Kafka with an optional callback for completion.
func (c *kafkaClient) Produce(ctx context.Context, record *kgo.Record, callback func(*kgo.Record, error)) error {
	if record == nil {
		return fmt.Errorf("cannot produce: record is nil")
	}
	if record.Topic == "" {
		return fmt.Errorf("cannot produce: record topic is empty")
	}

	// Log at debug level to reduce overhead
	// c.logger.Debug().Msg("Producing record to Kafka")

	if c.client == nil {
		return fmt.Errorf("cannot produce: client is not initialized")
	}
	if !c.IsConnected() {
		return fmt.Errorf("cannot produce: client is not connected")
	}

	c.client.Produce(ctx, record, callback)
	return nil
}

// Consume subscribes to the specified topics and processes incoming records with the provided handler.
func (c *kafkaClient) Consume(ctx context.Context, topics []string, handler func(*kgo.Record)) error {
	if c.client == nil {
		return fmt.Errorf("cannot consume: client is not initialized")
	}
	if !c.IsConnected() {
		return fmt.Errorf("cannot consume: client is not connected")
	}
	if len(topics) == 0 {
		return fmt.Errorf("cannot consume: no topics provided")
	}
	if handler == nil {
		return fmt.Errorf("cannot consume: handler is nil")
	}

	// Assign topics to consume
	c.client.AddConsumeTopics(topics...)

	// Start polling for records
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				fetches := c.client.PollFetches(ctx)
				fetches.EachError(func(topic string, partition int32, err error) {
					// c.logger.Error().
					// 	Str("topic", topic).
					// 	Int32("partition", partition).
					// 	Err(err).
					// 	Msg("Error while polling Kafka")
				})
				fetches.EachRecord(handler)
			}
		}
	}()

	return nil
}

// MarkRecordsProcessed marks and commits the offsets for a batch of Kafka records.
func (c *kafkaClient) MarkRecordsProcessed(ctx context.Context, records []*kgo.Record) error {
	if c.client == nil {
		return fmt.Errorf("cannot mark offsets: client is not initialized")
	}
	if !c.IsConnected() {
		return fmt.Errorf("cannot mark offsets: client is not connected")
	}

	// Create a context with a timeout to prevent hangs
	commitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Mark all records as processed
	c.client.MarkCommitRecords(records...)

	// Commit the marked offsets
	if err := c.client.CommitMarkedOffsets(commitCtx); err != nil {
		// c.logger.Error().
		// 	Err(err).
		// 	Int("record_count", len(records)).
		// 	Msg("Failed to commit offsets")
		return fmt.Errorf("failed to commit offsets: %w", err)
	}

	// c.logger.Debug().
	// 	Int("record_count", len(records)).
	// 	Msg("Successfully committed offsets")
	return nil
}

// Ping checks the connection to the Kafka broker.
func (c *kafkaClient) Ping(ctx context.Context) error {
	if c.client == nil {
		return fmt.Errorf("cannot ping: client is not initialized")
	}

	return c.client.Ping(ctx)
}

// IsConnected checks if the client is currently connected to Kafka.
func (c *kafkaClient) IsConnected() bool {
	if c.client == nil {
		return false
	}
	return c.Ping(context.Background()) == nil
}

// Close gracefully closes the Kafka client.
func (c *kafkaClient) Close() {
	if c.client != nil {
		c.client.Close()
		c.client = nil
	}
}
