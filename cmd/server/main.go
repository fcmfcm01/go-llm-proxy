package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fcmfcm01/go-llm-proxy/internal/config"
	"github.com/fcmfcm01/go-llm-proxy/internal/server"
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

	// Create server
	srv := server.NewServer(&server.ServerConfig{
		Port:         cfg.Server.Port,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}, logger)

	// Setup routes with proper components
	// Note: In a real implementation, this would wire up all the handlers properly
	srv.SetupRoutes()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown
	if err := srv.Stop(); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}
