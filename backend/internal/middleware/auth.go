package middleware

import (
	"net/http"
	"os"
	"youtube-market/internal/telegram"

	"github.com/gin-gonic/gin"
)

// TMAuthMiddleware проверяет init_data от Telegram Mini App
func TMAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Получаем init_data из заголовка или query параметра
		initData := c.GetHeader("init_data")
		if initData == "" {
			initData = c.Query("init_data")
		}

		botToken := os.Getenv("BOT_TOKEN")
		if botToken == "" {
			// Если BOT_TOKEN не установлен, пропускаем проверку
			c.Next()
			return
		}

		if initData == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "init_data is required"})
			c.Abort()
			return
		}

		// Валидируем init_data
		data, valid := telegram.ValidateInitData(initData, botToken)
		if !valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid init_data"})
			c.Abort()
			return
		}

		// Извлекаем user_id и username
		userID, err := telegram.ExtractUserID(data)
		if err == nil {
			c.Set("user_id", userID)
		}

		username := telegram.ExtractUsername(data)
		if username != "" {
			c.Set("username", username)
		}

		// Сохраняем все данные для дальнейшего использования
		c.Set("init_data", data)

		c.Next()
	}
}

