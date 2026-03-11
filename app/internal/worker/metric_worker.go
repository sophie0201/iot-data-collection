package worker

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"iot-data-collection/app/internal/cache"
	"iot-data-collection/app/internal/interfaces"
	"iot-data-collection/app/internal/models"
	"iot-data-collection/app/internal/queue"

	redisdriver "github.com/redis/go-redis/v9"
)

func RunMetricWorker(ctx context.Context, rdb *redisdriver.Client, db interfaces.DBClient, redisClient interfaces.RedisClient) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Metric worker 收到停止訊號，結束")
			return
		default:
			result, err := rdb.BRPop(ctx, 5*time.Second, queue.MetricQueueKey).Result()
			if err != nil || len(result) < 2 {
				continue
			}

			payload := result[1]
			var task interfaces.MetricTask
			if err := json.Unmarshal([]byte(payload), &task); err != nil {
				log.Printf("metric worker: 解析任務失敗: %v", err)
				continue
			}

			timestamp, err := time.Parse(time.RFC3339, task.Timestamp)
			if err != nil {
				timestamp = time.Now()
			}

			query := `
				INSERT INTO device_metrics (device_id, voltage, current, temperature, status, timestamp)
				VALUES ($1, $2, $3, $4, $5, $6)
				RETURNING id, device_id, voltage, current, temperature, status, timestamp, created_at
			`
			var metric models.DeviceMetric
			err = db.QueryRow(query, task.DeviceID, task.Voltage, task.Current, task.Temperature, task.Status, timestamp).Scan(
				&metric.ID, &metric.DeviceID, &metric.Voltage, &metric.Current,
				&metric.Temperature, &metric.Status, &metric.Timestamp, &metric.CreatedAt)
			if err != nil {
				log.Printf("metric worker: 寫入 DB 失敗 device=%s: %v", task.DeviceID, err)
				continue
			}

			cacheKey := cache.LatestMetricKey(task.DeviceID)
			if jsonBytes, err := json.Marshal(metric); err == nil {
				if setErr := redisClient.Set(ctx, cacheKey, string(jsonBytes), cache.LatestMetricTTL); setErr != nil {
					log.Printf("metric worker: 更新 cache 失敗 device=%s: %v，改為 invalidate", task.DeviceID, setErr)
					redisClient.Del(ctx, cacheKey)
				}
			} else {
				redisClient.Del(ctx, cacheKey)
			}

			log.Printf("metric worker: 已寫入 device=%s", task.DeviceID)
		}
	}
}
