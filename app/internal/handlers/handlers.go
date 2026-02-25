package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"iot-data-collection/app/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type Handlers struct {
	db  *sql.DB
	rdb *redis.Client
}

func NewHandlers(db *sql.DB, rdb *redis.Client) *Handlers {
	return &Handlers{
		db:  db,
		rdb: rdb,
	}
}

func (h *Handlers) HealthCheck(c *gin.Context) {
	if err := h.db.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "unhealthy",
			"message": "資料庫連線失敗",
			"error":   err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	if err := h.rdb.Ping(ctx).Err(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "unhealthy",
			"message": "Redis 連線失敗",
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

func (h *Handlers) CreateDeviceMetric(c *gin.Context) {
	deviceID := c.Param("deviceId")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "deviceId 參數不能為空",
		})
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

	var timestamp time.Time
	if req.Timestamp != "" {
		var err error
		timestamp, err = time.Parse(time.RFC3339, req.Timestamp)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "無效的時間格式，請使用 RFC3339 格式（例如：2024-01-01T12:00:00Z）",
			})
			return
		}
	} else {
		timestamp = time.Now()
	}

	query := `
		INSERT INTO device_metrics (device_id, voltage, current, temperature, status, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`

	var id int
	var createdAt time.Time
	err := h.db.QueryRow(query, deviceID, req.Voltage, req.Current, req.Temperature, req.Status, timestamp).
		Scan(&id, &createdAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "無法建立資料",
			"details": err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	cacheKey := "device_metric:" + deviceID + ":latest"
	h.rdb.Del(ctx, cacheKey)

	data := models.DeviceMetric{
		ID:          id,
		DeviceID:    deviceID,
		Voltage:     req.Voltage,
		Current:     req.Current,
		Temperature: req.Temperature,
		Status:      req.Status,
		Timestamp:   timestamp,
		CreatedAt:   createdAt,
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "資料建立成功",
		"data":    data,
	})
}

func (h *Handlers) GetDeviceMetrics(c *gin.Context) {
	deviceID := c.Param("deviceId")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "deviceId 參數不能為空",
		})
		return
	}

	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")
	limitStr := c.DefaultQuery("limit", "100")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 1000 {
		limit = 100 // 限制最多 1000 筆，預設 100 筆
	}
	offset, _ := strconv.Atoi(offsetStr)
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, device_id, voltage, current, temperature, status, timestamp, created_at 
		FROM device_metrics 
		WHERE device_id = $1
	`
	args := []interface{}{deviceID}
	argIndex := 2

	if startTimeStr != "" {
		startTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "無效的 start_time 格式，請使用 RFC3339 格式",
			})
			return
		}
		query += " AND timestamp >= $" + strconv.Itoa(argIndex)
		args = append(args, startTime)
		argIndex++
	}

	if endTimeStr != "" {
		endTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "無效的 end_time 格式，請使用 RFC3339 格式",
			})
			return
		}
		query += " AND timestamp <= $" + strconv.Itoa(argIndex)
		args = append(args, endTime)
		argIndex++
	}

	query += " ORDER BY timestamp DESC LIMIT $" + strconv.Itoa(argIndex) + " OFFSET $" + strconv.Itoa(argIndex+1)
	args = append(args, limit, offset)

	rows, err := h.db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "無法取得資料",
			"details": err.Error(),
		})
		return
	}
	defer rows.Close()

	var dataList []models.DeviceMetric
	for rows.Next() {
		var data models.DeviceMetric
		if err := rows.Scan(&data.ID, &data.DeviceID, &data.Voltage, &data.Current, 
			&data.Temperature, &data.Status, &data.Timestamp, &data.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "讀取資料時發生錯誤",
				"details": err.Error(),
			})
			return
		}
		dataList = append(dataList, data)
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id": deviceID,
		"count":     len(dataList),
		"data":      dataList,
	})
}

func (h *Handlers) GetDeviceLatest(c *gin.Context) {
	deviceID := c.Param("deviceId")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "deviceId 參數不能為空",
		})
		return
	}

	ctx := c.Request.Context()
	cacheKey := "device_metric:" + deviceID + ":latest"

	cached, err := h.rdb.Get(ctx, cacheKey).Result()
	if err == nil {
		var data models.DeviceMetric
		if jsonErr := json.Unmarshal([]byte(cached), &data); jsonErr == nil {
			c.JSON(http.StatusOK, gin.H{
				"data":   data,
				"source": "cache",
			})
			return
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
	err = h.db.QueryRow(query, deviceID).Scan(
		&data.ID, &data.DeviceID, &data.Voltage, &data.Current,
		&data.Temperature, &data.Status, &data.Timestamp, &data.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"error":    "找不到該設備的資料",
				"device_id": deviceID,
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "無法取得資料",
		})
		return
	}

	if jsonBytes, jsonErr := json.Marshal(data); jsonErr == nil {
		// TTL 設 60 秒
		h.rdb.Set(ctx, cacheKey, string(jsonBytes), 60*time.Second)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": data,
		"source": "database",
	})
}

func (h *Handlers) GetDevices(c *gin.Context) {
	query := `
		SELECT DISTINCT ON (device_id) 
			device_id, 
			timestamp as last_updated,
			status as latest_status
		FROM device_metrics
		ORDER BY device_id, timestamp DESC
	`

	rows, err := h.db.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "無法取得設備清單",
			"details": err.Error(),
		})
		return
	}
	defer rows.Close()

	var deviceList []models.DeviceListResponse
	for rows.Next() {
		var device models.DeviceListResponse
		if err := rows.Scan(&device.DeviceID, &device.LastUpdated, &device.LatestStatus); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "讀取資料時發生錯誤",
			"details": err.Error(),
			})
			return
		}
		deviceList = append(deviceList, device)
	}

	c.JSON(http.StatusOK, gin.H{
		"count":  len(deviceList),
		"devices": deviceList,
	})
}

func (h *Handlers) GetRedisKeys(c *gin.Context) {
	ctx := c.Request.Context()
	pattern := c.DefaultQuery("pattern", "*")

	limitStr := c.DefaultQuery("limit", "100")
	cursorStr := c.DefaultQuery("cursor", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 1000 {
		limit = 100 // 限制最多 1000 個，避免一次回傳太多
	}

	cursor, err := strconv.ParseUint(cursorStr, 10, 64)
	if err != nil {
		cursor = 0
	}

	var keys []string

	iter := h.rdb.Scan(ctx, cursor, pattern, int64(limit)).Iterator()
	scannedCount := 0

	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
		scannedCount++

		if scannedCount >= limit {
			break
		}
	}

	if err := iter.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "無法掃描 Redis keys",
			"details": err.Error(),
		})
		return
	}

	hasMore := scannedCount >= limit

	c.JSON(http.StatusOK, gin.H{
		"pattern":    pattern,
		"count":      len(keys),
		"keys":       keys,
		"cursor":     cursor,
		"has_more":   hasMore,
		"limit":      limit,
		"scanned":    scannedCount,
	})
}

func (h *Handlers) SetRedisValue(c *gin.Context) {
	ctx := c.Request.Context()
	key := c.Param("key")

	var req models.SetRedisValueRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "無效的請求資料",
			"details": err.Error(),
		})
		return
	}

	if req.TTL > 0 {
		err := h.rdb.Set(ctx, key, req.Value, time.Duration(req.TTL)*time.Second).Err()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "無法設定 Redis key",
				"details": err.Error(),
			})
			return
		}
	} else {
		err := h.rdb.Set(ctx, key, req.Value, 0).Err()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "無法設定 Redis key",
				"details": err.Error(),
			})
			return
		}
	}

	ttl, _ := h.rdb.TTL(ctx, key).Result()

	c.JSON(http.StatusOK, gin.H{
		"message": "Redis key 設定成功",
		"key":     key,
		"value":   req.Value,
		"ttl":     ttl.Seconds(),
	})
}

// GetRedisValue 取得 Redis 值
// GET /api/v1/redis/:key
func (h *Handlers) GetRedisValue(c *gin.Context) {
	ctx := c.Request.Context()
	key := c.Param("key")

	keyType, err := h.rdb.Type(ctx, key).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "無法取得 key 類型",
			"details": err.Error(),
		})
		return
	}

	exists, err := h.rdb.Exists(ctx, key).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "無法檢查 key 是否存在",
			"details": err.Error(),
		})
		return
	}

	if exists == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "key 不存在",
			"key":   key,
		})
		return
	}

	var value interface{}
	var ttl time.Duration

	switch keyType {
	case "string":
		value, err = h.rdb.Get(ctx, key).Result()
	case "hash":
		value, err = h.rdb.HGetAll(ctx, key).Result()
	case "list":
		value, err = h.rdb.LRange(ctx, key, 0, -1).Result()
	case "set":
		value, err = h.rdb.SMembers(ctx, key).Result()
	case "zset":
		value, err = h.rdb.ZRangeWithScores(ctx, key, 0, -1).Result()
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "不支援的 key 類型",
			"type":  keyType,
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "無法取得 key 值",
			"details": err.Error(),
		})
		return
	}

	ttl, _ = h.rdb.TTL(ctx, key).Result()

	c.JSON(http.StatusOK, gin.H{
		"key":   key,
		"type":  keyType,
		"value": value,
		"ttl":   ttl.Seconds(),
	})
}

func (h *Handlers) DeleteRedisKey(c *gin.Context) {
	ctx := c.Request.Context()
	key := c.Param("key")

	exists, err := h.rdb.Exists(ctx, key).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "無法檢查 key 是否存在",
			"details": err.Error(),
		})
		return
	}

	if exists == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "key 不存在",
			"key":   key,
		})
		return
	}

	deleted, err := h.rdb.Del(ctx, key).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "無法刪除 key",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "key 刪除成功",
		"key":     key,
		"deleted": deleted,
	})
}
