# IoT Data Collection 專案

IoT 設備即時資料回報與查詢系統，使用 Go + PostgreSQL + Redis，採用微服務架構。

## 快速啟動

```bash
# 啟動所有服務
docker-compose up -d

# 查看服務狀態
docker-compose ps

# 查看日誌
docker-compose logs -f

# 停止服務
docker-compose down
```

## 服務說明

- **app**: 應用程式服務（Port 8080）
- **database**: PostgreSQL 資料庫（Port 5432）
- **cache**: Redis 快取（Port 6379）
- **seeder**: 資料產生腳本，模擬設備定期回報資料

## API 端點

Base URL: `http://localhost:8080`

### 1. 接收設備資料回報
**POST** `/api/v1/devices/{deviceId}/metrics`

```bash
curl -X POST http://localhost:8080/api/v1/devices/device-001/metrics \
  -H "Content-Type: application/json" \
  -d '{
    "voltage": 220.5,
    "current": 45.2,
    "temperature": 35.8,
    "status": "normal"
  }'
```

**參數說明：**
- `voltage`: 電壓 (V)，範圍 100-240V
- `current`: 電流 (A)，範圍 0-100A
- `temperature`: 溫度 (°C)，範圍 0-100°C
- `status`: 狀態，值為 `normal` / `warning` / `error`
- `timestamp`: 時間戳記（選填，RFC3339 格式）

### 2. 查詢單一設備的歷史資料
**GET** `/api/v1/devices/{deviceId}/metrics`

```bash
# 查詢所有歷史資料
curl http://localhost:8080/api/v1/devices/device-001/metrics

# 查詢指定時間區間
curl "http://localhost:8080/api/v1/devices/device-001/metrics?start_time=2024-01-01T00:00:00Z&end_time=2024-01-01T23:59:59Z"

# 使用分頁
curl "http://localhost:8080/api/v1/devices/device-001/metrics?limit=50&offset=0"
```

**查詢參數：**
- `start_time`: 開始時間（RFC3339 格式）
- `end_time`: 結束時間（RFC3339 格式）
- `limit`: 每頁筆數（預設 100）
- `offset`: 偏移量（預設 0）

### 3. 取得單一設備最新一筆資料
**GET** `/api/v1/devices/{deviceId}/latest`

```bash
curl http://localhost:8080/api/v1/devices/device-001/latest
```

### 4. 列出所有設備清單
**GET** `/api/v1/devices`

```bash
curl http://localhost:8080/api/v1/devices
```

### 5. 健康檢查
**GET** `/health`

```bash
curl http://localhost:8080/health
```

## 環境變數

可在 `.env` 檔案中設定（選填）：

```env
POSTGRES_USER=user
POSTGRES_PASSWORD=password
POSTGRES_DB=iot_db
APP_PORT=8080
NUM_DEVICES=8          # Seeder 模擬設備數量
INTERVAL_SECONDS=7    # Seeder 發送間隔（秒）
```
