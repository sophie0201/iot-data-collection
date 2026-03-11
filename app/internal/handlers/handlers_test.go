package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"iot-data-collection/app/internal/mocks"

	"github.com/gin-gonic/gin"
)

func TestHealthCheck_Healthy(t *testing.T) {
	h := NewHandlers(&mocks.MockDB{}, &mocks.MockRedis{})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	h.HealthCheck(c)

	if w.Code != http.StatusOK {
		t.Errorf("期望 200，得到 %d", w.Code)
	}
}

func TestHealthCheck_RedisDown(t *testing.T) {
	h := NewHandlers(&mocks.MockDB{}, &mocks.MockRedis{
		PingErr: errors.New("redis連線失敗"),
	})

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	h.HealthCheck(c)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("期望 503，得到 %d", w.Code)
	}
}