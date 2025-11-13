package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"youtube-market/internal/db"
	"youtube-market/internal/models"

	"github.com/gin-gonic/gin"
)

func GetAdPhoto(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ad id"})
		return
	}

	var ad models.Ad
	if err := db.DB.First(&ad, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ad not found"})
		return
	}

	if ad.PhotoPath == "" {
		c.Status(http.StatusNotFound)
		return
	}

	token := getBotToken()
	if token == "" {
		c.Status(http.StatusNotFound)
		return
	}

	url := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", token, ad.PhotoPath)
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		if resp != nil {
			resp.Body.Close()
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to fetch photo"})
		return
	}
	defer resp.Body.Close()

	for k, values := range resp.Header {
		if len(values) == 0 {
			continue
		}
		switch k {
		case "Content-Type", "Content-Length":
			c.Header(k, values[0])
		}
	}

	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, resp.Body)
}

