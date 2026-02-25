package main

import (
	"log"

	"iot-data-collection/app/internal/config"
	"iot-data-collection/app/internal/database"
	"iot-data-collection/app/internal/redis"
	"iot-data-collection/app/internal/router"
)

func main() {
	cfg := config.Load()

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

	r := router.SetupRouter(db, rdb)

	// 啟動伺服器
	port := cfg.AppPort
	if port == "" {
		port = "8080"
	}

	log.Printf("伺服器啟動在 Port: %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("伺服器啟動失敗 Port: %s, Error: %v", port, err)
	}
}
