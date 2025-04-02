package consumer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"gpu-temp-alert/internal/model"
	"gpu-temp-alert/internal/notifier"

	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
)

// RedpandaConsumer consumes temperature readings from Redpanda
type RedpandaConsumer struct {
	client               *kgo.Client
	topic                string
	temperatureThreshold float64
	notifier             *notifier.SlackNotifier
	seenAlerts           map[string]time.Time // Map to track alerts to prevent duplicates
	seenAlertsMutex      sync.Mutex           // Mutex to protect the map
	alertCooldown        time.Duration        // Cooldown period for repeated alerts
}

// NewRedpandaConsumer creates a new Redpanda consumer
func NewRedpandaConsumer(
	ctx context.Context,
	brokers string,
	topic string,
	consumerGroup string,
	temperatureThreshold float64,
	notifier *notifier.SlackNotifier,
) (*RedpandaConsumer, error) {
	fmt.Printf("Attempting to connect to Redpanda brokers: %s\n", brokers)

	// Create Redpanda client options
	opts := []kgo.Opt{
		kgo.SeedBrokers(strings.Split(brokers, ",")...),
		kgo.ConsumerGroup(consumerGroup),
		kgo.ConsumeTopics(topic),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtEnd()), // Start from the newest messages
		kgo.DisableAutoCommit(),                         // We will manually commit
		kgo.SessionTimeout(30 * time.Second),
		kgo.RetryTimeout(30 * time.Second),
		kgo.RetryBackoffFn(func(attempts int) time.Duration {
			return time.Duration(attempts) * time.Second
		}),
	}

	// Create client with retry
	var client *kgo.Client
	var err error

	// Try to connect with retries
	for attempts := 1; attempts <= 5; attempts++ {
		fmt.Printf("Connection attempt %d/5 to Redpanda brokers\n", attempts)

		client, err = kgo.NewClient(opts...)
		if err != nil {
			fmt.Printf("Failed to create Redpanda client on attempt %d: %v\n", attempts, err)
			time.Sleep(2 * time.Duration(attempts) * time.Second)
			continue
		}

		// Test connection
		if err := checkConnection(ctx, client); err != nil {
			fmt.Printf("Connection test failed on attempt %d: %v\n", attempts, err)
			client.Close()
			time.Sleep(2 * time.Duration(attempts) * time.Second)
			continue
		}

		fmt.Println("Successfully connected to Redpanda!")
		return &RedpandaConsumer{
			client:               client,
			topic:                topic,
			temperatureThreshold: temperatureThreshold,
			notifier:             notifier,
			seenAlerts:           make(map[string]time.Time),
			alertCooldown:        5 * time.Minute, // Don't alert on the same device more than once per 5 minutes
		}, nil
	}

	// If we get here, all attempts failed
	return nil, fmt.Errorf("failed to connect to Redpanda after multiple attempts: %w", err)
}

// checkConnection verifies the connection to Redpanda
func checkConnection(ctx context.Context, client *kgo.Client) error {
	// Attempt to list topics to check connection
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Request a list of topics to verify connection
	req := kmsg.MetadataRequest{
		Topics: []kmsg.MetadataRequestTopic{},
	}

	resp, err := client.Request(ctx, &req)
	if err != nil {
		return fmt.Errorf("failed to connect to Redpanda: %w", err)
	}

	metaResp := resp.(*kmsg.MetadataResponse)
	if len(metaResp.Brokers) == 0 {
		return fmt.Errorf("no brokers found in Redpanda cluster")
	}

	// Print broker information for debugging
	fmt.Println("Successfully connected to Redpanda brokers:")
	for i, broker := range metaResp.Brokers {
		fmt.Printf("  Broker #%d: ID=%d, Host=%s, Port=%d\n",
			i+1, broker.NodeID, broker.Host, broker.Port)
	}

	return nil
}

// Start begins consuming messages
func (c *RedpandaConsumer) Start(ctx context.Context) error {
	log.Printf("Starting to consume from topic %s", c.topic)
	log.Printf("Monitoring for temperatures above %.2f°C", c.temperatureThreshold)

	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, stopping consumer")
			return nil

		default:
			// Poll for messages
			fetches := c.client.PollFetches(ctx)
			if fetches.IsClientClosed() {
				return fmt.Errorf("client closed")
			}

			if errs := fetches.Errors(); len(errs) > 0 {
				// Log errors but continue processing
				for _, err := range errs {
					log.Printf("Fetch error: %v", err)
				}
			}

			// Process records
			var processed int
			fetches.EachRecord(func(record *kgo.Record) {
				processed++
				c.processRecord(ctx, record)
			})

			// Commit offsets if we processed any records
			if processed > 0 {
				if err := c.client.CommitUncommittedOffsets(ctx); err != nil {
					log.Printf("Error committing offsets: %v", err)
				}
			}

			// If no records received, sleep briefly to avoid tight loop
			if processed == 0 {
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}

// processRecord processes a single record from Redpanda
func (c *RedpandaConsumer) processRecord(ctx context.Context, record *kgo.Record) {
	// Parse the temperature reading
	var reading model.TemperatureReading
	if err := json.Unmarshal(record.Value, &reading); err != nil {
		log.Printf("Error parsing record: %v", err)
		return
	}

	// Check if temperature exceeds threshold
	if reading.Temperature > c.temperatureThreshold {
		// Check if we've already alerted for this device recently
		if c.shouldSendAlert(reading.DeviceID) {
			log.Printf("HIGH TEMPERATURE ALERT: Device %s - %.2f°C exceeds threshold of %.2f°C",
				reading.DeviceID, reading.Temperature, c.temperatureThreshold)

			// Create alert event
			alert := &model.AlertEvent{
				DeviceID:             reading.DeviceID,
				Temperature:          reading.Temperature,
				Timestamp:            reading.Timestamp,
				TemperatureThreshold: c.temperatureThreshold,
				GPUModel:             reading.GPUModel,
				HostName:             reading.HostName,
				PowerConsume:         reading.PowerConsume,
				GPUUtil:              reading.GPUUtil,
			}

			// Send notification to Slack
			if err := c.notifier.SendTemperatureAlert(ctx, alert); err != nil {
				log.Printf("Failed to send alert to Slack: %v", err)
			}
		}
	}
}

// shouldSendAlert checks if we should send an alert for a device
// based on cooldown period to prevent alert storms
func (c *RedpandaConsumer) shouldSendAlert(deviceID string) bool {
	c.seenAlertsMutex.Lock()
	defer c.seenAlertsMutex.Unlock()

	now := time.Now()
	if lastAlerted, exists := c.seenAlerts[deviceID]; exists {
		// If last alert was less than cooldown period ago, don't alert
		if now.Sub(lastAlerted) < c.alertCooldown {
			return false
		}
	}

	// Update last alerted time
	c.seenAlerts[deviceID] = now
	return true
}

// Close closes the Redpanda client
func (c *RedpandaConsumer) Close() {
	if c.client != nil {
		c.client.Close()
	}
}
