# LLM Proxy API Documentation

This document describes the REST API for LLM Proxy, an OpenAI-compatible gateway for LLM providers.

## Table of Contents

- [Base URL](#base-url)
- [Authentication](#authentication)
- [Rate Limiting](#rate-limiting)
- [Error Handling](#error-handling)
- [OpenAI-Compatible Endpoints](#openai-compatible-endpoints)
- [Admin Endpoints](#admin-endpoints)
- [Health & Monitoring](#health--monitoring)
- [Examples](#examples)

## Base URL

```
Production:  https://api.llm-proxy.example.com/v1
Local:       http://localhost:8080/v1
```

## Authentication

### OpenAI API Key

For proxy requests, use your provider API key:

```http
Authorization: Bearer sk-your-api-key
```

### Admin Authentication

For admin endpoints, use session-based authentication:

```http
Cookie: session=your-session-token
```

Or Bearer token:

```http
Authorization: Bearer your-session-token
```

## Rate Limiting

Rate limits are applied per IP address using a token bucket algorithm.

**Headers:**

```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1640995200
```

**Exceeding limit:**

```http
HTTP/1.1 429 Too Many Requests
Retry-After: 60
```

## Error Handling

All errors follow this structure:

```json
{
  "error": {
    "message": "Error message",
    "type": "invalid_request_error",
    "code": "model_not_found"
  }
}
```

### Error Types

| Type | Description |
|------|-------------|
| `invalid_request_error` | Invalid request parameters |
| `authentication_error` | Invalid or missing authentication |
| `permission_denied` | Insufficient permissions |
| `not_found` | Resource not found |
| `rate_limit_error` | Rate limit exceeded |
| `server_error` | Internal server error |

## OpenAI-Compatible Endpoints

### Chat Completions

**Create a chat completion.**

```http
POST /v1/chat/completions
```

**Request Body:**

```json
{
  "model": "gpt-3.5-turbo",
  "messages": [
    {"role": "system", "content": "You are a helpful assistant"},
    {"role": "user", "content": "Hello!"}
  ],
  "max_tokens": 100,
  "temperature": 0.7,
  "top_p": 1,
  "n": 1,
  "stream": false,
  "logprobs": null,
  "stop": null
}
```

**Response:**

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
        "content": "Hello! How can I help you?"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 15,
    "completion_tokens": 9,
    "total_tokens": 24
  }
}
```

**Streaming Response (SSE):**

```
data: {"id": "chatcmpl-123", "object": "chat.completion", "choices": [...]}

data: {"id": "chatcmpl-123", "object": "chat.completion", "choices": [...]}

data: [DONE]
```

### Completions (Legacy)

**Generate a completion.**

```http
POST /v1/completions
```

**Request Body:**

```json
{
  "model": "text-davinci-003",
  "prompt": "Once upon a time",
  "max_tokens": 100,
  "temperature": 0.7,
  "top_p": 1,
  "n": 1,
  "stream": false,
  "logprobs": null,
  "stop": null
}
```

**Response:**

```json
{
  "id": "cmpl-123",
  "object": "text_completion",
  "created": 1677652288,
  "model": "text-davinci-003",
  "choices": [
    {
      "text": " in a land far far away...",
      "index": 0,
      "logprobs": null,
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 5,
    "completion_tokens": 10,
    "total_tokens": 15
  }
}
```

### Embeddings

**Create an embedding.**

```http
POST /v1/embeddings
```

**Request Body:**

```json
{
  "model": "text-embedding-ada-002",
  "input": "The quick brown fox",
  "user": "user-123"
}
```

**Response:**

```json
{
  "object": "list",
  "data": [
    {
      "object": "embedding",
      "embedding": [0.002, 0.003, ...],
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

### List Models

**List available models.**

```http
GET /v1/models
```

**Response:**

```json
{
  "object": "list",
  "data": [
    {
      "id": "gpt-3.5-turbo",
      "object": "model",
      "created": 1677610602,
      "owned_by": "openai"
    }
  ]
}
```

### Get Model

**Retrieve a specific model.**

```http
GET /v1/models/{model_id}
```

**Response:**

```json
{
  "id": "gpt-3.5-turbo",
  "object": "model",
  "created": 1677610602,
  "owned_by": "openai"
}
```

## Admin Endpoints

### Authentication

#### Login

**Authenticate and create a session.**

```http
POST /admin/api/v1/auth/login
```

**Request Body:**

```json
{
  "username": "admin",
  "password": "password123"
}
```

**Response:**

```http
HTTP/1.1 200 OK
Set-Cookie: session=abc123; HttpOnly; Secure; Max-Age=86400
```

#### Logout

**Invalidate the current session.**

```http
POST /admin/api/v1/auth/logout
```

**Response:**

```http
HTTP/1.1 200 OK
```

### Providers

#### List Providers

**Get all providers.**

```http
GET /admin/api/v1/providers
```

**Response:**

```json
[
  {
    "id": "prov-123",
    "name": "openai",
    "type": "openai",
    "api_key": "sk-***",
    "base_url": "https://api.openai.com/v1",
    "priority": 1,
    "enabled": true,
    "max_requests": 1000,
    "created_at": "2023-01-01T00:00:00Z",
    "updated_at": "2023-01-01T00:00:00Z"
  }
]
```

#### Create Provider

**Create a new provider.**

```http
POST /admin/api/v1/providers
```

**Request Body:**

```json
{
  "name": "anthropic",
  "type": "anthropic",
  "api_key": "sk-ant-***",
  "base_url": "https://api.anthropic.com",
  "priority": 2,
  "enabled": true,
  "max_requests": 500
}
```

**Response:**

```http
HTTP/1.1 201 Created
```

#### Update Provider

**Update an existing provider.**

```http
PUT /admin/api/v1/providers/{id}
```

**Request Body:**

```json
{
  "name": "openai",
  "priority": 1,
  "enabled": true,
  "max_requests": 2000
}
```

**Response:**

```http
HTTP/1.1 200 OK
```

#### Delete Provider

**Delete a provider.**

```http
DELETE /admin/api/v1/providers/{id}
```

**Response:**

```http
HTTP/1.1 204 No Content
```

#### Toggle Provider

**Enable or disable a provider.**

```http
POST /admin/api/v1/providers/{id}/toggle
```

**Response:**

```http
HTTP/1.1 200 OK
```

#### Get Provider Status

**Get provider health and status.**

```http
GET /admin/api/v1/providers/{id}/status
```

**Response:**

```json
{
  "id": "prov-123",
  "health": "healthy",
  "last_check": "2023-01-01T00:00:00Z",
  "response_time_ms": 150,
  "error_rate": 0.01,
  "total_requests": 1000,
  "successful_requests": 990,
  "failed_requests": 10
}
```

#### Update Provider Priority

**Update provider priority for load balancing.**

```http
POST /admin/api/v1/providers/{id}/priority
```

**Request Body:**

```json
{
  "priority": 1
}
```

**Response:**

```http
HTTP/1.1 200 OK
```

### Model Mappings

#### List Model Mappings

**Get all model mappings.**

```http
GET /admin/api/v1/model-mappings
```

**Response:**

```json
[
  {
    "id": "map-123",
    "provider_name": "openai",
    "local_name": "gpt-3.5-turbo",
    "remote_name": "gpt-3.5-turbo",
    "created_at": "2023-01-01T00:00:00Z"
  }
]
```

#### Create Model Mapping

**Create a new model mapping.**

```http
POST /admin/api/v1/model-mappings
```

**Request Body:**

```json
{
  "provider_name": "anthropic",
  "local_name": "claude-3-sonnet",
  "remote_name": "claude-3-sonnet-20240229"
}
```

#### Update Model Mapping

**Update an existing model mapping.**

```http
PUT /admin/api/v1/model-mappings/{id}
```

**Request Body:**

```json
{
  "local_name": "claude-3-sonnet",
  "remote_name": "claude-3-sonnet-20240229"
}
```

#### Delete Model Mapping

**Delete a model mapping.**

```http
DELETE /admin/api/v1/model-mappings/{id}
```

### Configuration

#### Export Configuration

**Export all configuration.**

```http
GET /admin/api/v1/config/export
```

**Response:**

```json
{
  "providers": [...],
  "model_mappings": [...],
  "settings": {...}
}
```

#### Import Configuration

**Import configuration.**

```http
POST /admin/api/v1/config/import
```

**Request Body:**

```json
{
  "providers": [...],
  "model_mappings": [...],
  "settings": {...}
}
```

### Audit Logs

#### List Audit Logs

**Get audit logs with filtering and pagination.**

```http
GET /admin/api/v1/audit-logs?page=1&limit=100&action=login&user=admin
```

**Query Parameters:**

- `page` - Page number (default: 1)
- `limit` - Items per page (default: 100, max: 1000)
- `action` - Filter by action
- `user` - Filter by user
- `start_date` - Filter from date (ISO8601)
- `end_date` - Filter to date (ISO8601)

**Response:**

```json
{
  "total": 500,
  "page": 1,
  "limit": 100,
  "logs": [
    {
      "id": "log-123",
      "timestamp": "2023-01-01T00:00:00Z",
      "user": "admin",
      "action": "login",
      "ip_address": "192.168.1.1",
      "user_agent": "Mozilla/5.0",
      "success": true
    }
  ]
}
```

#### Export Audit Logs

**Export audit logs to CSV.**

```http
GET /admin/api/v1/audit-logs/export?format=csv&start_date=2023-01-01
```

## Health & Monitoring

### Health Check (Basic)

**Basic health check endpoint.**

```http
GET /healthz
```

**Response:**

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "status": "ok",
  "timestamp": "2023-01-01T00:00:00Z"
}
```

### Readiness Check

**Check if the service is ready to receive traffic.**

```http
GET /healthz/ready
```

**Response:**

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "status": "ready",
  "checks": {
    "database": "ok",
    "providers": "ok",
    "cache": "ok"
  }
}
```

### Detailed Health Check

**Comprehensive health check with detailed information.**

```http
GET /healthz/detailed
```

**Response:**

```json
{
  "status": "ok",
  "timestamp": "2023-01-01T00:00:00Z",
  "version": "1.0.0",
  "uptime": "24h5m30s",
  "checks": {
    "database": {
      "status": "ok",
      "response_time_ms": 5
    },
    "providers": [
      {
        "name": "openai",
        "status": "ok",
        "response_time_ms": 150
      }
    ],
    "cache": {
      "status": "ok",
      "hit_rate": 0.85
    }
  },
  "metrics": {
    "total_requests": 10000,
    "successful_requests": 9900,
    "failed_requests": 100
  }
}
```

### Metrics (Prometheus)

**Prometheus-compatible metrics.**

```http
GET /metrics
```

**Response:**

```text
# HELP llm_proxy_requests_total Total number of requests
# TYPE llm_proxy_requests_total counter
llm_proxy_requests_total{endpoint="/v1/chat/completions",method="POST",status="200"} 1000

# HELP llm_proxy_request_duration_seconds Request duration
# TYPE llm_proxy_request_duration_seconds histogram
llm_proxy_request_duration_seconds_bucket{endpoint="/v1/chat/completions",method="POST",le="0.5"} 800
llm_proxy_request_duration_seconds_bucket{endpoint="/v1/chat/completions",method="POST",le="1.0"} 950
llm_proxy_request_duration_seconds_sum{endpoint="/v1/chat/completions",method="POST"} 850.5
llm_proxy_request_duration_seconds_count{endpoint="/v1/chat/completions",method="POST"} 1000

# HELP llm_proxy_provider_health Provider health status
# TYPE llm_proxy_provider_health gauge
llm_proxy_provider_health{name="openai"} 1
llm_proxy_provider_health{name="anthropic"} 1
```

## Examples

### Python Example

```python
import openai

client = openai.OpenAI(
    api_key="sk-your-api-key",
    base_url="http://localhost:8080/v1"
)

response = client.chat.completions.create(
    model="gpt-3.5-turbo",
    messages=[
        {"role": "user", "content": "Hello!"}
    ]
)

print(response.choices[0].message.content)
```

### cURL Examples

**Chat Completion:**

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-api-key" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'
```

**Streaming Chat Completion:**

```bash
curl -X POST http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-your-api-key" \
  -H "Accept: text/event-stream" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [{"role": "user", "content": "Tell me a story"}],
    "stream": true
  }'
```

**List Providers (Admin):**

```bash
curl -X GET http://localhost:8080/admin/api/v1/providers \
  -H "Cookie: session=your-session-token"
```

## SDKs

### Official OpenAI SDKs

The proxy is compatible with all OpenAI SDKs:

- **Python**: `pip install openai`
- **JavaScript**: `npm install openai`
- **Go**: `go get github.com/openai/openai-go`

### Usage

```python
# Python
from openai import OpenAI
client = OpenAI(api_key="sk-...", base_url="http://localhost:8080/v1")
```

```javascript
// JavaScript
import OpenAI from 'openai';
const client = new OpenAI({
  apiKey: 'sk-...',
  baseURL: 'http://localhost:8080/v1'
});
```

```go
// Go
package main

import (
  "context"
  "github.com/openai/openai-go"
)

client := openai.NewClient(
  openai.WithAPIKey("sk-..."),
  openai.WithBaseURL("http://localhost:8080/v1"),
)
```

## Rate Limiting

Default rate limits:

- **Default**: 1000 requests/minute
- **Burst**: 100 requests
- **Adjustable** via configuration

## Caching

### Response Caching

- **TTL**: 300 seconds (configurable)
- **Cache key**: Request method + URL + body hash
- **Enabled by default** for GET requests

### Headers

```
X-Cache: HIT
X-Cache-TTL: 300
X-Cache-Key: sha256:abc123...
```

## Versioning

The API version is included in the URL path (`/v1/`). Future versions will increment this path segment.

## WebSocket Support

Coming in v1.1:

```http
WS /v1/chat/completions
```

## Support

- **Documentation**: [Full docs](https://github.com/fcmfcm01/go-llm-proxy)
- **Issues**: [GitHub Issues](https://github.com/fcmfcm01/go-llm-proxy/issues)
- **Discussions**: [GitHub Discussions](https://github.com/fcmfcm01/go-llm-proxy/discussions)
