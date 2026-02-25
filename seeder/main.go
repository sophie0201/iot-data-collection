package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

type DeviceMetricRequest struct {
	Voltage     float64 `json:"voltage"`
	Current     float64 `json:"current"`
	Temperature float64 `json:"temperature"`
	Status      string  `json:"status"`
	Timestamp   string  `json:"timestamp,omitempty"`
}

// 設備狀態選項
var statuses = []string{"normal", "warning", "error"}

// 模擬設備資料產生器
type DeviceSimulator struct {
	DeviceID    string
	BaseVoltage float64 // 基礎電壓值
	BaseCurrent float64 // 基礎電流值
	BaseTemp    float64 // 基礎溫度值
	APIURL      string  // API 端點 URL
}

// GenerateMetric 產生一筆模擬資料
func (d *DeviceSimulator) GenerateMetric() DeviceMetricRequest {
	rand.Seed(time.Now().UnixNano())

	// 產生電壓值（100-240V，在基礎值附近波動）
	voltage := d.BaseVoltage + (rand.Float64()*20 - 10) // ±10V 波動
	if voltage < 100 {
		voltage = 100
	}
	if voltage > 240 {
		voltage = 240
	}

	// 產生電流值（0-100A，在基礎值附近波動）
	current := d.BaseCurrent + (rand.Float64()*10 - 5) // ±5A 波動
	if current < 0 {
		current = 0
	}
	if current > 100 {
		current = 100
	}

	// 產生溫度值（0-100°C，在基礎值附近波動）
	temperature := d.BaseTemp + (rand.Float64()*10 - 5) // ±5°C 波動
	if temperature < 0 {
		temperature = 0
	}
	if temperature > 100 {
		temperature = 100
	}

	// 根據數值決定狀態
	status := "normal"
	if voltage < 110 || voltage > 230 || current > 90 || temperature > 80 {
		status = "warning"
	}
	if voltage < 105 || voltage > 235 || current > 95 || temperature > 90 {
		status = "error"
	}
	// 隨機加入一些狀態變化
	if rand.Float64() < 0.1 { // 10% 機率改變狀態
		status = statuses[rand.Intn(len(statuses))]
	}

	return DeviceMetricRequest{
		Voltage:     roundToTwoDecimals(voltage),
		Current:     roundToTwoDecimals(current),
		Temperature: roundToTwoDecimals(temperature),
		Status:      status,
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

// SendMetric 發送資料到 API
func (d *DeviceSimulator) SendMetric(metric DeviceMetricRequest) error {
	url := fmt.Sprintf("%s/api/v1/devices/%s/metrics", d.APIURL, d.DeviceID)

	jsonData, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("序列化資料失敗: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("建立請求失敗: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("發送請求失敗: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("API 回應錯誤: status code %d", resp.StatusCode)
	}

	return nil
}

// roundToTwoDecimals 四捨五入到小數點後兩位
func roundToTwoDecimals(val float64) float64 {
	return float64(int(val*100+0.5)) / 100
}

func main() {
	apiURL := os.Getenv("API_URL")
	if apiURL == "" {
		apiURL = "http://app:8080"
	}

	// 預設 8 台
	numDevices := 8
	if envNum := os.Getenv("NUM_DEVICES"); envNum != "" {
		fmt.Sscanf(envNum, "%d", &numDevices)
	}

	// 發送間隔預設為 7 秒
	intervalSeconds := 7
	if envInterval := os.Getenv("INTERVAL_SECONDS"); envInterval != "" {
		fmt.Sscanf(envInterval, "%d", &intervalSeconds)
	}

	log.Printf("開始模擬 %d 台設備，每 %d 秒發送一次資料到 %s", numDevices, intervalSeconds, apiURL)

	// 建立多台設備模擬器
	simulators := make([]*DeviceSimulator, numDevices)
	for i := 0; i < numDevices; i++ {
		deviceID := fmt.Sprintf("device-%03d", i+1)
		
		// 每台設備有不同的數值
		simulators[i] = &DeviceSimulator{
			DeviceID:    deviceID,
			BaseVoltage: 200 + float64(i%5)*10,  // 200-240V
			BaseCurrent: 20 + float64(i%4)*15,    // 20-65A
			BaseTemp:    25 + float64(i%6)*10,    // 25-75°C
			APIURL:      apiURL,
		}
		log.Printf("初始化設備: %s (電壓: %.2fV, 電流: %.2fA, 溫度: %.2f°C)", 
			deviceID, simulators[i].BaseVoltage, simulators[i].BaseCurrent, simulators[i].BaseTemp)
	}

	// 等待 API 服務就緒
	log.Println("等待 API 服務就緒...")
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		resp, err := http.Get(apiURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			log.Println("API 服務已就緒")
			break
		}
		if i == maxRetries-1 {
			log.Fatal("API 服務無法連線，請確認服務是否正常啟動")
		}
		time.Sleep(2 * time.Second)
	}

	// 開始定期發送資料
	interval := time.Duration(intervalSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Println("開始發送資料...")

	// 立即發送第一筆資料
	for _, sim := range simulators {
		metric := sim.GenerateMetric()
		if err := sim.SendMetric(metric); err != nil {
			log.Printf("設備 %s 發送資料失敗: %v", sim.DeviceID, err)
		} else {
			log.Printf("設備 %s 發送資料成功: 電壓=%.2fV, 電流=%.2fA, 溫度=%.2f°C, 狀態=%s",
				sim.DeviceID, metric.Voltage, metric.Current, metric.Temperature, metric.Status)
		}
	}

	// 定期發送資料
	for range ticker.C {
		for _, sim := range simulators {
			metric := sim.GenerateMetric()
			if err := sim.SendMetric(metric); err != nil {
				log.Printf("設備 %s 發送資料失敗: %v", sim.DeviceID, err)
			} else {
				log.Printf("設備 %s 發送資料成功: 電壓=%.2fV, 電流=%.2fA, 溫度=%.2f°C, 狀態=%s",
					sim.DeviceID, metric.Voltage, metric.Current, metric.Temperature, metric.Status)
			}
			// 錯開每台設備的發送時間，避免同時發送
			time.Sleep(100 * time.Millisecond)
		}
	}
}
