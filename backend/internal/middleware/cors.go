package middleware

import (
	"github.com/gin-gonic/gin"
)

// CORSMiddleware настраивает CORS только для указанного домена
func CORSMiddleware() gin.HandlerFunc {
	allowedOrigin := "https://5997551-tm19392.twc1.net"
	
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		
		// Разрешаем только указанный домен
		if origin == allowedOrigin {
			c.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		}
		
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, init_data")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

