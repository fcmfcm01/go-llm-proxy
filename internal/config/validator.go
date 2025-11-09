package config

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Validate validates the configuration
func (c *Config) Validate() error {
	if err := c.Server.validate(); err != nil {
		return fmt.Errorf("server validation failed: %w", err)
	}

	if err := c.Proxy.validate(); err != nil {
		return fmt.Errorf("proxy validation failed: %w", err)
	}

	if err := c.Security.validate(); err != nil {
		return fmt.Errorf("security validation failed: %w", err)
	}

	if err := c.Logging.validate(); err != nil {
		return fmt.Errorf("logging validation failed: %w", err)
	}

	if err := c.Monitoring.validate(); err != nil {
		return fmt.Errorf("monitoring validation failed: %w", err)
	}

	return nil
}

// validate validates server configuration
func (s *ServerConfig) validate() error {
	// Validate ports
	if err := validatePort(s.Port); err != nil {
		return fmt.Errorf("invalid port: %w", err)
	}

	if err := validatePort(s.TLSPort); err != nil {
		return fmt.Errorf("invalid TLS port: %w", err)
	}

	// Check if TLS is enabled, ensure cert and key paths are set
	if s.EnableTLS {
		if s.TLSCertPath == "" {
			return fmt.Errorf("TLS certificate path is required when TLS is enabled")
		}
		if s.TLSKeyPath == "" {
			return fmt.Errorf("TLS key path is required when TLS is enabled")
		}
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

	// Validate CORS origins if CORS is enabled
	if s.EnableCORS {
		for _, origin := range s.CORSOrigins {
			if err := validateOrigin(origin); err != nil {
				return fmt.Errorf("invalid CORS origin %s: %w", origin, err)
			}
		}
	}

	// Validate rate limiting
	if err := s.RateLimit.validate(); err != nil {
		return fmt.Errorf("rate limit validation failed: %w", err)
	}

	return nil
}

// validate validates proxy configuration
func (p *ProxyConfig) validate() error {
	// Validate timeouts
	if p.ReadTimeout < 0 {
		return fmt.Errorf("read timeout must be non-negative")
	}

	if p.WriteTimeout < 0 {
		return fmt.Errorf("write timeout must be non-negative")
	}

	if p.IdleTimeout < 0 {
		return fmt.Errorf("idle timeout must be non-negative")
	}

	// Validate retries
	if p.MaxRetries < 0 {
		return fmt.Errorf("max retries must be non-negative")
	}

	// Validate providers
	for i, provider := range p.Providers {
		if err := provider.validate(); err != nil {
			return fmt.Errorf("provider %d validation failed: %w", i, err)
		}
	}

	return nil
}

// validate validates provider configuration
func (p *ProviderConfig) validate() error {
	// Check required fields
	if p.ID == "" {
		return fmt.Errorf("provider ID is required")
	}

	if p.Name == "" {
		return fmt.Errorf("provider name is required")
	}

	if p.APIURL == "" {
		return fmt.Errorf("provider API URL is required")
	}

	// Validate URL
	if _, err := url.Parse(p.APIURL); err != nil {
		return fmt.Errorf("invalid API URL: %w", err)
	}

	// Validate priority
	if p.Priority < 1 || p.Priority > 100 {
		return fmt.Errorf("priority must be between 1 and 100")
	}

	// Validate timeouts
	if p.Timeout < 0 {
		return fmt.Errorf("timeout must be non-negative")
	}

	if p.MaxRetries < 0 {
		return fmt.Errorf("max retries must be non-negative")
	}

	return nil
}

// validate validates security configuration
func (s *SecurityConfig) validate() error {
	// Validate session
	if err := s.Session.validate(); err != nil {
		return fmt.Errorf("session validation failed: %w", err)
	}

	// Validate password
	if err := s.Password.validate(); err != nil {
		return fmt.Errorf("password validation failed: %w", err)
	}

	// Validate CSRF
	if err := s.CSRF.validate(); err != nil {
		return fmt.Errorf("CSRF validation failed: %w", err)
	}

	return nil
}

// validate validates session configuration
func (s *SessionConfig) validate() error {
	if s.Name == "" {
		return fmt.Errorf("session name is required")
	}

	if s.Secret == "" {
		return fmt.Errorf("session secret is required")
	}

	if s.MaxAge < 0 {
		return fmt.Errorf("session max age must be non-negative")
	}

	return nil
}

// validate validates password configuration
func (p *PasswordConfig) validate() error {
	// Validate algorithm
	validAlgorithms := []string{"argon2id", "bcrypt"}
	if !contains(validAlgorithms, p.Algorithm) {
		return fmt.Errorf("invalid password algorithm %s, must be one of: %v", p.Algorithm, validAlgorithms)
	}

	// Validate Argon2 parameters
	if p.Algorithm == "argon2id" {
		if p.Memory < 8192 {
			return fmt.Errorf("argon2id memory must be at least 8 KB")
		}
		if p.Iterations < 1 {
			return fmt.Errorf("argon2id iterations must be at least 1")
		}
		if p.Parallelism < 1 {
			return fmt.Errorf("argon2id parallelism must be at least 1")
		}
	}

	// Validate bcrypt parameters
	if p.Algorithm == "bcrypt" {
		if p.BCryptCost < 4 || p.BCryptCost > 31 {
			return fmt.Errorf("bcrypt cost must be between 4 and 31")
		}
	}

	return nil
}

// validate validates CSRF configuration
func (c *CSRFConfig) validate() error {
	if c.CookieName == "" {
		return fmt.Errorf("CSRF cookie name is required")
	}

	if c.HeaderName == "" {
		return fmt.Errorf("CSRF header name is required")
	}

	if c.TokenLength < 16 {
		return fmt.Errorf("CSRF token length must be at least 16")
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

	// Validate file settings if output is file
	if l.Output == "file" {
		if l.FilePath == "" {
			return fmt.Errorf("log file path is required when output is file")
		}
	}

	return nil
}

// validate validates monitoring configuration
func (m *MonitoringConfig) validate() error {
	// Validate metrics
	if err := m.Metrics.validate(); err != nil {
		return fmt.Errorf("metrics validation failed: %w", err)
	}

	// Validate health checks
	if err := m.HealthChecks.validate(); err != nil {
		return fmt.Errorf("health checks validation failed: %w", err)
	}

	return nil
}

// validate validates metrics configuration
func (m *MetricsConfig) validate() error {
	if m.Path == "" {
		return fmt.Errorf("metrics path is required")
	}

	if m.Namespace == "" {
		return fmt.Errorf("metrics namespace is required")
	}

	return nil
}

// validate validates health check configuration
func (h *HealthChecksConfig) validate() error {
	if h.BasicPath == "" {
		return fmt.Errorf("basic health check path is required")
	}

	if h.DetailedPath == "" {
		return fmt.Errorf("detailed health check path is required")
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
