package redis

import (
	"context"
	"fmt"
	"time"

	"iot-data-collection/app/internal/config"

	"github.com/redis/go-redis/v9"
)


func NewRedisConnection(cfg *config.Config) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: "", 
		DB:       0, 
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("無法連線到 Redis: %w", err)
	}

	return rdb, nil
}
