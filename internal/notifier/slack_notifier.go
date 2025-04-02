package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"gpu-temp-alert/internal/model"
)

// SlackNotifier handles sending notifications to Slack
type SlackNotifier struct {
	webhookURL string
	channel    string
	httpClient *http.Client
}

// SlackMessage represents a Slack message
type SlackMessage struct {
	Channel     string       `json:"channel,omitempty"`
	Text        string       `json:"text,omitempty"`
	Username    string       `json:"username,omitempty"`
	IconEmoji   string       `json:"icon_emoji,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// Attachment represents a Slack message attachment
type Attachment struct {
	Fallback   string  `json:"fallback,omitempty"`
	Color      string  `json:"color,omitempty"`
	Pretext    string  `json:"pretext,omitempty"`
	Title      string  `json:"title,omitempty"`
	TitleLink  string  `json:"title_link,omitempty"`
	Text       string  `json:"text,omitempty"`
	Fields     []Field `json:"fields,omitempty"`
	Footer     string  `json:"footer,omitempty"`
	FooterIcon string  `json:"footer_icon,omitempty"`
	Timestamp  int64   `json:"ts,omitempty"`
}

// Field represents a field in a Slack attachment
type Field struct {
	Title string `json:"title,omitempty"`
	Value string `json:"value,omitempty"`
	Short bool   `json:"short,omitempty"`
}

// NewSlackNotifier creates a new Slack notifier
func NewSlackNotifier(webhookURL, channel string) (*SlackNotifier, error) {
	if webhookURL == "" {
		return nil, fmt.Errorf("slack webhook URL cannot be empty")
	}

	return &SlackNotifier{
		webhookURL: webhookURL,
		channel:    channel,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

// SendNotification sends a simple text notification to Slack
func (s *SlackNotifier) SendNotification(ctx context.Context, message string) error {
	slackMsg := SlackMessage{
		Channel:   s.channel,
		Text:      message,
		Username:  "GPU Temperature Monitor",
		IconEmoji: ":thermometer:",
	}

	return s.sendMessage(ctx, slackMsg)
}

// SendTemperatureAlert sends a detailed temperature alert to Slack
func (s *SlackNotifier) SendTemperatureAlert(ctx context.Context, alert *model.AlertEvent) error {
	// Format timestamp
	timestamp := alert.Timestamp.Format(time.RFC1123)

	// Determine color based on how far above threshold
	// Red for very high, orange for moderately high
	color := "#FF0000" // Default to red
	if alert.Temperature < alert.TemperatureThreshold+20 {
		color = "#FFA500" // Orange for less severe
	}

	// Create attachment
	attachment := Attachment{
		Fallback:  fmt.Sprintf("GPU IS MELTING: %.2f°C on %s", alert.Temperature, alert.DeviceID),
		Color:     color,
		Title:     "🔥 GPU IS MELTING 🔥",
		Text:      fmt.Sprintf("Temperature of *%.2f°C* detected, exceeding threshold of *%.2f°C*", alert.Temperature, alert.TemperatureThreshold),
		Timestamp: alert.Timestamp.Unix(),
		Fields: []Field{
			{
				Title: "Device ID",
				Value: alert.DeviceID,
				Short: true,
			},
			{
				Title: "Temperature",
				Value: fmt.Sprintf("%.2f°C", alert.Temperature),
				Short: true,
			},
			{
				Title: "Threshold",
				Value: fmt.Sprintf("%.2f°C", alert.TemperatureThreshold),
				Short: true,
			},
			{
				Title: "GPU Model",
				Value: alert.GPUModel,
				Short: true,
			},
			{
				Title: "Host",
				Value: alert.HostName,
				Short: true,
			},
			{
				Title: "GPU Utilization",
				Value: fmt.Sprintf("%.2f%%", alert.GPUUtil),
				Short: true,
			},
			{
				Title: "Power Consumption",
				Value: fmt.Sprintf("%.2f W", alert.PowerConsume),
				Short: true,
			},
			{
				Title: "Time",
				Value: timestamp,
				Short: false,
			},
		},
		Footer:     "GPU Temperature Monitoring System",
		FooterIcon: "https://platform.slack-edge.com/img/default_application_icon.png",
	}

	slackMsg := SlackMessage{
		Channel:     s.channel,
		Username:    "GPU Temperature Monitor",
		IconEmoji:   ":fire:",
		Attachments: []Attachment{attachment},
	}

	return s.sendMessage(ctx, slackMsg)
}

// sendMessage sends a message to Slack
func (s *SlackNotifier) sendMessage(ctx context.Context, message SlackMessage) error {
	// Marshal message to JSON
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("error marshaling Slack message: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", s.webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request to Slack: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response status: %s", resp.Status)
	}

	return nil
}
