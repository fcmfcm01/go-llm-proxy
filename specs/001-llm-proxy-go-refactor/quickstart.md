# Quickstart Guide: LLM Proxy Server

**Version**: 1.0.0  
**Date**: 2025-11-08

## Overview

The LLM Proxy Server is a high-performance Go-based proxy that provides a unified OpenAI-compatible API for multiple LLM providers.

## Quick Start (Docker)

### 1. Clone and Configure
```bash
git clone <repository>
cd go-llm-proxy
cp deployment/.env.example deployment/.env
```

### 2. Start the Service
```bash
cd deployment
docker-compose up -d
```

### 3. Verify Installation
```bash
curl https://localhost:8443/healthz
``"

## API Usage

### OpenAI-Compatible Requests
```bash
curl -X POST https://api.proxy.example.com/v1/chat/completions   -H 'Content-Type: application/json'   -H 'Authorization: Bearer YOUR_API_KEY'   -d '{
    "model": "gpt-4",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'
```

## Admin Interface

Access at: https://admin.proxy.example.com

- Manage providers
- Configure model mappings
- View metrics
- Monitor health

## Support

See full documentation in /docs directory.

