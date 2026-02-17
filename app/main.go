package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Hello, World!",
			"status":  "success",
		})
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
		})
	})

	// 啟動伺服器
	port := "8080"
	log.Printf("伺服器啟動在 Port: %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("伺服器啟動失敗 Port: %s, Error: %v", port, err)
	}
}