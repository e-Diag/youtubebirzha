package handlers

import (
	"net/http"
	"strings"

	"youtube-market/internal/db"
	"youtube-market/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func CheckScammer(c *gin.Context) {
	username := strings.TrimSpace(strings.TrimPrefix(c.Param("username"), "@"))
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username parameter is required"})
		return
	}

	var user models.User
	err := db.DB.Where("LOWER(username) = LOWER(?) AND is_scammer = true", username).First(&user).Error

	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusOK, gin.H{
			"safe": true,
			"msg":  "Юзер не был замечен в мошеннических схемах",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"safe": false,
		"msg":  "Осторожно! Мошенник",
	})
}

func GetBlacklist(c *gin.Context) {
	var scammers []models.User
	if err := db.DB.
		Where("is_scammer = ?", true).
		Order("username ASC").
		Find(&scammers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load blacklist"})
		return
	}

	response := make([]gin.H, 0, len(scammers))
	for _, user := range scammers {
		response = append(response, gin.H{
			"username":   user.Username,
			"created_at": user.CreatedAt,
			"updated_at": user.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, response)
}
