package mocks

import (
	"context"
	"database/sql"
	"time"

	"iot-data-collection/app/internal/interfaces"
)

// MockDB 模擬資料庫操作
type MockDB struct {
	PingErr    error
	QueryRows  *sql.Rows
	QueryErr   error
	QueryRowResult   *sql.Row
	ExecResult sql.Result
	ExecErr    error
}

func (m *MockDB) Ping() error {
	return m.PingErr
}

func (m *MockDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return m.QueryRows, m.QueryErr
}

func (m *MockDB) QueryRow(query string, args ...interface{}) *sql.Row {
	return m.QueryRowResult
}

func (m *MockDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return m.ExecResult, m.ExecErr
}

// MockRedis 模擬 Redis 操作
type MockRedis struct {
	PingErr      error
	GetResult    string
	GetErr       error
	SetErr       error
	DelResult    int64
	DelErr       error
}

func (m *MockRedis) Ping(ctx context.Context) error {
    return m.PingErr
}

func (m *MockRedis) Get(ctx context.Context, key string) (string, error) {
	return m.GetResult, m.GetErr
}

func (m *MockRedis) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return m.SetErr
}

func (m *MockRedis) Del(ctx context.Context, keys ...string) error {
	return m.DelErr
}

// MockMetricQueue 模擬 MetricQueue，用於測試
type MockMetricQueue struct {
	PushErr error
}

func (m *MockMetricQueue) Push(ctx context.Context, task *interfaces.MetricTask) error {
	return m.PushErr
}

