package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"iot-data-collection/app/internal/interfaces"
	"iot-data-collection/app/internal/models"
	"iot-data-collection/app/internal/service"

	"github.com/gin-gonic/gin"
)

var (
	ErrDBUnhealthy    = errors.New("database connection failed")
	ErrRedisUnhealthy = errors.New("redis connection failed")
)

type healthHandler struct {
	db  interfaces.DBClient
	rdb interfaces.RedisClient
}

func NewHealthHandler(db interfaces.DBClient, rdb interfaces.RedisClient) interfaces.HealthHandler {
	return &healthHandler{db: db, rdb: rdb}
}

func (hh *healthHandler) Check(ctx context.Context) error {
	if err := hh.db.Ping(); err != nil {
		return fmt.Errorf("%w: %v", ErrDBUnhealthy, err)
	}
	if err := hh.rdb.Ping(ctx); err != nil {
		return fmt.Errorf("%w: %v", ErrRedisUnhealthy, err)
	}
	return nil
}

type Handlers struct {
	HealthHandler interfaces.HealthHandler
	MetricSvc     service.DeviceMetricService
}

func (h *Handlers) HealthCheck(c *gin.Context) {
	err := h.HealthHandler.Check(c.Request.Context())
	if err != nil {
		msg := "服務異常"
		if errors.Is(err, ErrDBUnhealthy) {
			msg = "資料庫連線失敗"
		} else if errors.Is(err, ErrRedisUnhealthy) {
			msg = "Redis 連線失敗"
		}
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "unhealthy",
			"message": msg,
			"error":   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"message":   "服務正常運作",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// CreateDeviceMetric 接收設備 metric，驗證後交由 Service 非同步處理
func (h *Handlers) CreateDeviceMetric(c *gin.Context) {
	deviceID := c.Param("deviceId")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "deviceId 參數不能為空"})
		return
	}

	var req models.CreateDeviceMetricRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "無效的請求資料",
			"details": err.Error(),
		})
		return
	}

	in := service.SubmitMetricInput{
		DeviceID:    deviceID,
		Voltage:     req.Voltage,
		Current:     req.Current,
		Temperature: req.Temperature,
		Status:      req.Status,
		Timestamp:   req.Timestamp,
	}
	err := h.MetricSvc.SubmitMetric(c.Request.Context(), in)
	if err != nil {
		if errors.Is(err, service.ErrInvalidTimestamp) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "無效的時間格式，請使用 RFC3339 格式（例如：2024-01-01T12:00:00Z）",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "無法加入處理佇列",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message":   "資料已接受，將於背景處理",
		"device_id": deviceID,
	})
}

// GetDeviceMetrics 查詢設備歷史 metrics
func (h *Handlers) GetDeviceMetrics(c *gin.Context) {
	deviceID := c.Param("deviceId")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "deviceId 參數不能為空"})
		return
	}

	startTime, endTime, err := parseTimeRange(c.Query("start_time"), c.Query("end_time"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無效的 start_time 或 end_time 格式，請使用 RFC3339 格式"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	in := service.GetMetricsInput{
		DeviceID:  deviceID,
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     limit,
		Offset:    offset,
	}
	list, err := h.MetricSvc.GetMetrics(c.Request.Context(), in)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "無法取得資料",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id": deviceID,
		"count":     len(list),
		"data":      list,
	})
}

// GetDeviceLatest 取得設備最新一筆 metric
func (h *Handlers) GetDeviceLatest(c *gin.Context) {
	deviceID := c.Param("deviceId")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "deviceId 參數不能為空"})
		return
	}

	result, err := h.MetricSvc.GetLatest(c.Request.Context(), deviceID)
	if err != nil {
		if errors.Is(err, service.ErrDeviceNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error":      "找不到該設備的資料",
				"device_id": deviceID,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "無法取得資料"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   result.Data,
		"source": result.Source,
	})
}

// GetDevices 列出所有設備
func (h *Handlers) GetDevices(c *gin.Context) {
	list, err := h.MetricSvc.ListDevices(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "無法取得設備清單",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":   len(list),
		"devices": list,
	})
}

// parseTimeRange 解析 start_time、end_time，無效則回傳 error
func parseTimeRange(startStr, endStr string) (*time.Time, *time.Time, error) {
	var start, end *time.Time
	if startStr != "" {
		t, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			return nil, nil, err
		}
		start = &t
	}
	if endStr != "" {
		t, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			return nil, nil, err
		}
		end = &t
	}
	return start, end, nil
}
