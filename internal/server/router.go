package server

import (
	"net/http"

	"github.com/example/go-llm-proxy/internal/auth"
	"github.com/example/go-llm-proxy/internal/config"
	"github.com/example/go-llm-proxy/internal/converter"
	"github.com/example/go-llm-proxy/internal/proxy"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SetupRouter sets up the HTTP router
func SetupRouter(
	configManager *config.ConfigManager,
	converter *converter.Converter,
	authManager *auth.SessionManager,
	logger *logrus.Logger,
) *gin.Engine {

	if logger == nil {
		logger = logrus.New()
		logger.SetLevel(logrus.InfoLevel)
	}

	router := gin.Default()

	// Global middleware
	router.Use(CorsMiddleware())
	router.Use(LoggingMiddleware(logger))
	router.Use(RecoveryMiddleware())
	router.Use(MetricsMiddleware())
	router.Use(authManager.Middleware())

	// Health check (no auth required)
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
		})
	})

	// Metrics endpoint (no auth required for simplicity)
	router.GET("/metrics", MetricsHandler())

	// API v1 routes
	v1 := router.Group("/v1")
	{
		// Proxy routes (no auth required)
		proxyHandler := proxy.NewProxyHandler(configManager, converter, logger)
		v1.POST("/chat/completions", proxyHandler.HandleChatCompletions)
		v1.GET("/models", proxyHandler.HandleModels)
	}

	// Admin routes
	admin := router.Group("/admin")
	{
		// Login routes (no auth required)
		admin.GET("/login", func(c *gin.Context) {
			c.String(http.StatusOK, "Login Page - POST to /admin/do-login")
		})

		admin.POST("/do-login", func(c *gin.Context) {
			username := c.PostForm("username")
			password := c.PostForm("password")

			if sessionID, ok := authManager.Login(c, username, password, 24*time.Hour); ok {
				c.Redirect(http.StatusFound, "/admin")
			} else {
				c.Redirect(http.StatusFound, "/admin/login?error=1")
			}
		})

		// Protected admin routes
		adminHandler := NewHandler(configManager, proxy.NewProxyHandler(configManager, converter, logger), logger)
		admin.GET("/", func(c *gin.Context) {
			c.String(http.StatusOK, "Admin Dashboard - Providers API at /admin/api/v1/providers")
		})

		// API routes under admin
		api := admin.Group("/api/v1")
		{
			api.GET("/providers", adminHandler.GetProviders)
			api.POST("/providers", adminHandler.AddProvider)
			api.PUT("/providers/:id", adminHandler.UpdateProvider)
			api.DELETE("/providers/:id", adminHandler.DeleteProvider)
			api.POST("/providers/:id/toggle", adminHandler.ToggleProvider)
		}
	}

	return router
}
