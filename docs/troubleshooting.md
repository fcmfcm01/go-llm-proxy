# Troubleshooting Guide

This guide helps you diagnose and resolve common issues with LLM Proxy.

## Table of Contents

- [General Troubleshooting](#general-troubleshooting)
- [Startup Issues](#startup-issues)
- [Configuration Problems](#configuration-problems)
- [Authentication Issues](#authentication-issues)
- [Provider Connection Problems](#provider-connection-problems)
- [Performance Issues](#performance-issues)
- [High Availability Issues](#high-availability-issues)
- [Memory and CPU Issues](#memory-and-cpu-issues)
- [Network and Connectivity](#network-and-connectivity)
- [Docker and Container Issues](#docker-and-container-issues)
- [Kubernetes Issues](#kubernetes-issues)
- [Monitoring and Metrics](#monitoring-and-metrics)
- [Logging and Debugging](#logging-and-debugging)
- [Migration Issues](#migration-issues)
- [Error Reference](#error-reference)

## General Troubleshooting

### Quick Diagnostic Commands

```bash
# Check service health
curl http://localhost:8080/healthz

# Check detailed health
curl http://localhost:8080/healthz/detailed

# View metrics
curl http://localhost:8080/metrics

# Test provider connection
./llm-proxy test-providers

# Validate configuration
./llm-proxy validate-config config/config.yaml

# Check logs
tail -f /var/log/llm-proxy/app.log
```

### Enable Debug Logging

```bash
# Set debug level
export LLM_PROXY_LOGGING_LEVEL=debug
./llm-proxy serve

# Or in config
logging:
  level: "debug"
```

### Common First Steps

1. **Check health endpoints** - Always start with `/healthz` and `/healthz/detailed`
2. **Verify configuration** - Run `validate-config` before investigating further
3. **Check provider status** - Ensure API keys and endpoints are correct
4. **Review logs** - Start with `error` and `warn` level logs
5. **Test with curl** - Use simple HTTP requests to isolate the issue

## Startup Issues

### Service Won't Start

**Symptoms:**
- Binary exits immediately
- "port already in use" error
- "permission denied" error
- Configuration file not found

**Diagnosis:**

```bash
# Check if port is in use
lsof -i :8080
netstat -tulpn | grep 8080

# Check file permissions
ls -la config/config.yaml

# Check service status (systemd)
sudo systemctl status llm-proxy

# View startup logs
./llm-proxy serve --config config/config.yaml --log-level debug
```

**Solutions:**

**Port in use:**
```bash
# Find and kill process using the port
sudo lsof -ti:8080 | xargs sudo kill -9

# Or use a different port
server:
  port: 8081
```

**Permission denied:**
```bash
# Fix configuration file permissions
sudo chown llm-proxy:llm-proxy config/config.yaml
sudo chmod 600 config/config.yaml

# Or for development
chmod 644 config/config.yaml
```

**Configuration not found:**
```bash
# Verify config file exists
ls -la config/

# Use absolute path
./llm-proxy serve --config /path/to/config/config.yaml

# Generate sample config
./llm-proxy config --sample > config/config.yaml
```

### Invalid Configuration

**Symptoms:**
- "config validation failed" error
- "unrecognized key" warning
- Type mismatch errors

**Diagnosis:**

```bash
# Validate configuration
./llm-proxy validate-config config/config.yaml

# Check syntax
yamllint config/config.yaml

# View effective configuration
./llm-proxy config --dump --config config/config.yaml
```

**Solutions:**

**Environment variable substitution:**
```yaml
# Wrong - needs quotes
server:
  port: ${PORT}  # This won't work, needs to be quoted

# Correct
server:
  port: "${PORT}"  # Properly quoted
```

**Invalid enum values:**
```yaml
# Wrong
server:
  mode: "prod"  # Should be "production"

# Correct
server:
  mode: "production"
```

**Missing required fields:**
```yaml
# Wrong - missing secret_key
auth:
  session_timeout: 24h

# Correct
auth:
  session_timeout: 24h
  secret_key: "your-secret-key"  # Required
```

### TLS Certificate Issues

**Symptoms:**
- "certificate not found" error
- "private key doesn't match" error
- TLS handshake failures

**Diagnosis:**

```bash
# Check certificate validity
openssl x509 -in /path/to/cert.pem -text -noout

# Check private key matches certificate
openssl x509 -noout -modulus -in cert.pem | openssl md5
openssl rsa -noout -modulus -in key.pem | openssl md5

# Test TLS connection
openssl s_client -connect localhost:8443 -servername localhost
```

**Solutions:**

**Generate self-signed certificate (development):**
```bash
# Create directory
mkdir -p certs

# Generate certificate
openssl req -x509 -newkey rsa:4096 \
  -keyout certs/tls.key \
  -out certs/tls.crt \
  -days 365 \
  -nodes \
  -subj "/CN=localhost"

# Update config
server:
  enable_tls: true
  tls_cert: "certs/tls.crt"
  tls_key: "certs/tls.key"
```

**Use Let's Encrypt (production):**
```bash
# Install certbot
sudo apt-get install certbot

# Obtain certificate
sudo certbot certonly --standalone -d api.example.com

# Configure (certificates in /etc/letsencrypt/)
server:
  enable_tls: true
  tls_cert: "/etc/letsencrypt/live/api.example.com/fullchain.pem"
  tls_key: "/etc/letsencrypt/live/api.example.com/privkey.pem"
```

## Configuration Problems

### Environment Variables Not Loading

**Symptoms:**
- Configuration shows default values instead of environment variables
- Secrets not being substituted

**Diagnosis:**

```bash
# Check if environment variables are set
echo $LLM_PROXY_SERVER_PORT
echo $LLM_PROXY_AUTH_SECRET_KEY

# List all LLM_PROXY_ variables
env | grep LLM_PROXY

# View effective config
./llm-proxy config --dump
```

**Solutions:**

**Using .env file:**
```bash
# Create .env file
cat > .env << EOF
LLM_PROXY_SERVER_PORT=8080
LLM_PROXY_AUTH_SECRET_KEY=your-secret
OPENAI_API_KEY=sk-...
EOF

# Load in shell
export $(cat .env | xargs)

# Or use direnv
echo "export \$(cat .env | xargs)" >> ~/.bashrc
```

**Using systemd:**
```bash
# Edit service environment
sudo systemctl edit llm-proxy

# Add:
[Service]
Environment="LLM_PROXY_SERVER_PORT=8080"
Environment="LLM_PROXY_AUTH_SECRET_KEY=your-secret"
Environment="OPENAI_API_KEY=sk-..."

# Reload and restart
sudo systemctl daemon-reload
sudo systemctl restart llm-proxy
```

**Using Docker:**
```bash
# Pass environment variables
docker run -d \
  --name llm-proxy \
  -e LLM_PROXY_SERVER_PORT=8080 \
  -e LLM_PROXY_AUTH_SECRET_KEY=your-secret \
  -e OPENAI_API_KEY=sk-... \
  go-llm-proxy:latest

# Or use env file
docker run -d \
  --name llm-proxy \
  --env-file .env \
  go-llm-proxy:latest
```

### Provider Configuration Issues

**Symptoms:**
- Provider connection failures
- "401 Unauthorized" errors
- "Provider not found" errors

**Diagnosis:**

```bash
# Test provider connection
curl -H "Authorization: Bearer sk-..." \
  https://api.openai.com/v1/models

# Check provider configuration
./llm-proxy config --dump | grep -A 20 providers

# Validate API key
curl -I -H "Authorization: Bearer YOUR_KEY" \
  https://api.openai.com/v1/models
```

**Solutions:**

**Invalid API key:**
```yaml
# Verify API key format
providers:
  - name: "openai"
    type: "openai"
    api_key: "sk-..."  # Must start with "sk-"
    base_url: "https://api.openai.com/v1"
```

**Wrong base URL:**
```yaml
# OpenAI
base_url: "https://api.openai.com/v1"  # Not "https://api.openai.com/"

# Anthropic
base_url: "https://api.anthropic.com"  # Not "https://api.anthropic.com/v1"
```

**Provider disabled:**
```yaml
providers:
  - name: "openai"
    type: "openai"
    api_key: "sk-..."
    enabled: true  # Must be true to use provider
```

### Session Store Configuration

**Symptoms:**
- Sessions not persisting
- "session not found" errors
- Frequent logins required

**Diagnosis:**

```bash
# Check session directory permissions
ls -la /app/data/sessions

# Check Redis connectivity (if using Redis)
redis-cli ping

# View session store configuration
./llm-proxy config --dump | grep -A 10 auth
```

**Solutions:**

**Filesystem store permissions:**
```bash
# Create session directory
mkdir -p /app/data/sessions

# Set ownership
sudo chown -R llm-proxy:llm-proxy /app/data

# Set permissions
sudo chmod 700 /app/data/sessions
```

**Redis store connection:**
```yaml
auth:
  session_store: "redis"
  redis_url: "redis://:password@redis-host:6379/0"
  redis_key_prefix: "llm-proxy:session:"

# Test Redis connection
redis-cli -u redis://:password@redis-host:6379 ping
```

## Authentication Issues

### Cannot Login as Admin

**Symptoms:**
- "Invalid credentials" error
- 401 Unauthorized responses
- Login form not working

**Diagnosis:**

```bash
# Check if default admin exists
cat /app/data/users.json

# Check audit logs
tail -100 /var/log/llm-proxy-audit.log | grep login

# Verify session configuration
./llm-proxy config --dump | grep -A 5 auth
```

**Solutions:**

**Create default admin user:**
```bash
# Generate password hash
./llm-proxy hash-password --password "your-password"

# Create admin user
./llm-proxy create-admin \
  --username admin \
  --password-hash "YOUR_HASH_HERE" \
  --config config/config.yaml

# Or use environment variable for initial admin
export LLM_PROXY_ADMIN_USERNAME=admin
export LLM_PROXY_ADMIN_PASSWORD=your-password
```

**Reset admin password:**
```bash
# Option 1: Delete user file and restart (creates default)
rm /app/data/users.json
sudo systemctl restart llm-proxy

# Option 2: Update via API (if you can log in)
curl -X PUT http://localhost:8080/admin/api/v1/users/admin \
  -H "Content-Type: application/json" \
  -d '{"password": "new-password"}'
```

**Fix session timeout:**
```yaml
auth:
  session_timeout: 24h  # Set appropriate timeout
  secret_key: "32-character-secret-key-minimum"  # Must be at least 32 chars
```

### Session Expires Too Quickly

**Symptoms:**
- Frequent logouts
- Session timeout errors
- "Session expired" message

**Diagnosis:**

```bash
# Check current session timeout
./llm-proxy config --dump | grep session_timeout

# Check cookie configuration
curl -I -X POST http://localhost:8080/admin/api/v1/auth/login \
  -d '{"username":"admin","password":"..."}'
```

**Solutions:**

```yaml
auth:
  session_timeout: 24h  # Adjust to desired duration
  cookie_name: "session"  # Default
  cookie_secure: true  # Set to true in production
  cookie_http_only: true  # Recommended
  cookie_same_site: "lax"  # Or "strict" or "none"
```

### CSRF Token Issues

**Symptoms:**
- "CSRF token invalid" error
- 403 Forbidden on form submissions
- Login/logout failures

**Solutions:**

**Ensure CSRF is properly configured:**
```yaml
security:
  enable_csrf: true
  csrf_secret: "${CSRF_SECRET}"  # Required if CSRF enabled
```

**For development, you can disable CSRF:**
```yaml
security:
  enable_csrf: false  # Development only!
```

## Provider Connection Problems

### Timeouts and Slow Responses

**Symptoms:**
- Requests taking >30 seconds
- "timeout" errors
- Provider health checks failing

**Diagnosis:**

```bash
# Test provider response time
time curl -H "Authorization: Bearer sk-..." \
  https://api.openai.com/v1/models

# Check provider health
curl http://localhost:8080/admin/api/v1/providers/status

# View detailed health
curl http://localhost:8080/healthz/detailed
```

**Solutions:**

**Increase timeout:**
```yaml
proxy:
  timeout: 60  # Increase from default 30s
  request_timeout: 120  # For long-running requests

providers:
  - name: "openai"
    type: "openai"
    timeout: 60  # Provider-specific timeout
```

**Enable connection pooling:**
```yaml
performance:
  connection_pool_size: 100
  max_idle_connections: 100
  idle_connection_timeout: 90s
```

**Check provider status:**
```bash
# Pause unhealthy provider
curl -X POST http://localhost:8080/admin/api/v1/providers/{id}/toggle

# Or adjust priority
curl -X POST http://localhost:8080/admin/api/v1/providers/{id}/priority \
  -H "Content-Type: application/json" \
  -d '{"priority": 10}'
```

### Rate Limiting

**Symptoms:**
- "Rate limit exceeded" error
- 429 status codes
- Inconsistent failures

**Diagnosis:**

```bash
# Check rate limit headers
curl -I https://api.openai.com/v1/models \
  -H "Authorization: Bearer sk-..."

# View rate limit metrics
curl http://localhost:8080/metrics | grep rate_limit
```

**Solutions:**

**Configure rate limits per provider:**
```yaml
providers:
  - name: "openai"
    type: "openai"
    rate_limit:
      requests_per_minute: 1000
      burst: 100
```

**Use multiple providers:**
```yaml
providers:
  - name: "openai-us-east"
    type: "openai"
    priority: 1
    weight: 0.5  # Distribute load

  - name: "openai-eu-west"
    type: "openai"
    priority: 2
    weight: 0.5
```

### API Key Issues

**Symptoms:**
- 401 Unauthorized
- "Invalid API key" errors
- Authentication failures

**Diagnosis:**

```bash
# Test API key directly
curl -H "Authorization: Bearer YOUR_KEY" \
  https://api.openai.com/v1/models

# Verify key format
echo $OPENAI_API_KEY | head -c 10
```

**Solutions:**

**Check API key environment variable:**
```bash
# Set in shell
export OPENAI_API_KEY="sk-..."

# Set in .env file
echo "OPENAI_API_KEY=sk-..." >> .env

# Set in systemd
sudo systemctl edit llm-proxy
# Add Environment="OPENAI_API_KEY=sk-..."

# Set in Docker
docker run -e OPENAI_API_KEY="sk-..." ...
```

**Verify API key is correct:**
```bash
# For OpenAI
curl -H "Authorization: Bearer $OPENAI_API_KEY" \
  https://api.openai.com/v1/models

# For Anthropic
curl -H "x-api-key: $ANTHROPIC_API_KEY" \
  https://api.anthropic.com/v1/messages
```

## Performance Issues

### High Latency

**Symptoms:**
- Requests taking >2 seconds
- Users complaining of slow responses
- Timeout errors

**Diagnosis:**

```bash
# Check response times in metrics
curl http://localhost:8080/metrics | grep request_duration

# Enable pprof
curl http://localhost:8080/debug/pprof/profile?seconds=30

# Check provider latency
curl http://localhost:8080/admin/api/v1/providers/status
```

**Solutions:**

**Enable caching:**
```yaml
performance:
  cache_enabled: true
  cache_ttl: 300  # 5 minutes
  cache_size: 1000
```

**Enable compression:**
```yaml
performance:
  enable_compression: true
  compression_level: 6
  min_compress_size: 1024
```

**Connection pooling:**
```yaml
performance:
  connection_pool_size: 100
  max_idle_connections: 100
  keep_alive: 30s
```

**Check provider health:**
```bash
# Disable unhealthy providers
curl -X POST http://localhost:8080/admin/api/v1/providers/{id}/toggle

# Adjust provider priority
curl -X POST http://localhost:8080/admin/api/v1/providers/{id}/priority \
  -d '{"priority": 10}'
```

### Low Throughput

**Symptoms:**
- Cannot handle high request volume
- Queue buildup
- 503 Service Unavailable

**Diagnosis:**

```bash
# Check RPS
curl http://localhost:8080/metrics | grep requests_total

# Monitor connections
netstat -an | grep :8080 | wc -l

# Check resource usage
top -p $(pgrep llm-proxy)
```

**Solutions:**

**Scale horizontally:**
```yaml
# Kubernetes
kubectl scale deployment llm-proxy --replicas=5

# systemd - run multiple instances
sudo systemctl start llm-proxy@8080
sudo systemctl start llm-proxy@8081

# Docker Compose
docker-compose up -d --scale llm-proxy=3
```

**Optimize configuration:**
```yaml
server:
  max_header_bytes: 1048576  # 1MB
  max_connections: 1000

performance:
  connection_pool_size: 200
  max_connections_per_host: 200
```

### Cache Not Working

**Symptoms:**
- High cache miss rate
- Repeated identical requests
- Cache size not increasing

**Diagnosis:**

```bash
# Check cache metrics
curl http://localhost:8080/metrics | grep cache

# View cache configuration
./llm-proxy config --dump | grep -A 10 cache

# Enable cache debug logging
curl -X POST http://localhost:8080/admin/api/v1/debug/cache \
  -H "Authorization: Bearer admin-token"
```

**Solutions:**

**Enable cache:**
```yaml
performance:
  cache_enabled: true
  cache_ttl: 300
  cache_size: 1000
```

**Increase cache size:**
```yaml
performance:
  cache_size: 5000  # Increase from default 1000
  cache_ttl: 600  # 10 minutes
```

**Check cacheable endpoints:**
```yaml
performance:
  cache:
    GET:
      enabled: true  # Cache GET requests
      ttl: 600
    POST:
      enabled: false  # Don't cache POST by default
    exclude_paths:
      - "/admin"  # Don't cache admin endpoints
      - "/healthz"
```

## High Availability Issues

### Failover Not Working

**Symptoms:**
- Service unavailable when primary provider fails
- 503 errors instead of failover
- Requests not switching to backup provider

**Diagnosis:**

```bash
# Check provider health
curl http://localhost:8080/admin/api/v1/providers/status

# Enable debug logging for load balancer
export LLM_PROXY_LOGGING_LEVEL=debug
```

**Solutions:**

**Configure multiple providers:**
```yaml
providers:
  - name: "openai-primary"
    type: "openai"
    priority: 1
    enabled: true
    weight: 1.0

  - name: "openai-backup"
    type: "openai"
    priority: 2
    enabled: true
    weight: 1.0
```

**Adjust health check interval:**
```yaml
proxy:
  health_check_interval: 15s  # Check every 15s (default: 30s)
  health_check_timeout: 5s
```

**Test failover:**
```bash
# Manually disable primary
curl -X POST http://localhost:8080/admin/api/v1/providers/{id}/toggle

# Verify requests go to backup
curl http://localhost:8080/metrics | grep provider_requests
```

### Load Balancing Issues

**Symptoms:**
- Uneven distribution of requests
- One provider receiving all traffic
- Load not balancing

**Diagnosis:**

```bash
# Check request distribution
curl http://localhost:8080/metrics | grep provider_requests

# View load balancer stats
curl http://localhost:8080/admin/api/v1/providers/status
```

**Solutions:**

**Configure provider weights:**
```yaml
providers:
  - name: "openai-us"
    type: "openai"
    priority: 1
    weight: 0.7  # 70% of traffic

  - name: "openai-eu"
    type: "openai"
    priority: 2
    weight: 0.3  # 30% of traffic
```

**Check provider status:**
```bash
# Ensure all providers are enabled
curl http://localhost:8080/admin/api/v1/providers

# Adjust priorities
curl -X POST http://localhost:8080/admin/api/v1/providers/{id}/priority \
  -d '{"priority": 1}'
```

## Memory and CPU Issues

### High Memory Usage

**Symptoms:**
- Memory usage >100MB
- Out of memory errors
- System running slow

**Diagnosis:**

```bash
# Check memory usage
ps aux | grep llm-proxy
top -p $(pgrep llm-proxy)

# Generate heap profile
curl http://localhost:8080/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# Check for memory leaks
curl http://localhost:8080/debug/pprof/heap > heap1.prof
# Wait 10 minutes
sleep 600
curl http://localhost:8080/debug/pprof/heap > heap2.prof
go tool pprof -base heap1.prof heap2.prof
```

**Solutions:**

**Enable object pooling:**
```yaml
performance:
  enable_object_pooling: true
  object_pool_size: 100
```

**Limit cache memory:**
```yaml
performance:
  cache_enabled: true
  cache_size: 500  # Reduce from 1000
  cache_ttl: 300
```

**Reduce connection pool:**
```yaml
performance:
  connection_pool_size: 50  # Reduce from 100
  max_idle_connections: 50
```

**Enable GC optimizations:**
```yaml
# Set GOGC environment variable
export GOGC=100  # Default is 100, lower = more aggressive GC
```

### High CPU Usage

**Symptoms:**
- CPU usage >80%
- System unresponsive
- Slow request processing

**Diagnosis:**

```bash
# Check CPU usage
top -p $(pgrep llm-proxy)

# Generate CPU profile
curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof

# Check for hot spots
go tool pprof -text cpu.prof
```

**Solutions:**

**Optimize request processing:**
```yaml
performance:
  enable_compression: true  # CPU tradeoff for bandwidth
  cache_enabled: true  # Cache to reduce processing
```

**Reduce log verbosity:**
```yaml
logging:
  level: "info"  # Not "debug" in production
  format: "json"  # Faster than text
```

**Limit concurrent requests:**
```yaml
server:
  max_connections: 500  # Limit concurrent connections

proxy:
  max_retries: 2  # Reduce retries
```

## Network and Connectivity

### Connection Refused

**Symptoms:**
- "Connection refused" errors
- Cannot reach service
- 503 Service Unavailable

**Diagnosis:**

```bash
# Check if service is running
ps aux | grep llm-proxy

# Check port is listening
netstat -tulpn | grep 8080
ss -tulpn | grep 8080

# Test local connection
curl -v http://localhost:8080/healthz
```

**Solutions:**

**Start service:**
```bash
# systemd
sudo systemctl start llm-proxy
sudo systemctl status llm-proxy

# Docker
docker start llm-proxy
docker ps

# Direct binary
./llm-proxy serve --config config/config.yaml
```

**Check firewall:**
```bash
# UFW (Ubuntu)
sudo ufw status
sudo ufw allow 8080/tcp

# iptables
sudo iptables -L -n
sudo iptables -A INPUT -p tcp --dport 8080 -j ACCEPT

# firewalld (CentOS/RHEL)
sudo firewall-cmd --list-all
sudo firewall-cmd --add-port=8080/tcp --permanent
sudo firewall-cmd --reload
```

### Proxy/Behind Load Balancer

**Symptoms:**
- Wrong client IP in logs
- X-Forwarded-* headers missing
- Rate limiting per load balancer IP

**Diagnosis:**

```bash
# Check X-Forwarded-For header
curl -H "X-Forwarded-For: 1.2.3.4" http://localhost:8080/healthz

# View access logs
tail -f /var/log/llm-proxy/access.log
```

**Solutions:**

**Configure trusted proxies:**
```yaml
server:
  trusted_proxies:
    - "10.0.0.0/8"
    - "172.16.0.0/12"
    - "192.168.0.0/16"
    - "100.64.0.0/10"  # CGNAT
    - "127.0.0.1"  # Localhost
```

**Set proxy headers in load balancer:**

**Nginx:**
```nginx
location / {
    proxy_pass http://llm_proxy;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

**HAProxy:**
```yaml
backend llm_proxy
    option forwardfor
    http-request set-header X-Forwarded-Proto https if { ssl_fc }
```

## Docker and Container Issues

### Container Won't Start

**Symptoms:**
- Container exits immediately
- "Exited (1)" status
- Port binding failures

**Diagnosis:**

```bash
# Check container logs
docker logs llm-proxy

# Inspect container
docker inspect llm-proxy

# Run interactive for debugging
docker run -it --rm go-llm-proxy:latest /bin/sh
```

**Solutions:**

**Bind to 0.0.0.0 not localhost:**
```yaml
# docker-compose.yml
services:
  llm-proxy:
    ports:
      - "0.0.0.0:8080:8080"  # Not "127.0.0.1:8080:8080"
      - "0.0.0.0:8443:8443"
```

**Set appropriate resource limits:**
```yaml
services:
  llm-proxy:
    deploy:
      resources:
        limits:
          cpus: '1'
          memory: 1G
        reservations:
          cpus: '0.5'
          memory: 512M
```

**Check volume mounts:**
```bash
# Ensure directories exist
mkdir -p config data logs

# Verify permissions
chmod 644 config/*
chmod 755 data logs
```

### Container Crashes

**Symptoms:**
- Frequent restarts
- OOMKilled
- Segmentation fault

**Diagnosis:**

```bash
# View crash logs
docker logs --tail 100 llm-proxy

# Check resource usage
docker stats llm-proxy

# Enable debug mode
docker run -e LLM_PROXY_LOGGING_LEVEL=debug ...
```

**Solutions:**

**Increase memory limit:**
```yaml
services:
  llm-proxy:
    deploy:
      resources:
        limits:
          memory: 1G  # Increase from default
```

**Set GOGC:**
```yaml
services:
  llm-proxy:
    environment:
      - GOGC=100
    # Or for Alpine
    # Use the /healthz endpoint to keep container alive
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/healthz"]
      interval: 30s
      timeout: 10s
      retries: 3
```

## Kubernetes Issues

### Pod Won't Start

**Symptoms:**
- Pod in Pending state
- CrashLoopBackOff
- ImagePullBackOff

**Diagnosis:**

```bash
# Check pod status
kubectl get pods -n llm-proxy

# Describe pod
kubectl describe pod <pod-name> -n llm-proxy

# Check events
kubectl get events -n llm-proxy --sort-by='.lastTimestamp'
```

**Solutions:**

**Image pull error:**
```bash
# Check image exists
docker pull go-llm-proxy:latest

# Update image in deployment
kubectl set image deployment/llm-proxy \
  llm-proxy=your-registry/go-llm-proxy:v1.0.0 \
  -n llm-proxy
```

**Resource limits:**
```yaml
# pod.yaml
resources:
  limits:
    cpu: "1"
    memory: "1Gi"
  requests:
    cpu: "500m"
    memory: "512Mi"
```

**Fix configuration mount:**
```yaml
# deployment.yaml
volumeMounts:
  - name: config
    mountPath: /app/config
    readOnly: true
volumes:
  - name: config
    configMap:
      name: llm-proxy-config
```

### Service Not Accessible

**Symptoms:**
- Service IP not responding
- Connection timeout
- Endpoints showing 0/0

**Diagnosis:**

```bash
# Check service
kubectl get svc -n llm-proxy

# Check endpoints
kubectl get endpoints -n llm-proxy

# Port-forward for testing
kubectl port-forward svc/llm-proxy 8080:80 -n llm-proxy
```

**Solutions:**

**Fix service selector:**
```yaml
# service.yaml
selector:
  app: llm-proxy  # Must match pod labels

# deployment.yaml
labels:
  app: llm-proxy  # Pod labels
```

**Expose service properly:**
```yaml
# service.yaml
apiVersion: v1
kind: Service
metadata:
  name: llm-proxy
spec:
  type: LoadBalancer  # Or NodePort, ClusterIP
  ports:
    - port: 80
      targetPort: 8080
  selector:
    app: llm-proxy
```

### Horizontal Pod Autoscaler Not Working

**Symptoms:**
- High load but no new pods
- HPA reporting errors
- CPU/memory metrics not available

**Diagnosis:**

```bash
# Check HPA status
kubectl get hpa -n llm-proxy

# Describe HPA
kubectl describe hpa llm-proxy -n llm-proxy

# Check metrics server
kubectl get pods -n kube-system | grep metrics-server
```

**Solutions:**

**Install metrics server:**
```bash
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
```

**Configure HPA:**
```yaml
# hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: llm-proxy-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: llm-proxy
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

## Monitoring and Metrics

### Metrics Not Appearing

**Symptoms:**
- /metrics endpoint returns no data
- Grafana shows no data
- Prometheus target down

**Diagnosis:**

```bash
# Test metrics endpoint
curl http://localhost:8080/metrics

# Check metrics configuration
./llm-proxy config --dump | grep -A 10 metrics

# Verify Prometheus is scraping
curl http://localhost:9090/targets
```

**Solutions:**

**Enable metrics:**
```yaml
metrics:
  enabled: true
  port: 9090
  path: "/metrics"
```

**Check Prometheus configuration:**
```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'llm-proxy'
    static_configs:
      - targets: ['llm-proxy:9090']
    metrics_path: /metrics
    scrape_interval: 30s
```

**Test Prometheus target:**
```bash
# Reload Prometheus config
curl -X POST http://localhost:9090/-/reload

# Check targets
curl http://localhost:9090/api/v1/targets
```

### Health Checks Failing

**Symptoms:**
- Readiness check failing
- Service marked unready
- Traffic not routed to pod

**Diagnosis:**

```bash
# Test health checks
curl http://localhost:8080/healthz
curl http://localhost:8080/healthz/ready
curl http://localhost:8080/healthz/detailed

# Check provider health
curl http://localhost:8080/admin/api/v1/providers/status
```

**Solutions:**

**Fix provider configuration:**
```bash
# Ensure at least one provider is enabled
# and has valid API key

# Test provider connection
curl -H "Authorization: Bearer sk-..." \
  https://api.openai.com/v1/models
```

**Adjust health check configuration:**
```yaml
# readiness probe
readinessProbe:
  httpGet:
    path: /healthz/ready
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 10

# liveness probe
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 30
```

## Logging and Debugging

### Logs Not Appearing

**Symptoms:**
- No logs in console
- Log file not created
- Audit logs missing

**Diagnosis:**

```bash
# Check log level
./llm-proxy config --dump | grep level

# Verify log file path
ls -la /var/log/llm-proxy/

# Check log rotation
logrotate -d /etc/logrotate.d/llm-proxy
```

**Solutions:**

**Set appropriate log level:**
```yaml
logging:
  level: "info"  # debug, info, warn, error
  format: "json"  # or text
  output: "stdout"  # or stderr, file
```

**Configure log file:**
```yaml
logging:
  level: "info"
  file: "/var/log/llm-proxy/app.log"
  max_size: 100  # MB
  max_backups: 10
  max_age: 28
  compress: true
```

**Check permissions:**
```bash
# Create log directory
sudo mkdir -p /var/log/llm-proxy

# Set ownership
sudo chown llm-proxy:llm-proxy /var/log/llm-proxy

# Set permissions
sudo chmod 755 /var/log/llm-proxy
```

### Too Much Debug Output

**Symptoms:**
- Log files growing rapidly
- Disk space issues
- Performance degradation

**Solutions:**

**Reduce log level:**
```yaml
logging:
  level: "info"  # Change from debug to info
  format: "json"  # More compact than text
```

**Enable log rotation:**
```bash
# Create logrotate config
sudo cat > /etc/logrotate.d/llm-proxy << EOF
/var/log/llm-proxy/*.log {
    daily
    rotate 10
    size 100M
    compress
    delaycompress
    missingok
    notifempty
    create 644 llm-proxy llm-proxy
    postrotate
        systemctl reload llm-proxy
    endscript
}
EOF

# Test logrotate
sudo logrotate -d /etc/logrotate.d/llm-proxy
```

## Migration Issues

### Python to Go Migration

**Symptoms:**
- New Go version missing features
- Configuration incompatibility
- Performance regression

**Diagnosis:**

```bash
# Compare feature parity
./llm-proxy version
# vs Python version

# Check configuration
./llm-proxy config --dump
```

**Solutions:**

**Review migration guide:**
```bash
# See docs/migration.md for detailed steps

# Compare configurations
# Python: config.json
# Go: config.yaml
```

**Test feature parity:**
```bash
# Run regression tests
go test ./tests/e2e/... -run TestRegression

# Benchmark performance
go test ./tests/benchmark/... -bench=.
```

### Version Upgrade Issues

**Symptoms:**
- Configuration incompatible
- Feature not working after upgrade
- Breaking changes

**Solutions:**

**Review changelog:**
```bash
# Check CHANGELOG.md
cat CHANGELOG.md

# Review breaking changes
git log --grep="BREAKING" --oneline
```

**Migrate configuration:**
```bash
# Generate new config template
./llm-proxy config --sample > config/config.yaml.new

# Compare with existing
diff -u config/config.yaml config/config.yaml.new

# Update configuration accordingly
```

## Error Reference

### Common Error Codes

| Code | Message | Cause | Solution |
|------|---------|-------|----------|
| 400 | invalid_request_error | Invalid request parameters | Check request format |
| 401 | authentication_error | Invalid/missing API key | Verify API key |
| 403 | permission_denied | Insufficient permissions | Check admin auth |
| 404 | not_found | Resource doesn't exist | Verify ID/path |
| 429 | rate_limit_error | Too many requests | Implement backoff |
| 500 | server_error | Internal server error | Check logs |
| 503 | service_unavailable | No healthy providers | Check provider status |

### HTTP Status Codes

| Code | Meaning | Common Causes |
|------|---------|---------------|
| 200 | OK | Request successful |
| 201 | Created | Resource created |
| 204 | No Content | Success with no response |
| 400 | Bad Request | Invalid parameters |
| 401 | Unauthorized | Missing/invalid auth |
| 403 | Forbidden | Access denied |
| 404 | Not Found | Resource not found |
| 429 | Too Many Requests | Rate limit exceeded |
| 500 | Internal Server Error | Server-side error |
| 502 | Bad Gateway | Provider error |
| 503 | Service Unavailable | No healthy providers |
| 504 | Gateway Timeout | Provider timeout |

### Error Message Patterns

**"connection refused"**
- Service not running
- Wrong port
- Firewall blocking

**"timeout"**
- Provider slow/unresponsive
- Network issues
- Increase timeout config

**"unauthorized"**
- Invalid API key
- Missing credentials
- Session expired

**"rate limit exceeded"**
- Too many requests
- Implement exponential backoff
- Use multiple providers

**"no healthy providers"**
- All providers down
- All providers disabled
- Fix provider configuration

## Getting More Help

### Collect Information for Support

Before reporting an issue, gather this information:

```bash
# 1. Version information
./llm-proxy version

# 2. Configuration
./llm-proxy config --dump > config-dump.yaml

# 3. Health status
curl http://localhost:8080/healthz/detailed > health.json

# 4. Metrics
curl http://localhost:8080/metrics > metrics.txt

# 5. Recent logs
tail -100 /var/log/llm-proxy/app.log > recent-logs.txt

# 6. System info
uname -a
go version
```

### Support Channels

- **GitHub Issues**: https://github.com/fcmfcm01/go-llm-proxy/issues
- **GitHub Discussions**: https://github.com/fcmfcm01/go-llm-proxy/discussions
- **Email**: support@yourcompany.com
- **Documentation**: https://github.com/fcmfcm01/go-llm-proxy/docs

### Community Resources

- **Go Documentation**: https://golang.org/doc/
- **Gin Framework**: https://gin-gonic.com/docs/
- **Prometheus**: https://prometheus.io/docs/
- **Docker**: https://docs.docker.com/
- **Kubernetes**: https://kubernetes.io/docs/

## Prevention Best Practices

### Configuration Management

1. **Version control**: Keep config in Git
2. **Templates**: Use config templates for different environments
3. **Validation**: Always run `validate-config` before deploying
4. **Documentation**: Document all custom configurations
5. **Environment variables**: Use for secrets and environment-specific values

### Monitoring

1. **Health checks**: Monitor `/healthz` and `/healthz/ready`
2. **Metrics**: Set up Prometheus scraping
3. **Alerting**: Configure alerts for error rates, latency
4. **Logging**: Centralize logs (ELK, Splunk, etc.)
5. **Tracing**: Add distributed tracing for complex issues

### Operations

1. **Blue-green deployments**: For zero-downtime updates
2. **Canary releases**: Test new versions gradually
3. **Backup configuration**: Regular config backups
4. **Documentation**: Keep runbooks updated
5. **Training**: Ensure team knows troubleshooting procedures

### Performance

1. **Load testing**: Regular load tests
2. **Capacity planning**: Monitor growth trends
3. **Caching**: Enable appropriate caching
4. **Connection pooling**: Configure connection pools
5. **Rate limiting**: Prevent abuse

### Security

1. **Regular updates**: Keep dependencies updated
2. **Security scans**: Run gosec regularly
3. **Certificate monitoring**: Track certificate expiration
4. **Secret rotation**: Regular API key rotation
5. **Access control**: Limit admin access

---

**Note**: This guide covers the most common issues. For issues not covered here, please check the logs, enable debug logging, and refer to the documentation or seek support.
