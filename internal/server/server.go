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
}

// Metrics holds server metrics
type Metrics struct {
	TotalRequests   *Counter
	ActiveRequests  *Gauge
	RequestDuration *Histogram
	Errors          *Counter
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

	// Add logging middleware
	s.router.Use(s.loggingMiddleware())

	// Health check endpoints
	s.setupHealthRoutes()

	// Metrics endpoint
	s.setupMetricsRoutes()

	// API routes
	s.setupAPIRoutes()

	// Admin routes
	s.setupAdminRoutes()
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
