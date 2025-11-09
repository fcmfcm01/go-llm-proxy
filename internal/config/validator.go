package config

import (
	"fmt"
	"net/url"
	"strings"
)

// Validate validates the configuration
func (c *Config) Validate() error {
	if err := c.Server.validate(); err != nil {
		return fmt.Errorf("server validation failed: %w", err)
	}

	if err := c.Proxy.validate(); err != nil {
		return fmt.Errorf("proxy validation failed: %w", err)
	}

	if err := c.Auth.validate(); err != nil {
		return fmt.Errorf("auth validation failed: %w", err)
	}

	if err := c.Logging.validate(); err != nil {
		return fmt.Errorf("logging validation failed: %w", err)
	}

	if err := c.Metrics.validate(); err != nil {
		return fmt.Errorf("metrics validation failed: %w", err)
	}

	return nil
}

// validate validates server configuration
func (s *ServerConfig) validate() error {
	// Validate port
	if err := validatePort(s.Port); err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}

	// Validate timeouts
	if s.ReadTimeout < 0 {
		return fmt.Errorf("read timeout must be non-negative")
	}

	if s.WriteTimeout < 0 {
		return fmt.Errorf("write timeout must be non-negative")
	}

	if s.IdleTimeout < 0 {
		return fmt.Errorf("idle timeout must be non-negative")
	}

	return nil
}

// validate validates proxy configuration
func (p *ProxyConfig) validate() error {
	// Validate timeout
	if p.Timeout < 0 {
		return fmt.Errorf("timeout must be non-negative")
	}

	// Validate retries
	if p.MaxRetries < 0 {
		return fmt.Errorf("max retries must be non-negative")
	}

	return nil
}

// validate validates auth configuration
func (a *AuthConfig) validate() error {
	if a.Enabled {
		if a.Username == "" {
			return fmt.Errorf("username is required when auth is enabled")
		}
		if a.Password == "" {
			return fmt.Errorf("password is required when auth is enabled")
		}
	}
	return nil
}

// validate validates provider configuration
func (p *Provider) validate() error {
	// Check required fields
	if p.ID == "" {
		return fmt.Errorf("provider ID is required")
	}

	if p.Name == "" {
		return fmt.Errorf("provider name is required")
	}

	if p.URL == "" {
		return fmt.Errorf("provider URL is required")
	}

	// Validate URL
	if _, err := url.Parse(p.URL); err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	return nil
}

// validate validates logging configuration
func (l *LoggingConfig) validate() error {
	// Validate level
	validLevels := []string{"debug", "info", "warn", "error", "fatal"}
	if !contains(validLevels, l.Level) {
		return fmt.Errorf("invalid log level %s, must be one of: %v", l.Level, validLevels)
	}

	// Validate format
	validFormats := []string{"json", "text"}
	if !contains(validFormats, l.Format) {
		return fmt.Errorf("invalid log format %s, must be one of: %v", l.Format, validFormats)
	}

	// Validate output
	if l.Output != "stdout" && l.Output != "file" {
		return fmt.Errorf("invalid log output %s, must be 'stdout' or 'file'", l.Output)
	}

	return nil
}

// validate validates metrics configuration
func (m *MetricsConfig) validate() error {
	if m.Path == "" {
		return fmt.Errorf("metrics path is required")
	}

	return nil
}

// validatePort validates a port number
func validatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port %d is out of valid range (1-65535)", port)
	}
	return nil
}

// validateOrigin validates a CORS origin
func validateOrigin(origin string) error {
	// Allow localhost
	if strings.HasPrefix(origin, "http://localhost:") {
		return nil
	}

	if strings.HasPrefix(origin, "https://localhost:") {
		return nil
	}

	// Allow wildcards for development
	if origin == "*" {
		return nil
	}

	// Validate as URL
	u, err := url.Parse(origin)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if u.Scheme == "" {
		return fmt.Errorf("origin must include scheme (http or https)")
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("origin scheme must be http or https")
	}

	if u.Host == "" {
		return fmt.Errorf("origin must include host")
	}

	return nil
}

// contains checks if a slice contains a value
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}
