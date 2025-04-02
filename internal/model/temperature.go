package model

import "time"

// TemperatureReading represents a temperature reading from a GPU
type TemperatureReading struct {
	DeviceID     string    `json:"device_id"`
	Temperature  float64   `json:"temperature"`
	Timestamp    time.Time `json:"timestamp"`
	GPUModel     string    `json:"gpu_model"`
	HostName     string    `json:"host_name"`
	PowerConsume float64   `json:"power_consume"`
	GPUUtil      float64   `json:"gpu_util"`
}

// AlertEvent represents a high temperature alert event
type AlertEvent struct {
	DeviceID             string    `json:"device_id"`
	Temperature          float64   `json:"temperature"`
	Timestamp            time.Time `json:"timestamp"`
	TemperatureThreshold float64   `json:"temperature_threshold"`
	GPUModel             string    `json:"gpu_model"`
	HostName             string    `json:"host_name"`
	PowerConsume         float64   `json:"power_consume"`
	GPUUtil              float64   `json:"gpu_util"`
}
