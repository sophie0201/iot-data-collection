package queue

import (
	"context"
	"encoding/json"

	"iot-data-collection/app/internal/interfaces"

	redisdriver "github.com/redis/go-redis/v9"
)

// MetricQueueKey Redis 中儲存 metric 任務的 List key（供 worker 消費用）
const MetricQueueKey = "iot:metric:tasks"

// RedisMetricQueue 使用 Redis List 實作的 MetricQueue
type RedisMetricQueue struct {
	client *redisdriver.Client
}

// NewRedisMetricQueue 建立 Redis 版的 MetricQueue
func NewRedisMetricQueue(client *redisdriver.Client) interfaces.MetricQueue {
	return &RedisMetricQueue{client: client}
}

// Push 將任務加入佇列（LPUSH，非阻塞）
func (q *RedisMetricQueue) Push(ctx context.Context, task *interfaces.MetricTask) error {
	data, err := json.Marshal(task)
	if err != nil {
		return err
	}
	return q.client.LPush(ctx, MetricQueueKey, data).Err()
}
