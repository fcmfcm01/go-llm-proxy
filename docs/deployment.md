# Deployment Guide

This guide covers deploying LLM Proxy to various environments including development, staging, and production.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Deployment Methods](#deployment-methods)
- [Development Deployment](#development-deployment)
- [Docker Deployment](#docker-deployment)
- [Kubernetes Deployment](#kubernetes-deployment)
- [Systemd Deployment](#systemd-deployment)
- [Cloud Deployment](#cloud-deployment)
- [Configuration](#configuration)
- [Security](#security)
- [Monitoring](#monitoring)
- [Scaling](#scaling)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### System Requirements

| Environment | CPU | Memory | Storage | Network |
|-------------|-----|--------|---------|---------|
| Development | 1 core | 1GB | 10GB | 1 Gbps |
| Staging | 1 core | 2GB | 20GB | 1 Gbps |
| Production | 2+ cores | 4GB+ | 100GB+ | 10 Gbps |

### Software Dependencies

- **Go**: 1.21+ (for building from source)
- **Docker**: 20.10+ (for containerized deployment)
- **Docker Compose**: 2.0+ (for multi-container setup)
- **kubectl**: 1.25+ (for Kubernetes deployment)
- **systemd**: 249+ (for Linux service)

### Network Requirements

- **Ports**:
  - 8080 (HTTP) - Required
  - 8443 (HTTPS) - Recommended
  - 9090 (Metrics) - Optional
  - 3000 (Grafana) - Optional
  - 9090 (Prometheus) - Optional

### Provider API Keys

Obtain API keys from your LLM providers:

- **OpenAI**: https://platform.openai.com/api-keys
- **Anthropic**: https://console.anthropic.com/
- **Other providers**: See their documentation

## Deployment Methods

LLM Proxy supports multiple deployment methods:

1. **Docker Compose** - Quickest for development/staging
2. **Kubernetes** - Production-ready, scalable
3. **systemd** - Traditional Linux service
4. **Cloud** - AWS ECS, GCP Cloud Run, Azure Container Apps
5. **Binary** - Run directly without containers

Choose based on your needs:

| Method | Best For | Pros | Cons |
|--------|----------|------|------|
| Docker Compose | Development, Testing | Easy, reproducible | Not for production |
| Kubernetes | Production | Scalable, HA | Complex setup |
| systemd | Single server | Simple, lightweight | Manual scaling |
| Cloud | Cloud-native | Managed, elastic | Vendor lock-in |
| Binary | Minimal deployments | Fast, no dependencies | Manual management |

## Development Deployment

### Quick Start (Docker Compose)

```bash
# Clone the repository
git clone https://github.com/fcmfcm01/go-llm-proxy.git
cd go-llm-proxy

# Copy environment template
cp deployment/.env.example deployment/.env

# Edit configuration
nano deployment/.env

# Start services
docker-compose up -d

# Check status
docker-compose ps
```

### Build from Source

```bash
# Clone repository
git clone https://github.com/fcmfcm01/go-llm-proxy.git
cd go-llm-proxy

# Build binary
go build -o llm-proxy ./cmd/proxy

# Run directly
./llm-proxy serve --config config/config.yaml
```

### Development Configuration

Create `config/development.yaml`:

```yaml
server:
  port: 8080
  mode: "development"
  enable_tls: false
  read_timeout: 30
  write_timeout: 30

auth:
  session_timeout: 12h
  secret_key: "dev-secret-key-change-me"

logging:
  level: "debug"
  format: "text"
  enable_audit: false

metrics:
  enabled: true
  port: 9090

proxy:
  timeout: 30
  max_retries: 3

performance:
  connection_pool_size: 50
  cache_enabled: false
  enable_compression: false
```

## Docker Deployment

### Single Container

```bash
# Build image
docker build -t go-llm-proxy:latest .

# Run container
docker run -d \
  --name llm-proxy \
  -p 8080:8080 \
  -p 8443:8443 \
  -v $(pwd)/config:/app/config \
  -v $(pwd)/data:/app/data \
  go-llm-proxy:latest
```

### With Environment File

```bash
# Create .env file
cat > .env << EOF
LLM_PROXY_SECURITY_SESSION_SECRET=your-secret-key
LLM_PROXY_PROVIDER_OPENAI_API_KEY=sk-...
LLM_PROXY_PROVIDER_ANTHROPIC_API_KEY=sk-ant-...
EOF

# Run with environment
docker run -d \
  --name llm-proxy \
  --env-file .env \
  -p 8080:8080 \
  go-llm-proxy:latest
```

### Docker Compose (Production)

```bash
# Use production compose
docker-compose -f deployment/docker-compose.prod.yml up -d
```

### Health Check

```bash
# Check container health
docker ps

# View logs
docker logs llm-proxy

# Execute health check
docker exec llm-proxy /app/llm-proxy healthcheck
```

### Image Optimization

```dockerfile
# Multi-stage build for smaller image
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o llm-proxy ./cmd/proxy

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/llm-proxy /app/
CMD ["/app/llm-proxy"]
```

## Kubernetes Deployment

### Quick Deploy

```bash
# Clone repository
git clone https://github.com/fcmfcm01/go-llm-proxy.git
cd go-llm-proxy

# Create namespace
kubectl create namespace llm-proxy

# Apply manifests
kubectl apply -f deployment/k8s/

# Check status
kubectl get pods -n llm-proxy
kubectl get svc -n llm-proxy
```

### Using Kustomize

```bash
# Customize deployment
cat > kustomization.yaml << EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - deployment/k8s/
images:
  - name: go-llm-proxy
    newName: your-registry/go-llm-proxy
    newTag: v1.0.0
replicas:
  - name: llm-proxy
    count: 3
EOF

# Deploy with customization
kubectl apply -k .
```

### Production Configuration

```yaml
# production-values.yaml
replicaCount: 3

image:
  repository: your-registry/go-llm-proxy
  tag: "v1.0.0"
  pullPolicy: Always

resources:
  limits:
    cpu: 1000m
    memory: 512Mi
  requests:
    cpu: 500m
    memory: 256Mi

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
  targetMemoryUtilizationPercentage: 80

ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
  hosts:
    - host: api.llm-proxy.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: llm-proxy-tls
      hosts:
        - api.llm-proxy.example.com
```

### Install with Helm

```bash
# Add Helm repo (if published)
helm repo add llm-proxy https://fcmfcm01.github.io/llm-proxy

# Install with values
helm install llm-proxy llm-proxy/llm-proxy \
  --namespace llm-proxy \
  --create-namespace \
  -f production-values.yaml
```

### Verify Deployment

```bash
# Check pods
kubectl get pods -n llm-proxy

# Check services
kubectl get svc -n llm-proxy

# Check ingress
kubectl get ingress -n llm-proxy

# Test endpoint
kubectl port-forward svc/llm-proxy 8080:80 -n llm-proxy
curl http://localhost:8080/healthz
```

### Update Deployment

```bash
# Update image
kubectl set image deployment/llm-proxy \
  llm-proxy=go-llm-proxy:v1.1.0 \
  -n llm-proxy

# Check rollout
kubectl rollout status deployment/llm-proxy -n llm-proxy

# Rollback if needed
kubectl rollout undo deployment/llm-proxy -n llm-proxy
```

## systemd Deployment

### Installation

```bash
# Clone repository
git clone https://github.com/fcmfcm01/go-llm-proxy.git
cd go-llm-proxy

# Run deployment script
sudo ./scripts/deploy.sh systemd
```

### Manual Installation

```bash
# Create user
sudo useradd -r -s /bin/false -d /opt/llm-proxy -M llm-proxy

# Create directories
sudo mkdir -p /opt/llm-proxy/{bin,data,config}
sudo mkdir -p /var/log/llm-proxy

# Install binary
sudo cp llm-proxy /opt/llm-proxy/bin/
sudo chmod +x /opt/llm-proxy/bin/llm-proxy
sudo chown -R llm-proxy:llm-proxy /opt/llm-proxy

# Install service
sudo cp deployment/systemd/llm-proxy.service /etc/systemd/system/
sudo systemctl daemon-reload

# Configure environment
sudo cp deployment/systemd/llm-proxy.env /etc/default/llm-proxy
sudo chmod 640 /etc/default/llm-proxy
sudo chown root:llm-proxy /etc/default/llm-proxy

# Edit configuration
sudo nano /etc/default/llm-proxy

# Start service
sudo systemctl enable llm-proxy
sudo systemctl start llm-proxy

# Check status
sudo systemctl status llm-proxy
```

### Management

```bash
# Start service
sudo systemctl start llm-proxy

# Stop service
sudo systemctl stop llm-proxy

# Restart service
sudo systemctl restart llm-proxy

# Reload configuration
sudo systemctl reload llm-proxy

# View logs
sudo journalctl -u llm-proxy -f

# View status
sudo systemctl status llm-proxy
```

## Cloud Deployment

### AWS ECS

```json
{
  "family": "llm-proxy",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "512",
  "memory": "1024",
  "executionRoleArn": "arn:aws:iam::ACCOUNT:role/ecsTaskExecutionRole",
  "taskRoleArn": "arn:aws:iam::ACCOUNT:role/ecsTaskRole",
  "containerDefinitions": [
    {
      "name": "llm-proxy",
      "image": "your-account.dkr.ecr.region.amazonaws.com/go-llm-proxy:latest",
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        },
        {
          "containerPort": 8443,
          "protocol": "tcp"
        }
      ],
      "essential": true,
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/llm-proxy",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "ecs"
        }
      },
      "environment": [
        {
          "name": "LLM_PROXY_SERVER_MODE",
          "value": "production"
        }
      ]
    }
  ]
}
```

### GCP Cloud Run

```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: llm-proxy
  annotations:
    run.googleapis.com/ingress: all
spec:
  template:
    metadata:
      annotations:
        run.googleapis.com/cpu-throttling: "false"
        run.googleapis.com/memory: "1Gi"
        run.googleapis.com/cpu: "1"
    spec:
      containerConcurrency: 100
      containers:
      - image: gcr.io/PROJECT/go-llm-proxy:latest
        ports:
        - containerPort: 8080
        - containerPort: 8443
        env:
        - name: LLM_PROXY_SERVER_MODE
          value: "production"
        resources:
          limits:
            cpu: 1000m
            memory: 1Gi
```

### Azure Container Apps

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: llm-proxy
spec:
  replicas: 2
  selector:
    matchLabels:
      app: llm-proxy
  template:
    metadata:
      labels:
        app: llm-proxy
    spec:
      containers:
      - name: llm-proxy
        image: your-registry.azurecr.io/go-llm-proxy:latest
        ports:
        - containerPort: 8080
        - containerPort: 8443
        resources:
          limits:
            cpu: "1"
            memory: "2Gi"
        env:
        - name: LLM_PROXY_SERVER_MODE
          value: "production"
```

## Configuration

### Configuration File

Create `config/production.yaml`:

```yaml
server:
  port: 8080
  tls_port: 8443
  host: "0.0.0.0"
  mode: "production"
  read_timeout: 30
  write_timeout: 30
  idle_timeout: 90
  enable_tls: true
  tls_cert: "/certs/tls.crt"
  tls_key: "/certs/tls.key"

auth:
  session_timeout: 24h
  secret_key: "${LLM_PROXY_SECRET_KEY}"

logging:
  level: "info"
  format: "json"
  file: "/var/log/llm-proxy/app.log"
  enable_audit: true

metrics:
  enabled: true
  port: 9090

proxy:
  timeout: 30
  max_retries: 3
  request_timeout: 60
  idle_timeout: 90

performance:
  connection_pool_size: 100
  cache_enabled: true
  cache_ttl: 300
  enable_compression: true

providers:
  - name: "openai"
    type: "openai"
    api_key: "${OPENAI_API_KEY}"
    base_url: "https://api.openai.com/v1"
    priority: 1
    enabled: true
    max_requests: 1000
    max_retries: 3
    timeout: 30

  - name: "anthropic"
    type: "anthropic"
    api_key: "${ANTHROPIC_API_KEY}"
    base_url: "https://api.anthropic.com"
    priority: 2
    enabled: true
    max_requests: 500
    max_retries: 3
    timeout: 30
```

### Environment Variables

```bash
# Core configuration
export LLM_PROXY_SECRET_KEY="your-secret-key"
export LLM_PROXY_SERVER_MODE="production"

# Provider API keys
export OPENAI_API_KEY="sk-..."
export ANTHROPIC_API_KEY="sk-ant-..."

# Logging
export LLM_PROXY_LOGGING_LEVEL="info"
export LLM_PROXY_LOGGING_FORMAT="json"

# Performance
export LLM_PROXY_PERFORMANCE_CACHE_TTL="300"
export LLM_PROXY_PERFORMANCE_CONNECTION_POOL_SIZE="100"
```

## Security

### TLS/HTTPS

```bash
# Generate self-signed certificate (development)
openssl req -x509 -newkey rsa:4096 -keyout tls.key -out tls.crt -days 365 -nodes

# Use Let's Encrypt (production)
certbot certonly --standalone -d api.example.com
```

### Firewall Rules

```bash
# UFW (Ubuntu)
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable

# iptables
sudo iptables -A INPUT -p tcp --dport 22 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 80 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 443 -j ACCEPT
```

### Security Headers

The service automatically adds:
- Strict-Transport-Security
- X-Content-Type-Options
- X-Frame-Options
- X-XSS-Protection
- Content-Security-Policy

### Secrets Management

#### Kubernetes Secrets

```bash
# Create secret
kubectl create secret generic llm-proxy-secrets \
  --from-literal=session-secret="your-secret" \
  --from-literal=openai-api-key="sk-..." \
  -n llm-proxy

# Use in deployment
env:
- name: LLM_PROXY_SECURITY_SESSION_SECRET
  valueFrom:
    secretKeyRef:
      name: llm-proxy-secrets
      key: session-secret
```

#### Docker Secrets

```bash
# Create secret
echo "your-secret" | docker secret create session_secret -

# Use in service
services:
  llm-proxy:
    image: go-llm-proxy:latest
    secrets:
      - session_secret
```

## Monitoring

### Prometheus Integration

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'llm-proxy'
    static_configs:
      - targets: ['llm-proxy:9090']
    metrics_path: /metrics
    scrape_interval: 30s
```

### Grafana Dashboard

```bash
# Import dashboard
curl -X POST \
  http://admin:admin@grafana:3000/api/dashboards/db \
  -H 'Content-Type: application/json' \
  -d @deployment/grafana/dashboards/llm-proxy.json
```

### Alerting

```yaml
# alerts.yml
groups:
  - name: llm-proxy
    rules:
      - alert: LLMProxyDown
        expr: up{job="llm-proxy"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "LLM Proxy is down"

      - alert: LLMProxyHighErrorRate
        expr: rate(llm_proxy_requests_total{status=~"5.."}[5m]) / rate(llm_proxy_requests_total[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High error rate detected"
```

## Scaling

### Horizontal Scaling (Multiple Instances)

```yaml
# docker-compose.scale.yml
services:
  llm-proxy:
    image: go-llm-proxy:latest
    deploy:
      replicas: 3
    ports:
      - "8080-8082:8080"
    environment:
      - INSTANCE_ID=${INSTANCE_ID}
```

### Load Balancer (Nginx)

```nginx
# /etc/nginx/sites-available/llm-proxy
upstream llm_proxy {
    server 10.0.1.10:8080;
    server 10.0.1.11:8080;
    server 10.0.1.12:8080;
}

server {
    listen 443 ssl;
    server_name api.example.com;

    ssl_certificate /etc/ssl/certs/api.crt;
    ssl_certificate_key /etc/ssl/private/api.key;

    location / {
        proxy_pass http://llm_proxy;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Database (if using external session store)

```yaml
# docker-compose with Redis
services:
  redis:
    image: redis:7-alpine
    command: redis-server --requirepass password
    volumes:
      - redis-data:/data

  llm-proxy:
    image: go-llm-proxy:latest
    environment:
      - REDIS_URL=redis://:password@redis:6379
    depends_on:
      - redis
```

## Troubleshooting

### Check Service Health

```bash
# Health check
curl http://localhost:8080/healthz

# Detailed health
curl http://localhost:8080/healthz/detailed

# Metrics
curl http://localhost:8080/metrics
```

### View Logs

```bash
# Docker
docker logs llm-proxy

# systemd
sudo journalctl -u llm-proxy -f

# Kubernetes
kubectl logs -f deployment/llm-proxy -n llm-proxy

# File logs
tail -f /var/log/llm-proxy/app.log
```

### Common Issues

**Port already in use:**

```bash
# Find process
sudo lsof -i :8080

# Kill process
sudo kill -9 PID
```

**Permission denied:**

```bash
# Fix permissions
sudo chown -R llm-proxy:llm-proxy /opt/llm-proxy
sudo chmod +x /opt/llm-proxy/bin/llm-proxy
```

**Configuration not loading:**

```bash
# Validate config
./llm-proxy validate-config config/production.yaml

# Check file permissions
ls -la config/
```

### Debug Mode

```bash
# Run with debug logging
LLM_PROXY_LOGGING_LEVEL=debug ./llm-proxy serve

# Enable pprof
curl http://localhost:8080/debug/pprof/
```

### Performance Issues

```bash
# Check resource usage
docker stats llm-proxy
kubectl top pods -n llm-proxy

# Profile CPU
go tool pprof http://localhost:8080/debug/pprof/profile

# Profile memory
go tool pprof http://localhost:8080/debug/pprof/heap
```

## Best Practices

### Production Checklist

- [ ] Use TLS/HTTPS
- [ ] Set strong session secret
- [ ] Configure rate limiting
- [ ] Enable audit logging
- [ ] Set up monitoring (Prometheus)
- [ ] Configure alerts
- [ ] Use load balancer
- [ ] Enable caching
- [ ] Configure log rotation
- [ ] Set up backups
- [ ] Use secrets management
- [ ] Enable security headers
- [ ] Run as non-root user
- [ ] Use read-only filesystem
- [ ] Set resource limits

### Deployment Checklist

- [ ] Test in staging environment
- [ ] Run all tests
- [ ] Check security scan
- [ ] Verify performance
- [ ] Review configuration
- [ ] Check SSL certificates
- [ ] Test health checks
- [ ] Verify monitoring
- [ ] Check backup/restore
- [ ] Review documentation

## Support

- **Documentation**: [Full docs](https://github.com/fcmfcm01/go-llm-proxy)
- **Issues**: [GitHub Issues](https://github.com/fcmfcm01/go-llm-proxy/issues)
- **Discussions**: [GitHub Discussions](https://github.com/fcmfcm01/go-llm-proxy/discussions)
- **Email**: support@yourcompany.com
