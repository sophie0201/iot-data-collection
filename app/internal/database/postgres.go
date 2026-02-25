package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"iot-data-collection/app/internal/config"

	_ "github.com/lib/pq" // PostgreSQL driver
)

func NewPostgresConnection(cfg *config.Config) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.PostgresHost,
		cfg.PostgresPort,
		cfg.PostgresUser,
		cfg.PostgresPassword,
		cfg.PostgresDB,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("無法開啟資料庫連線: %w", err)
	}

	db.SetMaxOpenConns(25)                 
	db.SetMaxIdleConns(5)                  
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("無法連線到資料庫: %w", err)
	}

	if err := initTables(db); err != nil {
		log.Printf("Error: 初始化資料表時發生錯誤: %v", err)
	}

	return db, nil
}

// initTables 初始化資料表
func initTables(db *sql.DB) error {
	// 建立 IoT 設備資料表
	// 符合需求：device_id, timestamp, voltage, current, temperature, status
	query := `
	CREATE TABLE IF NOT EXISTS device_metrics (
		id SERIAL PRIMARY KEY,
		device_id VARCHAR(255) NOT NULL,
		voltage DECIMAL(5, 2) NOT NULL CHECK (voltage >= 100 AND voltage <= 240),  -- 電壓範圍：100-240V
		current DECIMAL(5, 2) NOT NULL CHECK (current >= 0 AND current <= 100),    -- 電流範圍：0-100A
		temperature DECIMAL(5, 2) NOT NULL CHECK (temperature >= 0 AND temperature <= 100), -- 溫度範圍：0-100°C
		status VARCHAR(20) NOT NULL CHECK (status IN ('normal', 'warning', 'error')), -- 狀態限制
		timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- 建立索引以提升查詢效能
	CREATE INDEX IF NOT EXISTS idx_device_id ON device_metrics(device_id);
	CREATE INDEX IF NOT EXISTS idx_timestamp ON device_metrics(timestamp);
	CREATE INDEX IF NOT EXISTS idx_device_timestamp ON device_metrics(device_id, timestamp DESC); -- 複合索引，用於查詢單一設備的歷史資料
	CREATE INDEX IF NOT EXISTS idx_status ON device_metrics(status);
	`

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("建立資料表失敗: %w", err)
	}

	log.Println("資料表初始化完成")
	return nil
}
