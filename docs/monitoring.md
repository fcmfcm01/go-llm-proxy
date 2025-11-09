# LLM Proxy Monitoring Guide

## Overview

The LLM Proxy provides comprehensive monitoring capabilities through Prometheus metrics, health check endpoints, and real-time status tracking. This guide covers how to monitor, alert on, and visualize system health and performance.

## Health Check Endpoints

The proxy exposes multiple health check endpoints for different purposes:

### 1. Basic Health Check - `/healthz`

**Purpose**: Basic liveness check to verify the service is running

**Endpoint**: `GET /healthz`

**Response**:
```json
{
  "status": "healthy",
  "uptime": "2h30m45s",
  "version": "1.0.0",
  "runtime": "release"
}
```

**Use Case**: Kubernetes liveness probe, basic uptime monitoring

### 2. Liveness Probe - `/healthz/live`

**Purpose**: Simple check that the process is alive (no dependencies checked)

**Endpoint**: `GET /healthz/live`

**Response**:
```json
{
  "status": "alive"
}
```

**Use Case**: Kubernetes liveness probe (faster than `/healthz`)

### 3. Readiness Probe - `/healthz/ready`

**Purpose**: Check if the service is ready to receive traffic (all dependencies available)

**Endpoint**: `GET /healthz/ready`

**Response** (Ready):
```json
{
  "status": "ready",
  "checks": {
    "config": {
      "status": "ok",
      "message": "Configuration loaded",
      "latency_ms": 2
    },
    "providers": {
      "status": "ok",
      "message": "Providers available",
      "latency_ms": 15
    },
    "dependencies": {
      "status": "ok",
      "message": "Dependencies available",
      "latency_ms": 5
    }
  },
  "latency": 22,
  "message": "All components are ready",
  "checksum": "ok"
}
```

**Response** (Not Ready):
```json
{
  "status": "not ready",
  "checks": {
    "config": {
      "status": "ok",
      "message": "Configuration loaded",
      "latency_ms": 2
    },
    "providers": {
      "status": "warning",
      "message": "No providers configured",
      "latency_ms": 5
    }
  },
  "latency": 7,
  "message": "One or more components are not ready",
  "checksum": "ok"
}
```

**Use Case**: Kubernetes readiness probe, load balancer health checks

### 4. Detailed Health Check - `/healthz/detailed`

**Purpose**: Comprehensive system information for diagnostics

**Endpoint**: `GET /healthz/detailed`

**Response**:
```json
{
  "status": "healthy",
  "timestamp": "2025-11-09T12:00:00Z",
  "uptime": "2h30m45s",
  "version": "1.0.0",
  "system": {
    "hostname": "localhost",
    "go_version": "go1.21.5",
    "go_os": "linux",
    "go_arch": "amd64",
    "num_cpu": 4,
    "build_info": {
      "git_commit": "abc123",
      "build_time": "2025-11-09T10:00:00Z",
      "compiler": "gc"
    }
  },
  "runtime": {
    "goroutines": 12,
    "memory": {
      "Alloc": 1048576,
      "Sys": 5242880,
      "HeapAlloc": 1048576,
      "HeapSys": 4194304
    }
  },
  "components": {
    "server": {
      "status": "ok",
      "message": "Server is running"
    },
    "proxy": {
      "status": "ok",
      "message": "Proxy is operational"
    },
    "config": {
      "status": "ok",
      "message": "Configuration loaded"
    },
    "metrics": {
      "status": "ok",
      "message": "Metrics collection active"
    },
    "health_checks": {
      "status": "ok",
      "message": "Health checks operational"
    }
  }
}
```

**Use Case**: System diagnostics, manual troubleshooting, detailed monitoring

## Prometheus Metrics

### Metrics Endpoint - `/metrics`

**Endpoint**: `GET /metrics`

**Format**: Prometheus text format

**Purpose**: Expose all metrics for Prometheus scraping

### Metrics Config Endpoint - `/metrics/config`

**Endpoint**: `GET /metrics/config`

**Purpose**: Get metadata about available metrics

**Response**:
```json
{
  "metrics": {
    "llm_proxy_requests_total": {
      "type": "counter",
      "help": "Total number of requests processed"
    }
  }
}
```

## Available Metrics

### Request Metrics

- `llm_proxy_requests_total` - Total number of requests processed
- `llm_proxy_errors_total` - Total number of errors
- `llm_proxy_request_duration_seconds` - Duration of requests in seconds
- `llm_proxy_requests_in_flight` - Current number of requests being processed

### Provider Metrics

- `llm_proxy_provider_requests_total` - Total requests to each provider (labels: `provider`, `status_code`)
- `llm_proxy_provider_request_duration_seconds` - Request duration by provider (labels: `provider`)
- `llm_proxy_provider_errors_total` - Total errors by provider (labels: `provider`, `error_type`)
- `llm_proxy_provider_success_rate` - Success rate by provider (0-100) (labels: `provider`)

### Response Metrics

- `llm_proxy_response_size_bytes` - Size of responses in bytes
- `llm_proxy_prompt_tokens` - Number of prompt tokens
- `llm_proxy_completion_tokens` - Number of completion tokens
- `llm_proxy_total_tokens` - Total number of tokens (prompt + completion)

### Business Metrics

- `llm_proxy_active_users` - Number of active users
- `llm_proxy_load_balanced_requests_total` - Total number of load balanced requests
- `llm_proxy_failed_over_requests_total` - Total number of requests that failed over
- `llm_proxy_retried_requests_total` - Total number of retried requests

### Health Metrics

- `llm_proxy_health_healthy_providers` - Number of healthy providers
- `llm_proxy_health_unhealthy_providers` - Number of unhealthy providers
- `llm_proxy_health_checks_total` - Total health checks performed (labels: `provider`, `result`)
- `llm_proxy_health_response_time_seconds` - Provider response times for health checks (labels: `provider`)

## Grafana Dashboard

A pre-configured Grafana dashboard is available at `deployment/grafana/dashboards/llm-proxy.json`.

### Dashboard Panels

1. **Request Rate** - Requests per second (last 5 minutes)
2. **Error Rate** - Errors and total requests per second
3. **Request Duration (p95)** - 95th percentile latency
4. **Requests In Flight** - Current concurrent requests
5. **Provider Request Distribution** - Requests by provider
6. **Provider Health Status** - Count of healthy/unhealthy providers
7. **Response Size Distribution** - Average response size
8. **Token Usage** - Prompt, completion, and total tokens per second
9. **Load Balancing Metrics** - Load balanced, failed over, and retried requests
10. **Provider Response Times (p95)** - Heatmap of provider latencies
11. **Health Check Response Times** - Provider health check latencies
12. **Active Users** - Current number of active users

### Importing the Dashboard

1. Open Grafana in your browser
2. Navigate to Dashboards → Import
3. Upload or paste the `deployment/grafana/dashboards/llm-proxy.json` file
4. Select your Prometheus data source
5. Click Import

## Prometheus Alerting

Alert rules are configured in `deployment/prometheus/alerts.yml`.

### Included Alerts

#### Warning Alerts

- **HighErrorRate** - Error rate > 0.1 errors/sec for 2 minutes
- **HighRequestLatency** - p95 latency > 5 seconds for 5 minutes
- **HighRequestsInFlight** - In-flight requests > 100 for 5 minutes
- **LowHealthyProviders** - Healthy providers < 2 for 3 minutes
- **HighFailoverRate** - Failover rate > 0.1 requests/sec
- **HighRetryRate** - Retry rate > 0.2 requests/sec
- **ProviderHealthCheckFailures** - Health check failures detected
- **HighProviderResponseTime** - Provider p95 response time > 3 seconds
- **VeryHighResponseSize** - Average response size > 10MB

#### Critical Alerts

- **CriticalErrorRate** - Error rate > 0.5 errors/sec for 1 minute
- **CriticalRequestLatency** - p95 latency > 10 seconds for 2 minutes
- **NoHealthyProviders** - All providers unhealthy for 1 minute

#### Info Alerts

- **NoRequests** - No requests received in 10 minutes

#### Infrastructure Alerts

- **InstanceDown** - Proxy instance is down

### Configuring Alertmanager

Add Alertmanager to your Prometheus configuration to receive notifications:

```yaml
alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093
```

## Prometheus Configuration

The Prometheus scrape configuration is in `deployment/prometheus/prometheus.yml`.

### Scrape Targets

- `llm-proxy` - Main metrics endpoint (15s interval)
- `llm-proxy-health` - Health check endpoint (30s interval)
- `llm-proxy-detailed` - Detailed health check (60s interval)
- `prometheus` - Prometheus self-monitoring
- `node-exporter` - System metrics (optional)
- `cadvisor` - Container metrics (optional)

## Kubernetes Monitoring

### Liveness Probe

```yaml
livenessProbe:
  httpGet:
    path: /healthz/live
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 30
```

### Readiness Probe

```yaml
readinessProbe:
  httpGet:
    path: /healthz/ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
```

### Prometheus ServiceMonitor

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: llm-proxy
spec:
  selector:
    matchLabels:
      app: llm-proxy
  endpoints:
    - port: metrics
      path: /metrics
      interval: 15s
    - port: health
      path: /healthz/ready
      interval: 30s
```

## Metric Collection Frequency

- **High-frequency metrics** (15s): Request rate, latency, errors
- **Medium-frequency metrics** (30s): Health checks, provider status
- **Low-frequency metrics** (60s): Detailed health, system info

## Monitoring Best Practices

### 1. Alert Thresholds

Set alert thresholds based on your SLOs:
- Error rate: < 1%
- p95 latency: < 2 seconds
- Success rate: > 99%

### 2. Dashboard Organization

Group related metrics:
- Request metrics (rate, latency, errors)
- Provider metrics (health, performance)
- Business metrics (users, tokens, cost)

### 3. Alert Routing

Configure alert routing by severity:
- Critical → On-call (PagerDuty, SMS)
- Warning → Team channel (Slack, email)
- Info → Ticket system

### 4. Retention

Configure Prometheus retention:
- High-resolution data: 7 days
- Aggregated data: 90 days
- Long-term storage: InfluxDB, Thanos, or Cortex

### 5. Security

- Restrict access to metrics endpoint
- Use TLS for Prometheus communication
- Implement authentication for admin endpoints
- Monitor for anomalous access patterns

## Troubleshooting

### High Error Rate

1. Check provider health: `llm_proxy_health_healthy_providers`
2. Review error types: `llm_proxy_provider_errors_total`
3. Check latency: `llm_proxy_request_duration_seconds`
4. Examine logs for specific error messages

### High Latency

1. Identify slow providers: `llm_proxy_provider_request_duration_seconds`
2. Check network metrics: Response time from providers
3. Review load balancing: `llm_proxy_load_balanced_requests_total`
4. Check resource usage: CPU, memory, connections

### No Requests

1. Verify service is running: `/healthz/live`
2. Check DNS resolution
3. Review proxy configuration
4. Test with curl: `curl http://localhost:8080/healthz`

### Provider Unhealthy

1. Check provider status: `llm_proxy_health_healthy_providers`
2. Review health check latency: `llm_proxy_health_response_time_seconds`
3. Verify provider configuration
4. Test provider connectivity directly

## Performance Baseline

Establish performance baselines for:
- Request rate: 100 RPS (adjust based on your needs)
- Error rate: < 0.1%
- p95 latency: < 1 second
- Memory usage: < 50MB
- CPU usage: < 50%

## Additional Resources

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [Prometheus Operator](https://prometheus-operator.dev/)
- [Alertmanager Documentation](https://prometheus.io/docs/alerting/latest/alertmanager/)

## Support

For issues or questions about monitoring:
1. Check the troubleshooting section above
2. Review logs: Look for error patterns
3. Examine metrics: Use Grafana to visualize trends
4. Consult Prometheus: Query metrics directly
5. Open an issue: Include relevant metrics and logs
