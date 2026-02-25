package models

import "time"

type DeviceMetric struct {
	ID          int       `json:"id" db:"id"`
	DeviceID    string    `json:"device_id" db:"device_id"`
	Voltage     float64   `json:"voltage" db:"voltage"`
	Current     float64   `json:"current" db:"current"`
	Temperature float64   `json:"temperature" db:"temperature"`
	Status      string    `json:"status" db:"status"`
	Timestamp   time.Time `json:"timestamp" db:"timestamp"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type CreateDeviceMetricRequest struct {
	Voltage     float64 `json:"voltage" binding:"required,min=100,max=240"`
	Current     float64 `json:"current" binding:"required,min=0,max=100"`
	Temperature float64 `json:"temperature" binding:"required,min=0,max=100"`
	Status      string  `json:"status" binding:"required,oneof=normal warning error"`
	Timestamp   string  `json:"timestamp,omitempty"`
}

type DeviceListResponse struct {
	DeviceID     string    `json:"device_id"`
	LastUpdated  time.Time `json:"last_updated"`
	LatestStatus string    `json:"latest_status"`
}

type SetRedisValueRequest struct {
	Value string `json:"value" binding:"required"`
	TTL   int    `json:"ttl,omitempty"`
}
