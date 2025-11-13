package handlers

import (
	"net/http"
	"strings"
	"time"

	"youtube-market/internal/db"
	"youtube-market/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetProfileAds(c *gin.Context) {
	username := strings.TrimSpace(c.Param("username"))
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "username parameter is required"})
		return
	}

	username = strings.TrimPrefix(username, "@")

	var ads []models.Ad
	if err := db.DB.
		Where("LOWER(username) = LOWER(?)", username).
		Order(gorm.Expr("CASE WHEN status = ? THEN 0 WHEN status = ? THEN 1 ELSE 2 END, updated_at DESC", models.AdStatusActive, models.AdStatusExpired)).
		Find(&ads).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch profile ads"})
		return
	}

	now := time.Now()
	response := make([]AdView, 0, len(ads))
	for _, ad := range ads {
		// Ensure status reflects current expiration
		if ad.Status == models.AdStatusActive && ad.ExpiresAt.Before(now) {
			ad.Status = models.AdStatusExpired
		}
		response = append(response, buildAdView(ad))
	}

	c.JSON(http.StatusOK, response)
}
