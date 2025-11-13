package main

import (
	"log"
	"os"
	"youtube-market/internal/db"
	"youtube-market/internal/handlers"
	"youtube-market/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		// В контейнере .env может отсутствовать — переменные окружения передаются напрямую.
		if !os.IsNotExist(err) {
			log.Printf("Warning: could not load .env file: %v", err)
		}
	}

	// Initialize database
	if err := db.Init(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Initialize Redis for rate limiting
	if err := middleware.InitRedis(); err != nil {
		log.Printf("Warning: Redis not available, rate limiting disabled: %v", err)
	}

	// Setup router
	r := setupRouter()

	// Start manager bot in background
	go handlers.RunManagerBot()

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func setupRouter() *gin.Engine {
	// Set release mode in production
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// Global middleware
	r.Use(middleware.SafeLoggerMiddleware())
	r.Use(middleware.CORSMiddleware())
	r.Use(middleware.RateLimitMiddleware())

	// Static files
	r.Static("/static", "./static")
	r.Static("/assets", "./static/assets")
	r.GET("/", func(c *gin.Context) {
		// Проверяем существование файла перед отправкой
		if _, err := os.Stat("./static/index.html"); os.IsNotExist(err) {
			log.Printf("Warning: static/index.html not found, serving 404")
			c.JSON(404, gin.H{"error": "index.html not found"})
			return
		}
		c.File("./static/index.html")
	})

	// Legal pages
	r.GET("/terms", func(c *gin.Context) {
		if _, err := os.Stat("./static/terms.html"); os.IsNotExist(err) {
			c.JSON(404, gin.H{"error": "terms.html not found"})
			return
		}
		c.File("./static/terms.html")
	})
	r.GET("/privacy", func(c *gin.Context) {
		if _, err := os.Stat("./static/privacy.html"); os.IsNotExist(err) {
			c.JSON(404, gin.H{"error": "privacy.html not found"})
			return
		}
		c.File("./static/privacy.html")
	})

	// Metrics endpoint (без аутентификации для мониторинга)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// API routes with TMA authentication
	api := r.Group("/api")
	api.Use(middleware.TMAuthMiddleware())
	{
		api.GET("/ads", handlers.GetAds)
		api.GET("/ads/:id/photo", handlers.GetAdPhoto)
		api.GET("/myads", handlers.GetMyAds)
		api.GET("/profile/:username", handlers.GetProfileAds)
		api.GET("/scammer/:username", handlers.CheckScammer)
		api.GET("/blacklist", handlers.GetBlacklist)
	}

	return r
}
