package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"go-llm-proxy/internal/config"
	"go-llm-proxy/internal/logging"
	"go-llm-proxy/internal/metrics"
	"go-llm-proxy/internal/server"
)

// @title LLM Proxy API
// @version 1.0
// @description OpenAI-compatible API gateway for LLM providers
// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey ApiKeyAuth
// @type apiKey
// @in header
// @name Authorization

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger := logging.NewLogger(cfg.Logging)
	logger.Info("Starting LLM Proxy", "version", "1.0.0", "port", cfg.Server.Port)

	// Initialize metrics
	metricsCollector := metrics.NewMetrics(cfg.Metrics)
	if err := metricsCollector.Start(); err != nil {
		logger.Warn("Failed to start metrics collector", "error", err)
	}
	defer metricsCollector.Stop()

	// Create HTTP server
	srv, err := server.NewServer(cfg, logger, metricsCollector)
	if err != nil {
		logger.Error("Failed to create server", "error", err)
		os.Exit(1)
	}

	// Start server in goroutine
	go func() {
		logger.Info("Server starting", "port", cfg.Server.Port)
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Stop(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("Server exited")
}

// Health check endpoint
// @Summary Health check
// @Description Get health status
// @Tags health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /healthz [get]
func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, map[string]string{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
