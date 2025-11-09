package config

import (
	"fmt"
	"os"
	"time"

	"github.com/example/go-llm-proxy/internal/config"
	"github.com/spf13/viper"
)

// DefaultConfig returns the default configuration
func DefaultConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			Mode:         "debug",
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		Proxy: config.ProxyConfig{
			DefaultBackend: "https://api.openai.com/v1",
			Timeout:        30 * time.Second,
			MaxRetries:     3,
		},
		Auth: config.AuthConfig{
			Enabled:        true,
			Username:       "admin",
			Password:       "admin123",
			SessionTimeout: 24 * time.Hour,
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
		Metrics: config.MetricsConfig{
			Enabled: true,
			Path:    "/metrics",
		},
		Providers: []config.Provider{},
		ModelMapping: map[string]string{
			"claude-3-opus-20240229":     "gpt-4",
			"claude-3-sonnet-20240229":   "gpt-4-turbo",
			"claude-3-haiku-20240307":    "gpt-3.5-turbo",
			"claude-3-5-sonnet-20241022": "gpt-4o",
		},
	}
}

// LoadConfig loads configuration from file
func LoadConfig(configPath string) (*config.Config, error) {
	if configPath == "" {
		configPath = "config.yaml"
	}

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config file
		defaultConfig := DefaultConfig()
		if err := SaveConfig(defaultConfig, configPath); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		return defaultConfig, nil
	}

	// Load from file
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// Enable environment variable override
	viper.AutomaticEnv()

	// Read config
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config config.Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// SaveConfig saves configuration to file
func SaveConfig(config *config.Config, configPath string) error {
	if configPath == "" {
		configPath = "config.yaml"
	}

	viper.Set("server", config.Server)
	viper.Set("proxy", config.Proxy)
	viper.Set("auth", config.Auth)
	viper.Set("logging", config.Logging)
	viper.Set("metrics", config.Metrics)
	viper.Set("providers", config.Providers)
	viper.Set("model_mapping", config.ModelMapping)

	return viper.WriteConfigAs(configPath)
}
