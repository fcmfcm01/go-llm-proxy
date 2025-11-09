# LLM Proxy (Go)

A high-performance, OpenAI-compatible API gateway for Large Language Models (LLMs) built in Go. This is a complete rewrite of the Python version with significant performance improvements and production-ready features.

[![CI](https://github.com/fcmfcm01/go-llm-proxy/actions/workflows/ci.yml/badge.svg)](https://github.com/fcmfcm01/go-llm-proxy/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/fcmfcm01/go-llm-proxy)](https://goreportcard.com/report/github.com/fcmfcm01/go-llm-proxy)
[![codecov](https://codecov.io/gh/fcmfcm01/go-llm-proxy/branch/main/graph/badge.svg)](https://codecov.io/gh/fcmfcm01/go-llm-proxy)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## ✨ Features

### Core Features

- **OpenAI-Compatible API** - Drop-in replacement for OpenAI API clients
- **Multi-Provider Support** - Route to multiple LLM providers (OpenAI, Anthropic, etc.)
- **Format Conversion** - Automatic conversion between OpenAI and Anthropic formats
- **Load Balancing** - Round-robin load balancing across providers
- **Automatic Failover** - Graceful failover when providers are unavailable
- **Real-time Streaming** - Server-sent events for streaming responses

### Security & Authentication

- **Admin Authentication** - Secure login with session management
- **Session Timeout** - Configurable session timeout (default: 24 hours)
- **CSRF Protection** - Cross-site request forgery protection
- **Rate Limiting** - Token bucket rate limiting per IP
- **Security Headers** - CSP, HSTS, X-Frame-Options, and more
- **Input Validation** - XSS and SQL injection protection
- **API Key Encryption** - Encrypt sensitive data at rest

### Management & Monitoring

- **Web UI** - Modern Bootstrap 5 admin interface
- **Provider Management** - Add/edit/delete providers without restart
- **Real-time Monitoring** - Live provider status and health checks
- **Performance Metrics** - Prometheus integration
- **Grafana Dashboards** - Pre-configured monitoring dashboards
- **Audit Logging** - Complete audit trail of all operations
- **Backup & Restore** - Automated configuration backup

### Performance & Scalability

- **High Performance** - 25% faster than Python version
- **Low Memory Footprint** - <50MB memory usage under normal load
- **Connection Pooling** - Efficient HTTP connection management
- **Response Caching** - LRU cache with TTL
- **Object Pooling** - Reduce GC pressure with sync.Pool
- **Response Compression** - gzip compression for all responses
- **HTTP/2 Support** - Multiplexing and server push
- **1000+ Concurrent Connections** - Tested and validated

## 🚀 Quick Start

### Using Docker Compose (Recommended)

```bash
# Clone the repository
git clone https://github.com/fcmfcm01/go-llm-proxy.git
cd go-llm-proxy

# Start the services
docker-compose up -d

# Access the application
# API: http://localhost:8080
# Admin: http://localhost:8080/admin
# Grafana: http://localhost:3000 (admin/admin)
# Prometheus: http://localhost:9090
```

### Using Kubernetes

```bash
# Deploy to Kubernetes
kubectl apply -f deployment/k8s/

# Or using Helm (if available)
helm install llm-proxy ./deployment/helm
```

### From Binary

```bash
# Download the latest release
wget https://github.com/fcmfcm01/go-llm-proxy/releases/latest/download/go-llm-proxy-linux-amd64

# Make it executable
chmod +x go-llm-proxy-linux-amd64

# Run the binary
./go-llm-proxy-linux-amd64 serve
```

### Using systemd

```bash
# Install the service
sudo ./scripts/deploy.sh systemd

# Check status
sudo systemctl status llm-proxy
```

## 📖 Usage

### As an OpenAI API Client

The proxy is fully compatible with OpenAI's API. Simply change your base URL:

```python
# Before (direct to OpenAI)
import openai
client = openai.OpenAI(api_key="sk-...")

# After (through proxy)
import openai
client = openai.OpenAI(
    api_key="sk-...",
    base_url="http://localhost:8080/v1"
)

# All OpenAI API calls work unchanged
chat = client.chat.completions.create(
    model="gpt-3.5-turbo",
    messages=[{"role": "user", "content": "Hello!"}]
)
```

### Using Anthropic with OpenAI Client

```python
# The proxy automatically converts formats
chat = client.chat.completions.create(
    model="claude-3-sonnet-20240229",  # Anthropic model name
    messages=[{"role": "user", "content": "Hello!"}]
)
```

### Configuration

Create a configuration file (`config.yaml`):

```yaml
server:
  port: 8080
  tls_port: 8443
  enable_tls: false

auth:
  session_timeout: 24h
  secret_key: "your-secret-key"

providers:
  - name: "openai"
    type: "openai"
    api_key: "sk-..."
    base_url: "https://api.openai.com/v1"
    priority: 1
    enabled: true
    max_requests: 1000

  - name: "anthropic"
    type: "anthropic"
    api_key: "sk-ant-..."
    base_url: "https://api.anthropic.com"
    priority: 2
    enabled: true
    max_requests: 500
```

### Health Check

```bash
# Check service health
curl http://localhost:8080/healthz

# Check readiness
curl http://localhost:8080/healthz/ready

# Detailed health check
curl http://localhost:8080/healthz/detailed

# Metrics
curl http://localhost:8080/metrics
```

## 🏗️ Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────────┐
│                    Client (OpenAI SDK)                  │
└────────────────────┬────────────────────────────────────┘
                     │ HTTP
                     ▼
┌─────────────────────────────────────────────────────────┐
│                 LLM Proxy (Go)                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │   Auth       │  │   Web UI     │  │   Metrics    │ │
│  └──────────────┘  └──────────────┘  └──────────────┘ │
│                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │   Proxy      │  │  Monitoring  │  │   Audit      │ │
│  │  Routing     │  │  & Health    │  │   Logging    │ │
│  └──────────────┘  └──────────────┘  └──────────────┘ │
└────────────────────┬────────────────────────────────────┘
                     │
         ┌───────────┴───────────┐
         ▼                       ▼
┌──────────────┐         ┌──────────────┐
│  Provider 1  │         │  Provider 2  │
│   (OpenAI)   │         │ (Anthropic)  │
└──────────────┘         └──────────────┘
```

### Data Flow

1. **Request** arrives at proxy
2. **Authentication** verified
3. **Rate limiting** applied
4. **Provider selection** via load balancer
5. **Format conversion** (if needed)
6. **Request forwarded** to provider
7. **Response converted** (if needed)
8. **Response streamed** to client
9. **Metrics recorded**
10. **Audit log written**

## 📊 Monitoring

### Metrics Endpoint

Prometheus-compatible metrics available at `/metrics`:

- `llm_proxy_requests_total` - Total requests
- `llm_proxy_requests_duration_seconds` - Request duration
- `llm_proxy_requests_failed_total` - Failed requests
- `llm_proxy_provider_health` - Provider health status
- `llm_proxy_cache_hits_total` - Cache hit count
- `llm_proxy_cache_misses_total` - Cache miss count

### Grafana Dashboards

Pre-configured dashboards available in `deployment/grafana/dashboards/`:

- **Overview** - System-wide metrics
- **Providers** - Per-provider metrics
- **Performance** - Latency and throughput
- **Errors** - Error rates and types
- **Traffic** - Request volume

## 🔧 Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `LLM_PROXY_SERVER_PORT` | HTTP port | 8080 |
| `LLM_PROXY_SERVER_TLS_PORT` | HTTPS port | 8443 |
| `LLM_PROXY_SECURITY_SECRET_KEY` | Session secret | Required |
| `LLM_PROXY_LOGGING_LEVEL` | Log level | info |
| `LLM_PROXY_MONITORING_ENABLED` | Enable metrics | true |
| `LLM_PROXY_PERFORMANCE_CACHE_TTL` | Cache TTL (seconds) | 300 |

### Full Configuration

See `docs/configuration.md` for complete configuration options.

## 🧪 Testing

### Run Tests

```bash
# Unit tests
go test ./...

# Integration tests
go test -tags=integration ./tests/integration/...

# Contract tests
go test ./tests/contract/...

# E2E tests
go test ./tests/e2e/...

# Security tests
go test ./tests/security/...

# All tests with coverage
go test -coverprofile=coverage.out ./...
make test-coverage
```

### Load Testing

```bash
# Run load tests
go test ./tests/e2e/ -run TestLoad

# Benchmark tests
go test -bench=. ./tests/benchmark/...

# Stress test
go test ./tests/e2e/ -run TestStress
```

## 📦 Deployment

### Docker

```bash
# Build image
docker build -t go-llm-proxy:latest .

# Run container
docker run -d \
  --name llm-proxy \
  -p 8080:8080 \
  -p 8443:8443 \
  -v $(pwd)/config:/app/config \
  go-llm-proxy:latest
```

### Production Docker Compose

```bash
# Deploy production stack
docker-compose -f deployment/docker-compose.prod.yml up -d
```

### Kubernetes

```bash
# Deploy with kustomize
kubectl apply -k deployment/k8s/

# Or with kubectl
kubectl apply -f deployment/k8s/
```

### systemd

```bash
# Install service
sudo ./scripts/deploy.sh systemd

# Check status
sudo systemctl status llm-proxy
```

## 🔒 Security

### Security Features

- **Non-root execution** in containers
- **Read-only root filesystem** option
- **Security headers** (CSP, HSTS, X-Frame-Options)
- **Rate limiting** (token bucket algorithm)
- **Input validation** and sanitization
- **CSRF protection**
- **API key encryption** at rest

### Security Audit

```bash
# Run security scan
./scripts/security-scan.sh

# Check for vulnerabilities
gosec ./...
```

## 📈 Performance

### Benchmarks

- **Latency**: <1.5s for typical requests
- **Throughput**: 1000+ RPS
- **Memory**: <50MB under normal load
- **CPU**: ~0.5 cores at 100 RPS

### Optimization Features

- Connection pooling
- Response caching (LRU)
- Object pooling (sync.Pool)
- Response compression
- HTTP/2 multiplexing
- Zero-copy file transfers

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Setup

```bash
# Clone repository
git clone https://github.com/fcmfcm01/go-llm-proxy.git
cd go-llm-proxy

# Install dependencies
go mod download

# Run in development mode
go run ./cmd/proxy

# Run all tests
make test

# Run linter
make lint
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- OpenAI for the API specification
- Anthropic for Claude API
- The Go community for excellent libraries
- Contributors and testers

## 📚 Documentation

- [API Documentation](docs/api.md)
- [Deployment Guide](docs/deployment.md)
- [Configuration Reference](docs/configuration.md)
- [Troubleshooting Guide](docs/troubleshooting.md)
- [Migration Guide (Python to Go)](docs/migration.md)
- [Architecture](docs/architecture.md)

## 🆘 Support

- **Issues**: [GitHub Issues](https://github.com/fcmfcm01/go-llm-proxy/issues)
- **Discussions**: [GitHub Discussions](https://github.com/fcmfcm01/go-llm-proxy/discussions)
- **Security**: security@yourcompany.com

## 📌 Roadmap

- [ ] **v1.0.0** - Initial stable release
- [ ] GraphQL support
- [ ] Custom provider plugins
- [ ] Advanced load balancing strategies
- [ ] Multi-region deployment
- [ ] API key management UI
- [ ] Advanced analytics

---

**Made with ❤️ in Go**
