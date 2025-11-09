package server

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/auth"
	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/logging"
	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/middleware"
	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/proxy"
	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/server/handlers"
)

// RouteConfig holds configuration for setting up routes
type RouteConfig struct {
	AuthService      *auth.AuthService
	AuditLogger      *logging.AuditLogger
	ProxyAuditLogger *logging.ProxyAuditLogger
	Logger           *logrus.Logger

	// Proxy components
	ProxyHandler     *proxy.ProxyHandler
	StreamingHandler *proxy.StreamingHandler
	LoadBalancer     proxy.LoadBalancer
	ModelMapper      *proxy.ModelMapper

	// Middleware config
	EnableRateLimit  bool
	EnableValidation bool
}

// SetupRoutes configures all application routes
func SetupRoutes(router *gin.Engine, config *RouteConfig) {
	// Create handlers
	authHandler := handlers.NewAuthHandler(config.AuthService, config.AuditLogger, config.Logger)

	// Public routes (no authentication required)
	setupPublicRoutes(router)

	// Admin authentication routes (no auth middleware)
	setupAdminAuthRoutes(router, authHandler)

	// Admin API routes (require authentication)
	setupAdminAPIRoutes(router, config.AuthService, authHandler)

	// Proxy API routes (require authentication or API key)
	setupProxyAPIRoutes(router, config)
}

// setupPublicRoutes configures public routes
func setupPublicRoutes(router *gin.Engine) {
	// Health check endpoints
	router.GET("/healthz", handleHealthCheck)
	router.GET("/healthz/ready", handleReadyCheck)
	router.GET("/healthz/live", handleLiveCheck)

	// Metrics endpoint (can be public or restricted based on requirements)
	router.GET("/metrics", handleMetrics)
}

// setupAdminAuthRoutes configures admin authentication routes
func setupAdminAuthRoutes(router *gin.Engine, authHandler *handlers.AuthHandler) {
	adminAuth := router.Group("/admin/api/v1/auth")
	{
		// CSRF protection for auth routes
		adminAuth.Use(middleware.SimpleCSRFMiddleware())

		adminAuth.POST("/login", authHandler.HandleLogin)
		adminAuth.POST("/logout", authHandler.HandleLogout)
		adminAuth.POST("/refresh", authHandler.HandleRefreshSession)
		adminAuth.GET("/me", authHandler.HandleGetCurrentUser)
	}
}

// setupAdminAPIRoutes configures protected admin API routes
func setupAdminAPIRoutes(router *gin.Engine, authService *auth.AuthService, authHandler *handlers.AuthHandler) {
	admin := router.Group("/admin/api/v1")

	// Apply authentication middleware
	admin.Use(middleware.AuthMiddleware(authService))

	// Require admin role
	admin.Use(middleware.RequireRole("admin"))

	// CSRF protection
	admin.Use(middleware.SimpleCSRFMiddleware())

	// Provider management endpoints (will be implemented in Phase 5)
	providers := admin.Group("/providers")
	{
		// These handlers will be created in Phase 5 (US1)
		providers.GET("", placeholderHandler("List providers"))
		providers.POST("", placeholderHandler("Create provider"))
		providers.GET("/:id", placeholderHandler("Get provider"))
		providers.PUT("/:id", placeholderHandler("Update provider"))
		providers.DELETE("/:id", placeholderHandler("Delete provider"))
		providers.POST("/:id/toggle", placeholderHandler("Toggle provider"))
	}

	// Model mapping endpoints
	models := admin.Group("/model-mappings")
	{
		models.GET("", placeholderHandler("List model mappings"))
		models.POST("", placeholderHandler("Create model mapping"))
		models.DELETE("/:id", placeholderHandler("Delete model mapping"))
	}

	// Provider status endpoints (Phase 6)
	status := admin.Group("/providers/status")
	{
		status.GET("", placeholderHandler("Get all provider statuses"))
		status.GET("/:id", placeholderHandler("Get provider status"))
		status.POST("/:id/priority", placeholderHandler("Update provider priority"))
	}

	// Configuration endpoints
	configGroup := admin.Group("/config")
	{
		configGroup.GET("/export", placeholderHandler("Export configuration"))
		configGroup.POST("/import", placeholderHandler("Import configuration"))
		configGroup.POST("/reload", placeholderHandler("Reload configuration"))
	}

	// Audit log endpoints
	audit := admin.Group("/audit")
	{
		audit.GET("/logs", placeholderHandler("Get audit logs"))
		audit.GET("/logs/:id", placeholderHandler("Get audit log"))
	}
}

// setupProxyAPIRoutes configures proxy API routes (Phase 4)
func setupProxyAPIRoutes(router *gin.Engine, config *RouteConfig) {
	// Create proxy handlers
	chatCompletionsHandler := handlers.NewChatCompletionsHandler(&handlers.ChatCompletionsConfig{
		ProxyHandler:     config.ProxyHandler,
		StreamingHandler: config.StreamingHandler,
		ModelMapper:      config.ModelMapper,
		ProxyAuditLogger: config.ProxyAuditLogger,
		Logger:           config.Logger,
	})

	completionsHandler := handlers.NewCompletionsHandler(&handlers.CompletionsConfig{
		ProxyHandler:     config.ProxyHandler,
		StreamingHandler: config.StreamingHandler,
		ModelMapper:      config.ModelMapper,
		ProxyAuditLogger: config.ProxyAuditLogger,
		Logger:           config.Logger,
	})

	embeddingsHandler := handlers.NewEmbeddingsHandler(&handlers.EmbeddingsConfig{
		ProxyHandler:     config.ProxyHandler,
		ModelMapper:      config.ModelMapper,
		ProxyAuditLogger: config.ProxyAuditLogger,
		Logger:           config.Logger,
	})

	modelsHandler := handlers.NewModelsHandler(&handlers.ModelsConfig{
		LoadBalancer:     config.LoadBalancer,
		ModelMapper:      config.ModelMapper,
		ProxyAuditLogger: config.ProxyAuditLogger,
		Logger:           config.Logger,
	})

	// OpenAI-compatible API routes
	v1 := router.Group("/v1")

	// Apply middleware
	// 1. CORS (if needed)
	v1.Use(middleware.CORSMiddleware([]string{"*"}))

	// 2. Optional authentication - can be public or authenticated based on configuration
	v1.Use(middleware.OptionalAuthMiddleware(config.AuthService))

	// 3. Request validation
	if config.EnableValidation {
		v1.Use(middleware.ValidationMiddleware(middleware.DefaultValidationConfig()))
	}

	// 4. Rate limiting
	if config.EnableRateLimit {
		rateLimitConfig := &middleware.RateLimitConfig{
			RequestsPerMinute: 60,
			BurstSize:         10,
			KeyFunc:           middleware.UserBasedKeyFunc,
			CleanupInterval:   5 * time.Minute,
		}
		v1.Use(middleware.RateLimitMiddleware(rateLimitConfig, config.Logger))
	}

	// Proxy endpoints
	v1.POST("/chat/completions", chatCompletionsHandler.HandleChatCompletions)
	v1.POST("/completions", completionsHandler.HandleCompletions)
	v1.POST("/embeddings", embeddingsHandler.HandleEmbeddings)
	v1.GET("/models", modelsHandler.HandleModels)
	v1.GET("/models/:model", modelsHandler.HandleModelDetails)
}

// Health check handlers

func handleHealthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status":  "healthy",
		"service": "go-llm-proxy",
	})
}

func handleReadyCheck(c *gin.Context) {
	// Check if service is ready (database connections, etc.)
	c.JSON(200, gin.H{
		"status": "ready",
	})
}

func handleLiveCheck(c *gin.Context) {
	// Liveness check - service is running
	c.JSON(200, gin.H{
		"status": "alive",
	})
}

func handleMetrics(c *gin.Context) {
	// Prometheus metrics will be implemented here
	c.String(200, "# Metrics endpoint - Prometheus format\n")
}

// placeholderHandler creates a placeholder handler for unimplemented routes
func placeholderHandler(description string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(501, gin.H{
			"error":    "not implemented",
			"endpoint": description,
			"message":  "This endpoint will be implemented in a future phase",
		})
	}
}
