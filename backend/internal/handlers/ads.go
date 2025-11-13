package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"youtube-market/internal/db"
	"youtube-market/internal/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const maxPremiumActiveAds = 3

func GetAds(c *gin.Context) {
	now := time.Now()

	category := strings.TrimSpace(c.Query("cat"))
	mode := strings.TrimSpace(c.Query("mode"))
	tag := strings.TrimSpace(c.Query("tag"))

	log.Printf("GetAds: запрос - category=%s, mode=%s, tag=%s", category, mode, tag)

	filteredQuery := db.DB.Where("status = ? AND expires_at > ?", models.AdStatusActive, now)

	if category != "" {
		filteredQuery = filteredQuery.Where("category = ?", category)
	}
	// Для категории "other" не применяем фильтр по mode, так как режим всегда "general"
	if mode != "" && category != "other" {
		filteredQuery = filteredQuery.Where("mode = ?", mode)
	}
	if tag != "" {
		if strings.EqualFold(tag, "all") {
			// ignore tag filter when "all"
		} else {
			filteredQuery = filteredQuery.Where("tag = ?", tag)
		}
	}

	filteredQuery = filteredQuery.Order("is_premium DESC, updated_at DESC")

	var filtered []models.Ad
	if err := filteredQuery.Find(&filtered).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch ads"})
		return
	}

	// Премиум объявления должны фильтроваться по тем же параметрам, что и обычные
	var premium []models.Ad
	premiumQuery := db.DB.Where("status = ? AND expires_at > ? AND is_premium = ?", models.AdStatusActive, now, true)

	if category != "" {
		premiumQuery = premiumQuery.Where("category = ?", category)
	}
	// Для категории "other" не применяем фильтр по mode, так как режим всегда "general"
	if mode != "" && category != "other" {
		premiumQuery = premiumQuery.Where("mode = ?", mode)
	}
	if tag != "" && !strings.EqualFold(tag, "all") {
		premiumQuery = premiumQuery.Where("tag = ?", tag)
	}

	premiumQuery = premiumQuery.Order("updated_at DESC")

	if err := premiumQuery.Find(&premium).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch premium ads"})
		return
	}

	combined := mergeAds(premium, filtered)

	log.Printf("GetAds: category=%s, mode=%s, tag=%s, найдено filtered=%d, premium=%d, combined=%d",
		category, mode, tag, len(filtered), len(premium), len(combined))

	response := make([]AdView, 0, len(combined))
	for _, ad := range combined {
		response = append(response, buildAdView(ad))
	}

	c.JSON(http.StatusOK, response)
}

func GetMyAds(c *gin.Context) {
	userIDStr := c.Query("user_id")
	log.Printf("GetMyAds: получен запрос с user_id=%s", userIDStr)
	if userIDStr == "" {
		log.Printf("GetMyAds: ошибка - user_id не указан")
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id parameter is required"})
		return
	}

	// Ищем объявления по ClientID (который менеджер вводит во время создания объявления)
	// ClientID совпадает с user_id из Telegram
	// Также ищем по UserID на случай если client_id не совпадает
	var ads []models.Ad
	query := db.DB.Where("client_id = ?", userIDStr)

	// Также пробуем найти по user_id (на случай если client_id не установлен правильно)
	userIDInt, err := strconv.ParseInt(userIDStr, 10, 64)
	if err == nil {
		query = query.Or("user_id = ?", userIDInt)
	}

	if err := query.
		Order(gorm.Expr("CASE WHEN status = ? THEN 0 WHEN status = ? THEN 1 ELSE 2 END, updated_at DESC", models.AdStatusActive, models.AdStatusExpired)).
		Find(&ads).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch ads"})
		return
	}

	log.Printf("GetMyAds: найдено %d объявлений для user_id=%s", len(ads), userIDStr)
	for _, ad := range ads {
		log.Printf("  - Ad ID=%d, ClientID=%s, UserID=%d, Status=%s", ad.ID, ad.ClientID, ad.UserID, ad.Status)
	}

	response := make([]AdView, 0, len(ads))
	for _, ad := range ads {
		response = append(response, buildAdView(ad))
	}

	c.JSON(http.StatusOK, response)
}

func activePremiumCount(excludeID *uint) (int64, error) {
	query := db.DB.Model(&models.Ad{}).Where("status = ? AND is_premium = ? AND expires_at > ?", models.AdStatusActive, true, time.Now())
	if excludeID != nil {
		query = query.Where("id <> ?", *excludeID)
	}

	var count int64
	err := query.Count(&count).Error
	return count, err
}
