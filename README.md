# GPU Temperature Alert

A Go application that monitors GPU temperature readings from Redpanda and sends alerts to Slack when temperatures exceed a configured threshold.

## Features

- Consumes GPU temperature data from Redpanda
- Monitors for temperatures exceeding the specified threshold (default: 190°C)
- Sends detailed alerts to Slack with device information
- Implements alert cooldown to prevent alert storms
- Supports configuration via environment variables
- Docker and docker-compose support for easy deployment

## Prerequisites

- Go 1.19 or later
- Redpanda cluster
- Slack workspace with webhook URL
- Docker and Docker Compose (optional)

## Project Structure

```
gpu-temp-alert/
├── cmd/
│   └── consumer/
│       └── main.go              # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go            # Configuration handling
│   ├── consumer/
│   │   └── redpanda_consumer.go # Redpanda consumer implementation
│   ├── model/
│   │   └── temperature.go       # Data models
│   └── notifier/
│       └── slack_notifier.go    # Slack notification implementation
├── Dockerfile                   # Docker image definition
├── docker-compose.yml           # Docker Compose configuration
└── go.mod                       # Go module definition
```

## Getting Started

### Setting Up Slack Webhook

1. Go to your Slack workspace settings
2. Navigate to "Apps & Integrations" → "Incoming Webhooks"
3. Create a new webhook or use an existing one
4. Copy the webhook URL for configuration

### Environment Variables

The application can be configured using the following environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `REDPANDA_BROKERS` | Comma-separated list of Redpanda brokers | `localhost:9092` |
| `SOURCE_TOPIC` | Topic to consume temperature readings from | `gpu-temperature` |
| `CONSUMER_GROUP` | Consumer group ID | `gpu-temp-alert-group` |
| `TEMPERATURE_THRESHOLD` | Temperature threshold in Celsius | `190.0` |
| `SLACK_WEBHOOK_URL` | Slack webhook URL (required) | - |
| `SLACK_CHANNEL` | Slack channel for notifications | `#alerts` |

### Local Development

1. Clone the repository:
   ```bash
   git clone https://github.com/yourusername/gpu-temp-alert.git
   cd gpu-temp-alert
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Update the module name in go.mod to match your GitHub username or organization.

4. Run the application:
   ```bash
   SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR_WEBHOOK_URL \
   REDPANDA_BROKERS=localhost:29092,localhost:29093 \
   go run cmd/consumer/main.go
   ```

### Using Docker

Build and run using Docker:

```bash
# Build the image
docker build -t gpu-temp-alert .

# Run the container
docker run -e REDPANDA_BROKERS=host.docker.internal:29092 \
           -e SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR_WEBHOOK_URL \
           gpu-temp-alert
```

### Using Docker Compose

1. Edit `docker-compose.yml` to update your Slack webhook URL and any other settings.

2. Start the service:
   ```bash
   docker-compose up -d
   ```

3. View logs:
   ```bash
   docker-compose logs -f
   ```

## Integrating with Existing Redpanda Setup

If you're running the GPU Temperature Publisher and want to add this alert service:

1. Ensure both services can connect to the same Redpanda cluster.
2. Make sure they're using the same topic name (`gpu-temperature` by default).
3. If using Docker Compose, update the networking configuration to connect to your existing Redpanda network.

## Alert Format

Slack alerts include:
- Device ID
- Current temperature and threshold
- GPU model and host name
- Utilization and power consumption
- Timestamp of the high temperature event

## Development

### VS Code Configuration

The project includes VS Code launch and settings files for development:

1. Open the project in VS Code
2. Update the Slack webhook URL in `.vscode/launch.json`
3. Use the "Launch GPU Temperature Alert" debug configuration

## License

This project is licensed under the MIT License - see the LICENSE file for details.
