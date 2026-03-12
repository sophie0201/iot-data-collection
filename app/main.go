package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"iot-data-collection/app/internal/config"
	"iot-data-collection/app/internal/database"
	"iot-data-collection/app/internal/queue"
	"iot-data-collection/app/internal/redis"
	"iot-data-collection/app/internal/router"
	"iot-data-collection/app/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("設定檔驗證失敗: %v", err)
	}

	db, err := database.NewPostgresConnection(cfg)
	if err != nil {
		log.Fatalf("無法連線到資料庫: %v", err)
	}
	defer db.Close()
	log.Println("資料庫連線成功")

	rdb, err := redis.NewRedisConnection(cfg)
	if err != nil {
		log.Fatalf("無法連線到 Redis: %v", err)
	}
	defer rdb.Close()
	log.Println("Redis 連線成功")

	metricQueue := queue.NewRedisMetricQueue(rdb)

	// 啟動背景 Worker 消費佇列並寫入 DB
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go worker.RunMetricWorker(ctx, rdb, db, redis.NewRedisAdapter(rdb))

	r := router.SetupRouter(db, rdb, metricQueue)

	// 啟動伺服器
	port := cfg.AppPort
	if port == "" {
		port = "8080"
	}

	log.Printf("伺服器啟動在 Port: %s", port)

	// 優雅關閉：收到 SIGINT/SIGTERM 時停止 worker 與 server
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("收到關閉訊號，停止服務...")
		cancel()
	}()

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("伺服器啟動失敗 Port: %s, Error: %v", port, err)
	}
}
