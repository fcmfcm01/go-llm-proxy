package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/example/go-llm-proxy/internal/auth"
	"github.com/example/go-llm-proxy/internal/config"
	"github.com/example/go-llm-proxy/internal/converter"
	"github.com/example/go-llm-proxy/internal/server"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	configPath string
	port       int
	host       string
	mode       string
	logLevel   string
)

var rootCmd = &cobra.Command{
	Use:   "go-llm-proxy",
	Short: "High-performance LLM proxy service",
	Long:  `Go LLM Proxy - A high-performance proxy for OpenAI-compatible APIs with format conversion`,
	Run: func(cmd *cobra.Command, args []string) {
		runServer()
	},
}

func init() {
	rootCmd.Flags().StringVarP(&configPath, "config", "c", "config.yaml", "Path to config file")
	rootCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")
	rootCmd.Flags().StringVarP(&host, "host", "H", "0.0.0.0", "Host to listen on")
	rootCmd.Flags().StringVarP(&mode, "mode", "m", "debug", "Server mode (debug/release)")
	rootCmd.Flags().StringVar(&logLevel, "log-level", "info", "Log level (debug/info/warn/error)")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runServer() {
	// Setup logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	logger.Info("Starting Go LLM Proxy...")

	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		logger.Fatalf("Failed to load config: %v", err)
	}

	// Override config with flags
	if port != 0 {
		cfg.Server.Port = port
	}
	if host != "" {
		cfg.Server.Host = host
	}
	if mode != "" {
		cfg.Server.Mode = mode
	}

	// Setup config manager
	configManager := config.NewConfigManager("data/config.json", logger)
	if err := configManager.Load(); err != nil {
		logger.Warn("Failed to load config manager: ", err)
	}

	// Setup converter
	conv := converter.NewConverter(cfg.ModelMapping, logger)

	// Setup auth
	authManager := auth.NewSessionManager(logger)

	// Setup router
	router := server.SetupRouter(configManager, conv, authManager, logger)

	// Create HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		logger.Infof("Server listening on %s:%d", cfg.Server.Host, cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server failed to start: %v", err)
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

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}
