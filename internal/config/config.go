package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds the application configuration
type Config struct {
	// Redpanda configuration
	RedpandaBrokers string
	SourceTopic     string
	ConsumerGroup   string

	// Alert configuration
	TemperatureThreshold float64

	// Slack configuration
	SlackWebhookURL string
	SlackChannel    string
}

// Load loads configuration from environment variables with fallbacks
func Load() (*Config, error) {
	config := &Config{
		RedpandaBrokers:      getEnv("REDPANDA_BROKERS", "redpanda-1:29092,redpanda-2:29093"),
		SourceTopic:          getEnv("SOURCE_TOPIC", "gpu-temperature"),
		ConsumerGroup:        getEnv("CONSUMER_GROUP", "gpu-temp-alert-group"),
		TemperatureThreshold: getEnvAsFloat("TEMPERATURE_THRESHOLD", 190.0),
		SlackWebhookURL:      getEnv("SLACK_WEBHOOK_URL", ""),
		SlackChannel:         getEnv("SLACK_CHANNEL", "#alerts"),
	}

	fmt.Printf("Configuration loaded. Redpanda brokers: %s\n", config.RedpandaBrokers)

	// Validate configuration
	if config.SlackWebhookURL == "" {
		return nil, fmt.Errorf("SLACK_WEBHOOK_URL must be provided")
	}

	if config.TemperatureThreshold <= 0 {
		return nil, fmt.Errorf("TEMPERATURE_THRESHOLD must be greater than 0")
	}

	return config, nil
}

// getEnv gets an environment variable with a fallback value
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// getEnvAsInt gets an environment variable as an integer with a fallback value
func getEnvAsInt(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return fallback
}

// getEnvAsFloat gets an environment variable as a float with a fallback value
func getEnvAsFloat(key string, fallback float64) float64 {
	if value, exists := os.LookupEnv(key); exists {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return fallback
}
