package router

import (
	"database/sql"

	"iot-data-collection/app/internal/handlers"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func SetupRouter(db *sql.DB, rdb *redis.Client) *gin.Engine {
	r := gin.Default()
	h := handlers.NewHandlers(db, rdb)

	r.GET("/health", h.HealthCheck)

	v1 := r.Group("/api/v1")
	{
		devices := v1.Group("/devices")
		{
			devices.GET("", h.GetDevices)                                    // GET /api/v1/devices - 列出所有設備
			devices.POST("/:deviceId/metrics", h.CreateDeviceMetric)          // POST /api/v1/devices/{deviceId}/metrics - 接收設備資料回報
			devices.GET("/:deviceId/metrics", h.GetDeviceMetrics)             // GET /api/v1/devices/{deviceId}/metrics - 查詢歷史資料
			devices.GET("/:deviceId/latest", h.GetDeviceLatest)               // GET /api/v1/devices/{deviceId}/latest - 取得最新一筆資料
		}
	}

	return r
}
