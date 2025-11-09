# Configuration Reference

This document describes all configuration options for LLM Proxy.

## Table of Contents

- [Configuration File](#configuration-file)
- [Server Configuration](#server-configuration)
- [Authentication Configuration](#authentication-configuration)
- [Logging Configuration](#logging-configuration)
- [Metrics Configuration](#metrics-configuration)
- [Proxy Configuration](#proxy-configuration)
- [Performance Configuration](#performance-configuration)
- [Provider Configuration](#provider-configuration)
- [Model Mapping Configuration](#model-mapping-configuration)
- [Security Configuration](#security-configuration)
- [Environment Variables](#environment-variables)
- [Configuration Examples](#configuration-examples)

## Configuration File

LLM Proxy uses YAML configuration files. The default location is `/app/config/config.yaml`.

### Loading Configuration

```bash
# Specify config file
./llm-proxy serve --config /path/to/config.yaml

# Validate config
./llm-proxy validate-config /path/to/config.yaml

# Generate sample config
./llm-proxy config --sample > sample-config.yaml
```

### Environment Variable Substitution

Configuration supports environment variable substitution:

```yaml
server:
  secret_key: "${SECRET_KEY}"  # Replaced with environment value

providers:
  - name: "openai"
    api_key: "${OPENAI_API_KEY}"  # Replaced with environment value
```

## Server Configuration

Configure the HTTP server.

```yaml
server:
  port: 8080                      # HTTP port (default: 8080)
  tls_port: 8443                  # HTTPS port (default: 8443)
  host: "0.0.0.0"                 # Host to bind to (default: 0.0.0.0)
  mode: "production"              # Mode: development, staging, production
  read_timeout: 30                # Read timeout in seconds (default: 30)
  write_timeout: 30               # Write timeout in seconds (default: 30)
  idle_timeout: 90                # Idle timeout in seconds (default: 90)
  enable_tls: false               # Enable TLS/HTTPS (default: false)
  tls_cert: "/path/to/cert.pem"   # TLS certificate path
  tls_key: "/path/to/key.pem"     # TLS private key path
  enable_http2: true              # Enable HTTP/2 support (default: true)
  max_header_bytes: 1048576       # Max header size in bytes (default: 1MB)
  trusted_proxies:                # Trusted proxy IPs for X-Forwarded-*
    - "10.0.0.0/8"
    - "172.16.0.0/12"
```

### TLS Configuration

```yaml
server:
  enable_tls: true
  tls_cert: "/etc/ssl/certs/llm-proxy.crt"
  tls_key: "/etc/ssl/private/llm-proxy.key"
  # Or use automatic certificate management
  tls:
    auto: true
    cert_manager: letsencrypt
    email: admin@example.com
    domains:
      - api.example.com
```

## Authentication Configuration

Configure authentication and session management.

```yaml
auth:
  session_timeout: 24h            # Session timeout (default: 24h)
  secret_key: "your-secret-key"   # Secret key for session signing (required)
  cookie_name: "session"          # Cookie name (default: session)
  cookie_secure: true             # Secure cookie flag (default: true in prod)
  cookie_http_only: true          # HTTP only flag (default: true)
  cookie_same_site: "strict"      # SameSite policy: strict, lax, none
  session_store: "filesystem"     # Session store: filesystem, redis, memory
  redis_url: "redis://localhost:6379"  # Redis URL (if using redis store)
  max_sessions: 1000              # Max concurrent sessions (default: 1000)
  login_rate_limit: 5/minute      # Login rate limit
```

### Session Store Options

#### Filesystem Store (default)

```yaml
auth:
  session_store: "filesystem"
  session_path: "/app/data/sessions"
```

#### Redis Store

```yaml
auth:
  session_store: "redis"
  redis_url: "redis://:password@redis:6379/0"
  redis_key_prefix: "llm-proxy:session:"
```

#### Memory Store (development only)

```yaml
auth:
  session_store: "memory"
```

## Logging Configuration

Configure logging behavior.

```yaml
logging:
  level: "info"                   # Log level: debug, info, warn, error
  format: "json"                  # Format: json, text
  file: "/var/log/llm-proxy.log"  # Log file path (optional)
  max_size: 100                   # Max file size in MB (default: 100)
  max_backups: 10                 # Max backup files (default: 10)
  max_age: 28                     # Max age in days (default: 28)
  compress: true                  # Compress old log files (default: true)
  enable_audit: true              # Enable audit logging (default: true)
  audit_file: "/var/log/llm-proxy-audit.log"
  output: "stdout"                # Output: stdout, stderr, file
  timestamp_format: "2006-01-02T15:04:05"  # Timestamp format
```

### Audit Logging

```yaml
logging:
  enable_audit: true
  audit_events:                   # Events to audit
    - "login"
    - "logout"
    - "provider_create"
    - "provider_update"
    - "provider_delete"
    - "config_change"
  audit_fields:                   # Fields to log
    - "timestamp"
    - "user"
    - "action"
    - "ip_address"
    - "user_agent"
    - "success"
```

## Metrics Configuration

Configure metrics and monitoring.

```yaml
metrics:
  enabled: true                   # Enable metrics collection
  port: 9090                      # Metrics port (default: 9090)
  path: "/metrics"                # Metrics endpoint path
  namespace: "llm_proxy"          # Prometheus namespace
  subsystem: "proxy"              # Prometheus subsystem
  collect_default_metrics: true   # Collect default Go metrics
  collect_process_metrics: true   # Collect process metrics
  collect_gc_metrics: true        # Collect GC metrics
  request_duration_buckets:       # Histogram buckets for request duration
    - 0.1
    - 0.5
    - 1
    - 2
    - 5
  request_size_buckets:           # Histogram buckets for request size
    - 100
    - 1000
    - 10000
    - 100000
```

## Proxy Configuration

Configure proxy behavior.

```yaml
proxy:
  timeout: 30                     # Proxy timeout in seconds (default: 30)
  max_retries: 3                  # Max retry attempts (default: 3)
  request_timeout: 60             # Request timeout in seconds (default: 60)
  idle_timeout: 90                # Idle timeout in seconds (default: 90)
  buffer_size: 4096               # Buffer size for streaming (default: 4096)
  max_request_size: 10485760      # Max request size in bytes (default: 10MB)
  enable_streaming: true          # Enable streaming responses (default: true)
  enable_request_id: true         # Enable request ID tracking (default: true)
  default_provider:               # Default provider fallback
    name: "openai"
  health_check_interval: 30s      # Provider health check interval
  health_check_timeout: 5s        # Health check timeout
```

## Performance Configuration

Configure performance optimizations.

```yaml
performance:
  connection_pool_size: 100       # HTTP connection pool size
  max_idle_connections: 100       # Max idle connections
  idle_connection_timeout: 90s    # Idle connection timeout
  connection_timeout: 30s         # Connection timeout
  keep_alive: 30s                 # Keep-alive duration
  max_connections_per_host: 100   # Max connections per host
  disable_compression: false      # Disable compression
  enable_compression: true        # Enable gzip compression
  compression_level: 6            # Compression level (1-9, default: 6)
  min_compress_size: 1024         # Min size to compress (default: 1KB)
  cache_enabled: true             # Enable response caching
  cache_ttl: 300                  # Cache TTL in seconds (default: 300)
  cache_size: 1000                # Max cache entries (default: 1000)
  cache_cleanup_interval: 60s     # Cache cleanup interval
  enable_object_pooling: true     # Enable object pooling
  object_pool_size: 100           # Object pool size
```

### Caching Configuration

```yaml
performance:
  cache:
    enabled: true
    ttl: 300                      # Default TTL in seconds
    max_size: 1000                # Max cache entries
    max_memory: 100MB             # Max memory usage
    cleanup_interval: 60s         # Cleanup interval
    cache:
      GET:
        enabled: true
        ttl: 600
      POST:
        enabled: false            # Don't cache POST by default
    exclude_paths:                # Paths to exclude from cache
      - "/admin"
      - "/healthz"
      - "/metrics"
```

## Provider Configuration

Configure LLM providers.

```yaml
providers:
  - name: "openai"                # Provider name (required, unique)
    type: "openai"                # Provider type: openai, anthropic, custom
    api_key: "sk-..."             # API key (required)
    base_url: "https://api.openai.com/v1"  # Base URL (required)
    priority: 1                   # Priority (lower = higher priority)
    enabled: true                 # Enabled status
    max_requests: 1000            # Max requests per minute
    max_retries: 3                # Max retry attempts
    timeout: 30                   # Request timeout in seconds
    health_check:                 # Health check configuration
      enabled: true
      interval: 30s
      timeout: 5s
      endpoint: "/models"         # Health check endpoint
    headers:                      # Custom headers
      "X-Custom-Header": "value"
    rate_limit:                   # Rate limiting
      requests_per_minute: 1000
      burst: 100
    weight: 1.0                   # Load balancing weight
    max_connections: 100          # Max concurrent connections
    # Provider-specific options
    models:                       # Available models
      - "gpt-3.5-turbo"
      - "gpt-3.5-turbo-16k"
      - "gpt-4"
      - "gpt-4-32k"
```

### Provider Types

#### OpenAI

```yaml
  - name: "openai"
    type: "openai"
    api_key: "${OPENAI_API_KEY}"
    base_url: "https://api.openai.com/v1"
    priority: 1
    enabled: true
```

#### Anthropic

```yaml
  - name: "anthropic"
    type: "anthropic"
    api_key: "${ANTHROPIC_API_KEY}"
    base_url: "https://api.anthropic.com"
    priority: 2
    enabled: true
```

#### Custom Provider

```yaml
  - name: "custom"
    type: "custom"
    api_key: "${CUSTOM_API_KEY}"
    base_url: "https://api.custom.com/v1"
    priority: 3
    enabled: true
    format:                       # Custom format conversion
      request:
        path: "/chat/completions"
        method: "POST"
      response:
        path: "choices.0.message.content"
        type: "text"
    models:
      - "custom-model-1"
      - "custom-model-2"
```

## Model Mapping Configuration

Configure model name mappings.

```yaml
model_mappings:
  - id: "map-1"                   # Mapping ID (optional, auto-generated)
    provider_name: "openai"       # Provider name (required)
    local_name: "gpt-3.5-turbo"   # Local model name (required)
    remote_name: "gpt-3.5-turbo"  # Remote model name (required)
    enabled: true                 # Enabled status (default: true)
    metadata:                     # Additional metadata
      max_tokens: 4096
      pricing: "0.0015/1K tokens"
```

### Automatic Model Mapping

```yaml
model_mappings:
  auto_create: true               # Auto-create mappings for new models
  default_provider: "openai"      # Default provider for unmapped models
  mappings:
    "gpt-3.5-turbo":              # Local name
      provider: "openai"
      remote_name: "gpt-3.5-turbo"  # Remote name
    "claude-3-sonnet":
      provider: "anthropic"
      remote_name: "claude-3-sonnet-20240229"
```

## Security Configuration

Configure security settings.

```yaml
security:
  rate_limit:                     # Global rate limiting
    enabled: true
    requests_per_minute: 1000
    burst: 100
  enable_cors: false              # Enable CORS (default: false)
  cors_allowed_origins:           # Allowed CORS origins
    - "https://example.com"
    - "https://app.example.com"
  cors_allowed_methods:           # Allowed CORS methods
    - "GET"
    - "POST"
    - "OPTIONS"
  cors_allowed_headers:           # Allowed CORS headers
    - "Content-Type"
    - "Authorization"
  cors_exposed_headers:           # Exposed CORS headers
    - "X-RateLimit-Limit"
  cors_allow_credentials: true    # Allow credentials
  enable_csrf: true               # Enable CSRF protection
  csrf_secret: "${CSRF_SECRET}"   # CSRF secret
  security_headers:               # Security headers
    content_security_policy: "default-src 'self'"
    x_frame_options: "DENY"
    x_content_type_options: "nosniff"
    x_xss_protection: "1; mode=block"
    strict_transport_security: "max-age=31536000; includeSubDomains"
  input_validation:               # Input validation
    max_length: 1000000           # Max request length
    sanitize_html: true           # Sanitize HTML input
    block_sql_injection: true     # Block SQL injection patterns
    block_xss: true               # Block XSS patterns
  api_key_encryption:             # API key encryption
    enabled: true
    encryption_key: "${ENCRYPTION_KEY}"  # Encryption key
    algorithm: "AES-256-GCM"      # Encryption algorithm
```

### Rate Limiting

```yaml
security:
  rate_limit:
    enabled: true
    storage: "memory"             # Storage: memory, redis
    redis_url: "redis://localhost:6379"  # Redis storage
    strategies:
      - name: "ip-based"          # IP-based rate limiting
        key: "${remote_ip}"
        limit: 1000/minute
        burst: 100
      - name: "user-based"        # User-based rate limiting (requires auth)
        key: "${user_id}"
        limit: 500/minute
        burst: 50
```

## Environment Variables

All configuration options can be set via environment variables.

### Naming Convention

Convert configuration key to uppercase and use `LLM_PROXY_` prefix:

- `server.port` → `LLM_PROXY_SERVER_PORT`
- `auth.session_timeout` → `LLM_PROXY_AUTH_SESSION_TIMEOUT`
- `providers.0.api_key` → `LLM_PROXY_PROVIDER_0__API_KEY`

### Common Environment Variables

```bash
# Server
export LLM_PROXY_SERVER_PORT=8080
export LLM_PROXY_SERVER_TLS_PORT=8443
export LLM_PROXY_SERVER_ENABLE_TLS=true

# Auth
export LLM_PROXY_AUTH_SESSION_SECRET=your-secret-key
export LLM_PROXY_AUTH_SESSION_TIMEOUT=24h

# Logging
export LLM_PROXY_LOGGING_LEVEL=info
export LLM_PROXY_LOGGING_FORMAT=json

# Metrics
export LLM_PROXY_METRICS_ENABLED=true
export LLM_PROXY_METRICS_PORT=9090

# Performance
export LLM_PROXY_PERFORMANCE_CACHE_TTL=300
export LLM_PROXY_PERFORMANCE_ENABLE_COMPRESSION=true

# Security
export LLM_PROXY_SECURITY_RATE_LIMIT=1000
export LLM_PROXY_SECURITY_CORS_ALLOWED_ORIGINS="https://example.com"

# Providers (JSON array)
export LLM_PROXY_PROVIDERS='[
  {
    "name": "openai",
    "type": "openai",
    "api_key": "sk-...",
    "base_url": "https://api.openai.com/v1",
    "priority": 1,
    "enabled": true
  }
]'
```

### Using .env Files

Create a `.env` file:

```bash
# Server
LLM_PROXY_SERVER_PORT=8080
LLM_PROXY_SERVER_ENABLE_TLS=true

# Auth
LLM_PROXY_AUTH_SESSION_SECRET=your-secret-key

# Providers
LLM_PROXY_PROVIDER_OPENAI_API_KEY=sk-...
LLM_PROXY_PROVIDER_ANTHROPIC_API_KEY=sk-ant-...
```

Load in configuration:

```yaml
server:
  port: "${LLM_PROXY_SERVER_PORT}"  # Loads from .env
  secret_key: "${LLM_PROXY_AUTH_SESSION_SECRET}"  # Loads from .env
```

## Configuration Examples

### Development Configuration

```yaml
server:
  port: 8080
  mode: "development"
  enable_tls: false

auth:
  session_timeout: 12h
  secret_key: "dev-secret-change-me"

logging:
  level: "debug"
  format: "text"
  enable_audit: false

metrics:
  enabled: true

proxy:
  timeout: 30
  max_retries: 3

performance:
  connection_pool_size: 50
  cache_enabled: false
  enable_compression: false

providers:
  - name: "openai-dev"
    type: "openai"
    api_key: "${OPENAI_API_KEY}"
    base_url: "https://api.openai.com/v1"
    priority: 1
    enabled: true
```

### Production Configuration

```yaml
server:
  port: 8080
  tls_port: 8443
  host: "0.0.0.0"
  mode: "production"
  read_timeout: 30
  write_timeout: 30
  enable_tls: true
  tls_cert: "/etc/ssl/certs/llm-proxy.crt"
  tls_key: "/etc/ssl/private/llm-proxy.key"

auth:
  session_timeout: 24h
  secret_key: "${LLM_PROXY_SECRET_KEY}"
  cookie_secure: true
  cookie_http_only: true
  session_store: "redis"
  redis_url: "redis://:password@redis:6379/0"

logging:
  level: "info"
  format: "json"
  file: "/var/log/llm-proxy/app.log"
  max_size: 100
  max_backups: 10
  enable_audit: true

metrics:
  enabled: true
  port: 9090

proxy:
  timeout: 30
  max_retries: 3
  request_timeout: 60
  health_check_interval: 30s

performance:
  connection_pool_size: 100
  max_idle_connections: 100
  keep_alive: 30s
  enable_compression: true
  compression_level: 6
  cache_enabled: true
  cache_ttl: 300
  cache_size: 1000

security:
  rate_limit:
    enabled: true
    requests_per_minute: 1000
    storage: "redis"
  enable_csrf: true
  input_validation:
    max_length: 1000000
    sanitize_html: true
  api_key_encryption:
    enabled: true
    encryption_key: "${ENCRYPTION_KEY}"

providers:
  - name: "openai-primary"
    type: "openai"
    api_key: "${OPENAI_API_KEY}"
    base_url: "https://api.openai.com/v1"
    priority: 1
    enabled: true
    max_requests: 1000
    timeout: 30
    health_check:
      enabled: true
      interval: 30s

  - name: "anthropic-backup"
    type: "anthropic"
    api_key: "${ANTHROPIC_API_KEY}"
    base_url: "https://api.anthropic.com"
    priority: 2
    enabled: true
    max_requests: 500
    timeout: 30
    health_check:
      enabled: true
      interval: 30s
```

### High-Availability Configuration

```yaml
server:
  port: 8080
  enable_tls: true
  enable_http2: true
  read_timeout: 30
  write_timeout: 30

auth:
  session_timeout: 24h
  secret_key: "${LLM_PROXY_SECRET_KEY}"
  session_store: "redis"
  redis_url: "redis://redis-cluster:6379/0"
  max_sessions: 10000

logging:
  level: "info"
  format: "json"
  output: "stdout"

metrics:
  enabled: true
  port: 9090

proxy:
  timeout: 30
  max_retries: 3
  health_check_interval: 15s

performance:
  connection_pool_size: 200
  max_idle_connections: 200
  enable_compression: true
  cache_enabled: true
  cache_ttl: 300
  cache_size: 5000
  enable_object_pooling: true
  object_pool_size: 200

security:
  rate_limit:
    enabled: true
    requests_per_minute: 5000
    storage: "redis"

providers:
  - name: "openai-us-east"
    type: "openai"
    api_key: "${OPENAI_API_KEY}"
    base_url: "https://api.openai.com/v1"
    priority: 1
    enabled: true
    max_requests: 2000
    max_connections: 200

  - name: "openai-eu-west"
    type: "openai"
    api_key: "${OPENAI_API_KEY_EU}"
    base_url: "https://api.openai.com/v1"
    priority: 2
    enabled: true
    max_requests: 2000
    max_connections: 200

  - name: "anthropic"
    type: "anthropic"
    api_key: "${ANTHROPIC_API_KEY}"
    base_url: "https://api.anthropic.com"
    priority: 3
    enabled: true
    max_requests: 1000
    max_connections: 100
```

### Minimal Configuration

```yaml
server:
  port: 8080

auth:
  secret_key: "your-secret-key"

providers:
  - name: "openai"
    type: "openai"
    api_key: "sk-..."
    base_url: "https://api.openai.com/v1"
    priority: 1
    enabled: true
```

## Configuration Validation

Validate configuration before deployment:

```bash
# Validate configuration file
./llm-proxy validate-config config/production.yaml

# Check configuration
./llm-proxy config --check --config config/production.yaml

# Show effective configuration
./llm-proxy config --dump --config config/production.yaml

# Test provider connections
./llm-proxy test-providers --config config/production.yaml
```

## Configuration Hot Reload

Configuration can be reloaded without restarting:

```bash
# Send SIGHUP signal
kill -HUP $(pgrep llm-proxy)

# Or use reload endpoint
curl -X POST http://localhost:8080/admin/api/v1/reload
```

## Configuration Best Practices

1. **Use environment variables** for secrets and sensitive data
2. **Enable TLS** in production
3. **Set strong session secrets** (minimum 32 characters)
4. **Configure rate limiting** to prevent abuse
5. **Enable audit logging** for compliance
6. **Use Redis** for session storage in production
7. **Set appropriate timeouts** based on provider SLA
8. **Enable caching** to improve performance
9. **Configure monitoring** (metrics, health checks)
10. **Test configuration** before deploying to production

## Support

- **Documentation**: [Full docs](https://github.com/fcmfcm01/go-llm-proxy)
- **Issues**: [GitHub Issues](https://github.com/fcmfcm01/go-llm-proxy/issues)
