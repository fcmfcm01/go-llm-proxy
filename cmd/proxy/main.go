package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/fcmfcm01/go-llm-proxy/internal/config"
	"github.com/fcmfcm01/go-llm-proxy/internal/logging"
	"github.com/fcmfcm01/go-llm-proxy/internal/server"
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
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger := logging.NewSimpleLogger("info")
	logger.Info("Starting LLM Proxy", "version", "1.0.0", "port", cfg.Server.Port)

	// Create HTTP server
	srv := server.NewServer(&server.ServerConfig{
		Port:         cfg.Server.Port,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}, logger.Logger)

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
	srv.Stop()

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
