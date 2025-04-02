package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gpu-temp-alert/internal/config"
	"gpu-temp-alert/internal/consumer"
	"gpu-temp-alert/internal/notifier"
)

func main() {
	// Set up logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting GPU Temperature Alert Service")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Print configuration
	log.Printf("Configuration: Redpanda Brokers: %s, Topic: %s, Threshold: %.2f°C",
		cfg.RedpandaBrokers, cfg.SourceTopic, cfg.TemperatureThreshold)

	// Create context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create Slack notifier
	slack, err := notifier.NewSlackNotifier(cfg.SlackWebhookURL, cfg.SlackChannel)
	if err != nil {
		log.Fatalf("Failed to create Slack notifier: %v", err)
	}

	// Send startup notification
	err = slack.SendNotification(ctx, "🚀 GPU Temperature Alert Service started. Monitoring temperatures...")
	if err != nil {
		log.Printf("Warning: Failed to send startup notification to Slack: %v", err)
	}

	// Create consumer with retry logic
	log.Printf("Connecting to Redpanda brokers at %s...", cfg.RedpandaBrokers)

	var cons *consumer.RedpandaConsumer
	var retryErr error
	maxRetries := 10

	for i := 0; i < maxRetries; i++ {
		cons, err = consumer.NewRedpandaConsumer(
			ctx,
			cfg.RedpandaBrokers,
			cfg.SourceTopic,
			cfg.ConsumerGroup,
			cfg.TemperatureThreshold,
			slack,
		)
		if err == nil {
			break
		}

		retryErr = err
		retryDelay := time.Duration(i+1) * 5 * time.Second
		log.Printf("Failed to create Redpanda consumer (attempt %d/%d): %v", i+1, maxRetries, err)
		log.Printf("Retrying in %v...", retryDelay)
		time.Sleep(retryDelay)
	}

	if cons == nil {
		log.Fatalf("Failed to create Redpanda consumer after %d attempts: %v", maxRetries, retryErr)
	}

	log.Printf("Successfully connected to Redpanda cluster at %s", cfg.RedpandaBrokers)
	defer cons.Close()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start consuming in a separate goroutine
	consumeErrors := make(chan error, 1)
	go func() {
		if err := cons.Start(ctx); err != nil {
			consumeErrors <- err
		}
	}()

	// Wait for termination signal or error
	select {
	case sig := <-sigChan:
		log.Printf("Received signal %s, shutting down gracefully", sig)
	case err := <-consumeErrors:
		log.Printf("Consumer error: %v, shutting down", err)
	}

	// Cancel context to initiate shutdown
	cancel()

	// Allow some time for clean shutdown
	shutdownTimeout := 5 * time.Second
	log.Printf("Allowing %v for clean shutdown...", shutdownTimeout)
	time.Sleep(shutdownTimeout)

	// Send shutdown notification
	err = slack.SendNotification(ctx, "🛑 GPU Temperature Alert Service shutting down")
	if err != nil {
		log.Printf("Warning: Failed to send shutdown notification to Slack: %v", err)
	}

	log.Println("Application shutdown complete")
}
