# Quick Start Guide

Get up and running with LLM Proxy in minutes!

## Table of Contents

- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [First Run](#first-run)
- [Configure Providers](#configure-providers)
- [Test the API](#test-the-api)
- [Access Admin UI](#access-admin-ui)
- [Next Steps](#next-steps)

## Prerequisites

### System Requirements

- **OS**: Linux, macOS, or Windows
- **Memory**: 512MB minimum, 1GB recommended
- **CPU**: 1 core minimum
- **Disk**: 1GB free space

### Required Software

- **Docker** and **Docker Compose** (recommended)
- **OR** Go 1.21+ (to build from source)

### Provider API Keys

You'll need at least one LLM provider API key:

- **OpenAI**: https://platform.openai.com/api-keys
- **Anthropic**: https://console.anthropic.com/
- **Other providers**: See their documentation

## Installation

### Option 1: Docker Compose (Recommended)

**Step 1: Clone the repository**

```bash
git clone https://github.com/fcmfcm01/go-llm-proxy.git
cd go-llm-proxy
```

**Step 2: Create environment file**

```bash
# Create .env file
cat > .env << EOF
# Server Configuration
LLM_PROXY_SERVER_PORT=8080
LLM_PROXY_SERVER_MODE=development

# Authentication (CHANGE THIS!)
LLM_PROXY_AUTH_SECRET_KEY=change-me-to-a-random-string

# Providers (add your API keys)
LLM_PROXY_PROVIDER_OPENAI_API_KEY=sk-your-openai-key
LLM_PROXY_PROVIDER_ANTHROPIC_API_KEY=sk-ant-your-anthropic-key
EOF
```

**Step 3: Start services**

```bash
# Start LLM Proxy
docker-compose up -d llm-proxy

# Or start full stack with monitoring
docker-compose up -d
```

**Step 4: Verify**

```bash
# Check running containers
docker-compose ps

# Test health
curl http://localhost:8080/healthz
```

### Option 2: Build from Source

**Step 1: Install Go 1.21+**

```bash
# macOS
brew install go

# Ubuntu/Debian
wget https://go.dev/dl/go1.21.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Windows
# Download from https://go.dev/dl/
```

**Step 2: Build binary**

```bash
git clone https://github.com/fcmfcm01/go-llm-proxy.git
cd go-llm-proxy
go build -o llm-proxy ./cmd/proxy
```

**Step 3: Create configuration**

```bash
# Generate sample config
./llm-proxy config --sample > config/config.yaml

# Edit config
nano config/config.yaml
```

**Step 4: Run**

```bash
./llm-proxy serve --config config/config.yaml
```

## First Run

### Create Admin User

After starting the service, create an admin user:

```bash
# Via API
curl -X POST http://localhost:8080/admin/api/v1/auth/setup \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin123",
    "email": "admin@example.com"
  }'
```

Or use the CLI (if available in your binary):

```bash
./llm-proxy create-admin \
  --username admin \
  --password admin123 \
  --config config/config.yaml
```

### Verify Service is Running

```bash
# Check health
curl http://localhost:8080/healthz

# Expected response:
{
  "status": "ok",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

## Configure Providers

### Via Admin UI (Easiest)

1. Open browser to http://localhost:8080/admin
2. Log in with admin credentials (admin/admin123)
3. Navigate to "Providers" tab
4. Click "Add Provider"
5. Fill in details:
   - **Name**: "openai"
   - **Type**: "openai"
   - **API Key**: Your OpenAI API key
   - **Base URL**: "https://api.openai.com/v1"
   - **Priority**: 1
   - **Enabled**: Yes
6. Click "Save"
7. Repeat for Anthropic if needed

### Via API

**Add OpenAI provider:**

```bash
curl -X POST http://localhost:8080/admin/api/v1/providers \
  -H "Content-Type: application/json" \
  -H "Cookie: session=your-session-cookie" \
  -d '{
    "name": "openai",
    "type": "openai",
    "api_key": "sk-your-openai-key",
    "base_url": "https://api.openai.com/v1",
    "priority": 1,
    "enabled": true,
    "max_requests": 1000
  }'
```

**Add Anthropic provider:**

```bash
curl -X POST http://localhost:8080/admin/api/v1/providers \
  -H "Content-Type: application/json" \
  -H "Cookie: your-session-cookie" \
  -d '{
    "name": "anthropic",
    "type": "anthropic",
    "api_key": "sk-ant-your-anthropic-key",
    "base_url": "https://api.anthropic.com",
    "priority": 2,
    "enabled": true,
    "max_requests": 500
  }'
```

**Verify providers:**

```bash
curl http://localhost:8080/admin/api/v1/providers \
  -H "Cookie: your-session-cookie"
```

### Via Configuration File

Edit `config/config.yaml`:

```yaml
server:
  port: 8080
  host: "0.0.0.0"

auth:
  session_timeout: 24h
  secret_key: "change-me-to-a-random-string"

providers:
  - name: "openai"
    type: "openai"
    api_key: "sk-your-openai-key"
    base_url: "https://api.openai.com/v1"
    priority: 1
    enabled: true
    max_requests: 1000

  - name: "anthropic"
    type: "anthropic"
    api_key: "sk-ant-your-anthropic-key"
    base_url: "https://api.anthropic.com"
    priority: 2
    enabled: true
    max_requests: 500
```

Restart the service:

```bash
# Docker
docker-compose restart llm-proxy

# systemd
sudo systemctl restart llm-proxy

# Direct binary
# Stop and start again
```

## Test the API

### 1. Chat Completion (OpenAI-Compatible)

**Using curl:**

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-openai-key" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {"role": "system", "content": "You are a helpful assistant"},
      {"role": "user", "content": "Hello! How are you?"}
    ]
  }'
```

**Expected response:**

```json
{
  "id": "chatcmpl-123",
  "object": "chat.completion",
  "created": 1677652288,
  "model": "gpt-3.5-turbo",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "Hello! I'\''m doing well, thank you. How can I help you today?"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 15,
    "completion_tokens": 12,
    "total_tokens": 27
  }
}
```

**Using Python OpenAI SDK:**

```python
from openai import OpenAI

client = OpenAI(
    api_key="sk-your-openai-key",
    base_url="http://localhost:8080/v1"
)

response = client.chat.completions.create(
    model="gpt-3.5-turbo",
    messages=[
        {"role": "user", "content": "Hello! How are you?"}
    ]
)

print(response.choices[0].message.content)
```

**Using JavaScript OpenAI SDK:**

```javascript
import OpenAI from 'openai';

const client = new OpenAI({
  apiKey: 'sk-your-openai-key',
  baseURL: 'http://localhost:8080/v1',
});

const response = await client.chat.completions.create({
  model: 'gpt-3.5-turbo',
  messages: [
    {role: 'user', content: 'Hello! How are you?'}
  ],
});

console.log(response.choices[0].message.content);
```

### 2. Streaming Response

**Using curl with Server-Sent Events:**

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-openai-key" \
  -H "Accept: text/event-stream" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {"role": "user", "content": "Tell me a short story"}
    ],
    "stream": true
  }'
```

**Expected streaming output:**

```
data: {"id":"chatcmpl-123","object":"chat.completion","choices":[{"index":0,"delta":{"content":"Once"}}]}

data: {"id":"chatcmpl-123","object":"chat.completion","choices":[{"index":0,"delta":{"content":" upon"}}]}

data: {"id":"chatcmpl-123","object":"chat.completion","choices":[{"index":0,"delta":{"content":" a time"}}]}

data: {"id":"chatcmpl-123","object":"chat.completion","choices":[{"index":0,"delta":{"content":", there was a"}}]}

data: [DONE]
```

**Using Python with streaming:**

```python
from openai import OpenAI

client = OpenAI(
    api_key="sk-your-openai-key",
    base_url="http://localhost:8080/v1"
)

stream = client.chat.completions.create(
    model="gpt-3.5-turbo",
    messages=[
        {"role": "user", "content": "Tell me a story"}
    ],
    stream=True
)

for chunk in stream:
    if chunk.choices[0].delta.content is not None:
        print(chunk.choices[0].delta.content, end="")
```

### 3. List Available Models

```bash
curl -H "Authorization: Bearer sk-your-openai-key" \
  http://localhost:8080/v1/models
```

**Expected response:**

```json
{
  "object": "list",
  "data": [
    {
      "id": "gpt-3.5-turbo",
      "object": "model",
      "created": 1677610602,
      "owned_by": "openai"
    },
    {
      "id": "gpt-4",
      "object": "model",
      "created": 1677610602,
      "owned_by": "openai"
    }
  ]
}
```

### 4. Embeddings

```bash
curl -X POST http://localhost:8080/v1/embeddings \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-openai-key" \
  -d '{
    "model": "text-embedding-ada-002",
    "input": "The quick brown fox"
  }'
```

**Expected response:**

```json
{
  "object": "list",
  "data": [
    {
      "object": "embedding",
      "embedding": [0.002, 0.003, -0.001, ...],
      "index": 0
    }
  ],
  "model": "text-embedding-ada-002",
  "usage": {
    "prompt_tokens": 5,
    "total_tokens": 5
  }
}
```

### 5. Using Anthropic Models

**Request Anthropic model via OpenAI-compatible endpoint:**

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-anthropic-key" \
  -d '{
    "model": "claude-3-sonnet-20240229",
    "messages": [
      {"role": "user", "content": "Hello! How are you?"}
    ]
  }'
```

**The proxy automatically:**
- Converts OpenAI format to Anthropic format
- Routes to the Anthropic provider
- Converts the response back to OpenAI format
- Returns OpenAI-compatible response

## Access Admin UI

### Open Admin Interface

Navigate to: **http://localhost:8080/admin**

**Default credentials:**
- Username: `admin`
- Password: `admin123` (change immediately!)

### Admin UI Features

#### Dashboard

View system overview:
- Total requests
- Active providers
- Health status
- Response time metrics

#### Providers

Manage LLM providers:
- **List** all configured providers
- **Add** new providers
- **Edit** provider settings
- **Delete** providers
- **Toggle** enable/disable
- **View** health and status
- **Adjust** priority

#### Model Mappings

Configure model name mappings:
- Map local model names to provider models
- Auto-create mappings for new models
- Override default mappings

#### Monitoring

Real-time provider monitoring:
- Health status per provider
- Response time metrics
- Error rates
- Request volume

#### Configuration

- Export configuration
- Import configuration
- Reload configuration (without restart)
- View audit logs

### Create Additional Admin Users

**Via API:**

```bash
curl -X POST http://localhost:8080/admin/api/v1/users \
  -H "Content-Type: application/json" \
  -H "Cookie: admin-session" \
  -d '{
    "username": "user2",
    "password": "secure-password",
    "email": "user2@example.com"
  }'
```

### Change Admin Password

**Via Admin UI:**
1. Log in
2. Click your username (top right)
3. Click "Profile"
4. Update password

**Via API:**

```bash
curl -X PUT http://localhost:8080/admin/api/v1/users/admin \
  -H "Content-Type: application/json" \
  -H "Cookie: admin-session" \
  -d '{
    "password": "new-secure-password"
  }'
```

## Common Use Cases

### 1. OpenAI-Compatible Gateway

Use as a drop-in replacement for OpenAI API:

```python
# Your existing code
from openai import OpenAI

client = OpenAI(
    api_key="sk-your-provider-key",
    base_url="http://localhost:8080/v1"  # Just change this!
)

# Everything else works the same
response = client.chat.completions.create(
    model="gpt-3.5-turbo",
    messages=[...]
)
```

### 2. Multi-Provider Failover

Configure multiple providers for high availability:

```yaml
providers:
  - name: "openai-us"
    type: "openai"
    api_key: "sk-..."
    base_url: "https://api.openai.com/v1"
    priority: 1
    enabled: true
    weight: 0.7

  - name: "openai-eu"
    type: "openai"
    api_key: "sk-..."
    base_url: "https://api.openai.com/v1"
    priority: 2
    enabled: true
    weight: 0.3

  - name: "anthropic"
    type: "anthropic"
    api_key: "sk-ant-..."
    base_url: "https://api.anthropic.com"
    priority: 3
    enabled: true
    weight: 0.2
```

The proxy will:
1. Route to highest priority provider
2. Load balance across providers with same priority
3. Automatically fail over if provider is down
4. Use weighted distribution

### 3. API Key Management

**Single API key for all requests:**

Configure provider API keys once, then use any key for client requests:

```yaml
# Server config
providers:
  - name: "openai"
    type: "openai"
    api_key: "${OPENAI_API_KEY}"  # Set via environment
    base_url: "https://api.openai.com/v1"
    priority: 1
    enabled: true
```

```bash
# Client can use any Bearer token
curl -H "Authorization: Bearer any-key-here" \
  http://localhost:8080/v1/chat/completions \
  -d '...'
```

### 4. Rate Limiting

**Global rate limiting:**

```yaml
security:
  rate_limit:
    enabled: true
    requests_per_minute: 1000
    burst: 100
```

**Per-provider rate limiting:**

```yaml
providers:
  - name: "openai"
    type: "openai"
    api_key: "sk-..."
    rate_limit:
      requests_per_minute: 500
      burst: 50
```

### 5. Caching

**Enable response caching:**

```yaml
performance:
  cache_enabled: true
  cache_ttl: 300  # 5 minutes
  cache_size: 1000
```

**View cache statistics:**

```bash
# View metrics
curl http://localhost:8080/metrics | grep cache

# Example metrics:
# llm_proxy_cache_hits_total 850
# llm_proxy_cache_misses_total 150
# llm_proxy_cache_size 45
```

## Next Steps

### Production Deployment

1. **Read the [Deployment Guide](docs/deployment.md)**
   - Kubernetes deployment
   - Production Docker Compose
   - systemd service
   - Cloud platforms (AWS, GCP, Azure)

2. **Review [Configuration Reference](docs/configuration.md)**
   - All configuration options
   - Environment variables
   - Best practices

3. **Set up monitoring**
   - Prometheus metrics at `/metrics`
   - Grafana dashboard (see deployment/)
   - Health checks at `/healthz`

### Learning More

- **[API Documentation](docs/api.md)** - Complete API reference
- **[Configuration Guide](docs/configuration.md)** - All config options
- **[Deployment Guide](docs/deployment.md)** - Production deployment
- **[Architecture](docs/architecture.md)** - System design
- **[Migration Guide](docs/migration.md)** - Migrate from Python version
- **[Troubleshooting](docs/troubleshooting.md)** - Common issues

### Example Projects

Check out example implementations:

```bash
# Python examples
git clone https://github.com/fcmfcm01/go-llm-proxy-examples.git
cd go-llm-proxy-examples/python
```

```bash
# JavaScript examples
cd go-llm-proxy-examples/javascript
```

### Support

- **Documentation**: [GitHub Wiki](https://github.com/fcmfcm01/go-llm-proxy/wiki)
- **Issues**: [GitHub Issues](https://github.com/fcmfcm01/go-llm-proxy/issues)
- **Discussions**: [GitHub Discussions](https://github.com/fcmfcm01/go-llm-proxy/discussions)

## Troubleshooting

### Service Won't Start

```bash
# Check if port is already in use
lsof -i :8080
netstat -tulpn | grep 8080

# Check configuration
./llm-proxy validate-config config/config.yaml

# View logs
tail -f /var/log/llm-proxy/app.log
```

### Provider Connection Failed

```bash
# Test provider connection directly
curl -H "Authorization: Bearer sk-your-key" \
  https://api.openai.com/v1/models

# Check provider status via API
curl http://localhost:8080/admin/api/v1/providers/status
```

### Authentication Not Working

```bash
# Create admin user if not exists
curl -X POST http://localhost:8080/admin/api/v1/auth/setup \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'

# Test login
curl -X POST http://localhost:8080/admin/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'
```

### Performance Issues

```bash
# Check metrics
curl http://localhost:8080/metrics

# Check health
curl http://localhost:8080/healthz/detailed

# Enable debug logging
export LLM_PROXY_LOGGING_LEVEL=debug
```

For more help, see **[Troubleshooting Guide](docs/troubleshooting.md)**

## Security Notes

### For Development

- Use default credentials for testing
- Enable debug logging
- Use HTTP (not HTTPS)
- No rate limiting

### For Production

- **CHANGE DEFAULT PASSWORD** immediately
- Use HTTPS/TLS
- Set strong session secret (32+ characters)
- Enable rate limiting
- Enable audit logging
- Use Redis for session storage
- Keep dependencies updated

## Performance Tips

1. **Enable caching** for repeated requests
2. **Use connection pooling** for better throughput
3. **Enable gzip compression** to reduce bandwidth
4. **Monitor provider health** for optimal routing
5. **Use multiple providers** for load distribution

## Summary

You now have:
- ✅ LLM Proxy running
- ✅ Admin UI accessible
- ✅ At least one provider configured
- ✅ API working with OpenAI-compatible requests
- ✅ Admin user created

**You're all set!** Start building with the OpenAI-compatible API at `http://localhost:8080/v1`

---

**Need help?** Check out the full documentation or open an issue on GitHub.
