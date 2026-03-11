package interfaces

import (
	"context"
	"database/sql"
	"time"

)

type DBClient interface {
	Ping() error
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Exec(query string, args ...interface{}) (sql.Result, error)
}

type RedisClient interface {
	Ping(ctx context.Context) error
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) error
}

type MetricTask struct {
	DeviceID    string  `json:"device_id"`
	Voltage     float64 `json:"voltage"`
	Current     float64 `json:"current"`
	Temperature float64 `json:"temperature"`
	Status      string  `json:"status"`
	Timestamp   string  `json:"timestamp"`
}

type MetricQueue interface {
	Push(ctx context.Context, task *MetricTask) error
}

type HealthHandler interface {
	Check(ctx context.Context) error
}
