package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/server/handlers"
)

// Server represents the HTTP server
type Server struct {
	router      *gin.Engine
	httpServer  *http.Server
	logger      *logrus.Logger
	config      *ServerConfig
	metrics     *Metrics
	shutdownCtx context.Context
	shutdownFn  context.CancelFunc
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port            int
	TLSPort         int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	EnableTLS       bool
	EnableHTTP2     bool
	EnableCORS      bool
	DataDir         string
}

// NewServer creates a new server instance
func NewServer(config *ServerConfig, logger *logrus.Logger) *Server {
	// Create Gin router
	if logger != nil {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Initialize metrics
	metrics := NewMetrics()

	return &Server{
		router:      router,
		config:      config,
		logger:      logger,
		metrics:     metrics,
		shutdownCtx: context.Background(),
	}
}

// SetupRoutes sets up all routes
func (s *Server) SetupRoutes() {
	// Add recovery middleware
	s.router.Use(gin.Recovery())

	// Health check endpoints
	s.router.GET("/healthz", handleHealthCheck)
	s.router.GET("/healthz/ready", handleReadyCheck)
	s.router.GET("/healthz/live", handleLiveCheck)

	// Metrics endpoint
	s.setupMetricsRoutes()

	// For now, just add a simple proxy route to test
	// In production, this would be properly configured with all components
	s.router.POST("/v1/chat/completions", handleTestChatCompletions)
}

// SetupAdminRoutes sets up admin API routes
func (s *Server) SetupAdminRoutes() {
	// Setup admin API routes
	_ = s.router.Group("/admin/api/v1")
	{
		// Provider routes - placeholder
		// These will be set up by the main setup method with proper dependencies
		// Placeholder for now - will be initialized in SetupFullRoutes

		// Model mapping routes - placeholder
		// These will be set up by the main setup method with proper dependencies
		// Placeholder for now - will be initialized in SetupFullRoutes
	}
}

// SetupFullRoutes sets up all routes with proper dependencies
func (s *Server) SetupFullRoutes() {
	// Add recovery middleware
	s.router.Use(gin.Recovery())

	// Health check endpoints
	s.setupHealthRoutes()

	// Metrics endpoint
	s.setupMetricsRoutes()

	// Setup admin API routes with dependencies
	s.setupProviderAdminRoutes()

	// Proxy routes
	s.router.POST("/v1/chat/completions", handleTestChatCompletions)
}

// setupProviderAdminRoutes sets up provider admin routes with proper dependencies
func (s *Server) setupProviderAdminRoutes() {
	// For now, use simple initialization
	// In a full implementation, these would be injected from main
	admin := s.router.Group("/admin/api/v1")
	{
		// Provider routes
		providerGroup := admin.Group("/providers")
		{
			// GET /admin/api/v1/providers - list all providers
			providerGroup.GET("/", func(c *gin.Context) {
				c.JSON(200, gin.H{
					"message": "Provider list endpoint - to be implemented with full dependencies",
				})
			})

			// GET /admin/api/v1/providers/status - get provider status
			providerGroup.GET("/status", func(c *gin.Context) {
				c.JSON(200, gin.H{
					"message": "Provider status endpoint - to be implemented with full dependencies",
				})
			})

			// POST /admin/api/v1/providers - create provider
			providerGroup.POST("/", func(c *gin.Context) {
				c.JSON(200, gin.H{
					"message": "Provider create endpoint - to be implemented with full dependencies",
				})
			})

			// PUT /admin/api/v1/providers/:id - update provider
			providerGroup.PUT("/:id", func(c *gin.Context) {
				c.JSON(200, gin.H{
					"message": "Provider update endpoint - to be implemented with full dependencies",
				})
			})

			// DELETE /admin/api/v1/providers/:id - delete provider
			providerGroup.DELETE("/:id", func(c *gin.Context) {
				c.JSON(200, gin.H{
					"message": "Provider delete endpoint - to be implemented with full dependencies",
				})
			})

			// POST /admin/api/v1/providers/:id/toggle - toggle provider
			providerGroup.POST("/:id/toggle", func(c *gin.Context) {
				c.JSON(200, gin.H{
					"message": "Provider toggle endpoint - to be implemented with full dependencies",
				})
			})

			// POST /admin/api/v1/providers/:id/priority - update provider priority
			providerGroup.POST("/:id/priority", func(c *gin.Context) {
				c.JSON(200, gin.H{
					"message": "Provider priority endpoint - to be implemented with full dependencies",
				})
			})
		}

		// Model mapping routes
		mappingGroup := admin.Group("/model-mappings")
		{
			// GET /admin/api/v1/model-mappings - list all mappings
			mappingGroup.GET("/", func(c *gin.Context) {
				c.JSON(200, gin.H{
					"message": "Model mapping list endpoint - to be implemented with full dependencies",
				})
			})

			// POST /admin/api/v1/model-mappings - create mapping
			mappingGroup.POST("/", func(c *gin.Context) {
				c.JSON(200, gin.H{
					"message": "Model mapping create endpoint - to be implemented with full dependencies",
				})
			})

			// PUT /admin/api/v1/model-mappings/:id - update mapping
			mappingGroup.PUT("/:id", func(c *gin.Context) {
				c.JSON(200, gin.H{
					"message": "Model mapping update endpoint - to be implemented with full dependencies",
				})
			})

			// DELETE /admin/api/v1/model-mappings/:id - delete mapping
			mappingGroup.DELETE("/:id", func(c *gin.Context) {
				c.JSON(200, gin.H{
					"message": "Model mapping delete endpoint - to be implemented with full dependencies",
				})
			})
		}
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	s.SetupRoutes()

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.Port),
		Handler:      s.router,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		IdleTimeout:  s.config.IdleTimeout,
	}

	// Start server in a goroutine
	go func() {
		s.logger.Infof("Starting HTTP server on port %d", s.config.Port)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	s.logger.Info("Server started successfully")
	return nil
}

// StartWithTLS starts the server with TLS
func (s *Server) StartWithTLS(certPath, keyPath string) error {
	s.SetupRoutes()

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.config.Port),
		Handler:      s.router,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		IdleTimeout:  s.config.IdleTimeout,
	}

	// Start server with TLS
	go func() {
		s.logger.Infof("Starting HTTPS server on port %d", s.config.Port)
		if err := s.httpServer.ListenAndServeTLS(certPath, keyPath); err != nil && err != http.ErrServerClosed {
			s.logger.Fatalf("Failed to start TLS server: %v", err)
		}
	}()

	s.logger.Info("TLS Server started successfully")
	return nil
}

// Stop stops the server gracefully
func (s *Server) Stop() error {
	s.logger.Info("Shutting down server...")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()

	// Shutdown HTTP server
	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	s.logger.Info("Server stopped successfully")
	return nil
}

// WaitForInterrupt waits for interrupt signal and shuts down gracefully
func (s *Server) WaitForInterrupt() {
	// Create channel for OS signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	s.logger.Info("Received shutdown signal")

	// Stop server
	if err := s.Stop(); err != nil {
		s.logger.Errorf("Error during server shutdown: %v", err)
	}
}

// Router returns the Gin router
func (s *Server) Router() *gin.Engine {
	return s.router
}

// GetMetrics returns the server metrics
func (s *Server) GetMetrics() *Metrics {
	return s.metrics
}

// setupHealthRoutes sets up health check routes with handlers
func (s *Server) setupHealthRoutes() {
	// Create handlers
	healthHandler := handlers.NewHealthzHandler(s.logger)
	readinessHandler := handlers.NewReadinessHandler(s.logger)
	detailedHealthHandler := handlers.NewDetailedHealthHandler(s.logger)
	metricsHandler := handlers.NewMetricsHandler(s.logger)

	// Wire health check routes
	s.router.GET("/healthz", healthHandler.HandleHealthz)
	s.router.GET("/healthz/live", healthHandler.HandleLive)
	s.router.GET("/healthz/ready", readinessHandler.HandleReady)
	s.router.GET("/healthz/detailed", detailedHealthHandler.HandleDetailed)

	// Wire metrics routes
	s.router.GET("/metrics", metricsHandler.HandleMetrics)
	s.router.GET("/metrics/config", metricsHandler.HandleMetricsConfig)
}

// handleTestChatCompletions is a simple test handler
func handleTestChatCompletions(c *gin.Context) {
	// For testing, just return a simple response
	c.JSON(200, gin.H{
		"id":      "test-response",
		"object":  "chat.completion",
		"created": 1234567890,
		"model":   "gpt-4",
		"choices": []gin.H{
			{
				"index": 0,
				"message": gin.H{
					"role":    "assistant",
					"content": "This is a test response from the proxy.",
				},
				"finish_reason": "stop",
			},
		},
		"usage": gin.H{
			"prompt_tokens":     10,
			"completion_tokens": 10,
			"total_tokens":      20,
		},
	})
}
