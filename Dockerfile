FROM golang:1.19-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o /gpu-temp-alert ./cmd/consumer

# Create a minimal image
FROM alpine:3.18

# Add CA certificates for secure connections to Slack
RUN apk add --no-cache ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /gpu-temp-alert .

# Run the binary
ENTRYPOINT ["/app/gpu-temp-alert"]