package config

import "time"

type Config struct {
	Server       ServerConfig      `mapstructure:"server"       json:"server"`
	Proxy        ProxyConfig       `mapstructure:"proxy"        json:"proxy"`
	Auth         AuthConfig        `mapstructure:"auth"         json:"auth"`
	Logging      LoggingConfig     `mapstructure:"logging"      json:"logging"`
	Metrics      MetricsConfig     `mapstructure:"metrics"      json:"metrics"`
	Providers    []Provider        `mapstructure:"providers"    json:"providers"`
	ModelMapping map[string]string `mapstructure:"model_mapping" json:"model_mapping"`
}

type ServerConfig struct {
	Host         string        `mapstructure:"host"          json:"host"`
	Port         int           `mapstructure:"port"          json:"port"`
	Mode         string        `mapstructure:"mode"          json:"mode"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"  json:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout" json:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"  json:"idle_timeout"`
}

type ProxyConfig struct {
	DefaultBackend string        `mapstructure:"default_backend" json:"default_backend"`
	Timeout        time.Duration `mapstructure:"timeout"          json:"timeout"`
	MaxRetries     int           `mapstructure:"max_retries"      json:"max_retries"`
}

type AuthConfig struct {
	Enabled        bool          `mapstructure:"enabled"        json:"enabled"`
	Username       string        `mapstructure:"username"       json:"username"`
	Password       string        `mapstructure:"password"       json:"password"`
	SessionTimeout time.Duration `mapstructure:"session_timeout" json:"session_timeout"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"  json:"level"`
	Format string `mapstructure:"format" json:"format"`
	Output string `mapstructure:"output" json:"output"`
}

type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled" json:"enabled"`
	Path    string `mapstructure:"path"    json:"path"`
}

type Provider struct {
	ID         string `mapstructure:"id"`
	Name       string `mapstructure:"name"`
	URL        string `mapstructure:"url"`
	APIKey     string `mapstructure:"api_key"`
	Enabled    bool   `mapstructure:"enabled"`
	CreatedAt  int64  `mapstructure:"created_at"`
	UpdatedAt  int64  `mapstructure:"updated_at"`
}

// ProviderStatus represents the health status of a provider
type ProviderStatus struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Healthy         bool   `json:"healthy"`
	LastCheck       int64  `json:"last_check"`
	ResponseTime    int64  `json:"response_time_ms,omitempty"`
	ConsecutiveFailures int `json:"consecutive_failures,omitempty"`
}
