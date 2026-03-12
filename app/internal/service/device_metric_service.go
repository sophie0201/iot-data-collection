package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"time"

	"iot-data-collection/app/internal/cache"
	"iot-data-collection/app/internal/interfaces"
	"iot-data-collection/app/internal/models"

	"golang.org/x/sync/singleflight"
)

// SubmitMetricInput 提交 metric 的輸入
type SubmitMetricInput struct {
	DeviceID    string
	Voltage     float64
	Current     float64
	Temperature float64
	Status      string
	Timestamp   string // 空字串表示使用當下時間
}

// GetMetricsInput 查詢歷史 metrics 的輸入
type GetMetricsInput struct {
	DeviceID  string
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
	Offset    int
}

// GetLatestResult 取得最新一筆的結果（含資料來源）
type GetLatestResult struct {
	Data   models.DeviceMetric
	Source string // "cache" 或 "database"
}

// DeviceMetricService 設備指標的業務邏輯介面
type DeviceMetricService interface {
	SubmitMetric(ctx context.Context, in SubmitMetricInput) error
	GetMetrics(ctx context.Context, in GetMetricsInput) ([]models.DeviceMetric, error)
	GetLatest(ctx context.Context, deviceID string) (*GetLatestResult, error)
	ListDevices(ctx context.Context) ([]models.DeviceListResponse, error)
}

// deviceMetricServiceImpl 實作
type deviceMetricServiceImpl struct {
	db          interfaces.DBClient
	rdb         interfaces.RedisClient
	metricQueue interfaces.MetricQueue
	sf          singleflight.Group
}

// NewDeviceMetricService 建立 DeviceMetricService
func NewDeviceMetricService(
	db interfaces.DBClient,
	rdb interfaces.RedisClient,
	metricQueue interfaces.MetricQueue,
) DeviceMetricService {
	return &deviceMetricServiceImpl{
		db:          db,
		rdb:         rdb,
		metricQueue: metricQueue,
	}
}

func (s *deviceMetricServiceImpl) SubmitMetric(ctx context.Context, in SubmitMetricInput) error {
	timestampStr := in.Timestamp
	if timestampStr != "" {
		if _, err := time.Parse(time.RFC3339, timestampStr); err != nil {
			return ErrInvalidTimestamp
		}
	} else {
		timestampStr = time.Now().Format(time.RFC3339)
	}

	task := &interfaces.MetricTask{
		DeviceID:    in.DeviceID,
		Voltage:     in.Voltage,
		Current:     in.Current,
		Temperature: in.Temperature,
		Status:      in.Status,
		Timestamp:   timestampStr,
	}
	if err := s.metricQueue.Push(ctx, task); err != nil {
		return err
	}
	return nil
}

func (s *deviceMetricServiceImpl) GetMetrics(ctx context.Context, in GetMetricsInput) ([]models.DeviceMetric, error) {
	limit := in.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	offset := in.Offset
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, device_id, voltage, current, temperature, status, timestamp, created_at 
		FROM device_metrics 
		WHERE device_id = $1
	`
	args := []interface{}{in.DeviceID}
	argIndex := 2

	if in.StartTime != nil {
		query += " AND timestamp >= $" + strconv.Itoa(argIndex)
		args = append(args, *in.StartTime)
		argIndex++
	}
	if in.EndTime != nil {
		query += " AND timestamp <= $" + strconv.Itoa(argIndex)
		args = append(args, *in.EndTime)
		argIndex++
	}

	query += " ORDER BY timestamp DESC LIMIT $" + strconv.Itoa(argIndex) + " OFFSET $" + strconv.Itoa(argIndex+1)
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.DeviceMetric
	for rows.Next() {
		var d models.DeviceMetric
		if err := rows.Scan(&d.ID, &d.DeviceID, &d.Voltage, &d.Current,
			&d.Temperature, &d.Status, &d.Timestamp, &d.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, nil
}

func (s *deviceMetricServiceImpl) GetLatest(ctx context.Context, deviceID string) (*GetLatestResult, error) {
	cacheKey := cache.LatestMetricKey(deviceID)

	cached, err := s.rdb.Get(ctx, cacheKey)
	if err == nil {
		var data models.DeviceMetric
		if jsonErr := json.Unmarshal([]byte(cached), &data); jsonErr == nil {
			return &GetLatestResult{Data: data, Source: "cache"}, nil
		}
	}

	sfKey := "GetLatest:" + deviceID
	v, err, _ := s.sf.Do(sfKey, func() (interface{}, error) {
		cached2, err2 := s.rdb.Get(ctx, cacheKey)
		if err2 == nil {
			var data models.DeviceMetric
			if jsonErr := json.Unmarshal([]byte(cached2), &data); jsonErr == nil {
				return &GetLatestResult{Data: data, Source: "cache"}, nil
			}
		}

		query := `
			SELECT id, device_id, voltage, current, temperature, status, timestamp, created_at 
			FROM device_metrics 
			WHERE device_id = $1
			ORDER BY timestamp DESC
			LIMIT 1
		`
		var data models.DeviceMetric
		err2 = s.db.QueryRow(query, deviceID).Scan(
			&data.ID, &data.DeviceID, &data.Voltage, &data.Current,
			&data.Temperature, &data.Status, &data.Timestamp, &data.CreatedAt)
		if err2 != nil {
			if err2 == sql.ErrNoRows {
				return nil, ErrDeviceNotFound
			}
			return nil, err2
		}

		if jsonBytes, jsonErr := json.Marshal(data); jsonErr == nil {
			s.rdb.Set(ctx, cacheKey, string(jsonBytes), cache.LatestMetricTTL)
		}

		return &GetLatestResult{Data: data, Source: "database"}, nil
	})

	if err != nil {
		return nil, err
	}
	return v.(*GetLatestResult), nil
}

func (s *deviceMetricServiceImpl) ListDevices(ctx context.Context) ([]models.DeviceListResponse, error) {
	query := `
		SELECT DISTINCT ON (device_id) 
			device_id, 
			timestamp as last_updated,
			status as latest_status
		FROM device_metrics
		ORDER BY device_id, timestamp DESC
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []models.DeviceListResponse
	for rows.Next() {
		var d models.DeviceListResponse
		if err := rows.Scan(&d.DeviceID, &d.LastUpdated, &d.LatestStatus); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	return list, nil
}
