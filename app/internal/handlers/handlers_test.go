package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"iot-data-collection/app/internal/mocks"
	"iot-data-collection/app/internal/service"

	"github.com/gin-gonic/gin"
)

func TestHealthCheck_Healthy(t *testing.T) {
	metricSvc := service.NewDeviceMetricService(&mocks.MockDB{}, &mocks.MockRedis{}, &mocks.MockMetricQueue{})
	h := NewHandlers(&mocks.MockDB{}, &mocks.MockRedis{}, metricSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	h.HealthCheck(c)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，得到 %d", w.Code)
	}
}

func TestHealthCheck_RedisDown(t *testing.T) {
	metricSvc := service.NewDeviceMetricService(&mocks.MockDB{}, &mocks.MockRedis{}, &mocks.MockMetricQueue{})
	h := NewHandlers(&mocks.MockDB{}, &mocks.MockRedis{
		PingErr: errors.New("redis連線失敗"),
	}, metricSvc)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	h.HealthCheck(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("期望 503，得到 %d", w.Code)
	}
}