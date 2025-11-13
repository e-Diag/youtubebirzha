package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// SafeLoggerMiddleware логирует запросы без PII (персональных данных)
func SafeLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// Получаем user_id из контекста (если есть)
		var userID interface{}
		if uid, exists := c.Get("user_id"); exists {
			userID = uid
		}

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		// Логируем только безопасные данные
		if userID != nil {
			log.Printf("[%s] %s %s | %d | %v | user_id=%v",
				method, path, c.ClientIP(), statusCode, latency, userID)
		} else {
			log.Printf("[%s] %s %s | %d | %v",
				method, path, c.ClientIP(), statusCode, latency)
		}
	}
}

