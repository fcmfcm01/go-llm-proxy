# Kubernetes Deployment for LLM Proxy

This directory contains Kubernetes manifests for deploying LLM Proxy in a production Kubernetes cluster.

## Prerequisites

- Kubernetes 1.25+
- Ingress Controller (nginx-ingress or similar)
- cert-manager for TLS certificates
- Prometheus Operator (optional, for monitoring)
- StorageClass configured

## Quick Start

### 1. Apply all manifests

```bash
# Using kubectl
kubectl apply -f k8s/

# Or using kustomize
kubectl apply -k k8s/
```

### 2. Verify deployment

```bash
# Check namespace
kubectl get namespace llm-proxy

# Check pods
kubectl get pods -n llm-proxy

# Check services
kubectl get svc -n llm-proxy

# Check ingress
kubectl get ingress -n llm-proxy
```

### 3. Access the application

- **API Endpoint**: `https://api.llm-proxy.example.com`
- **Admin Interface**: `https://admin.llm-proxy.example.com`
- **Metrics**: `https://metrics.llm-proxy.example.com`

## Manifests Overview

### Core Resources

- `namespace.yaml` - Dedicated namespace for the application
- `configmap.yaml` - Application configuration
- `secret.yaml` - Sensitive data (secrets)
- `rbac.yaml` - Service account and RBAC
- `pvc.yaml` - Persistent volume claims
- `deployment.yaml` - Main application deployment
- `service.yaml` - Service definitions
- `hpa.yaml` - Horizontal Pod Autoscaler
- `pdb.yaml` - Pod Disruption Budget

### Networking

- `ingress.yaml` - Ingress rules for external access
- `network-policy.yaml` - Network security policies

### Monitoring

- `monitoring.yaml` - ServiceMonitor and Prometheus rules

### Utilities

- `kustomization.yaml` - Kustomize configuration

## Configuration

### Environment Variables

Update `configmap.yaml` with your configuration:

```yaml
data:
  config.yaml: |
    server:
      mode: "production"
      port: 8080
    auth:
      session_timeout: 24h
    # ... other config
```

### Secrets

Create secrets with:

```bash
# Generate session secret
kubectl create secret generic llm-proxy-secret \
  --from-literal=session-secret=$(openssl rand -base64 32) \
  -n llm-proxy

# TLS certificates
kubectl create secret tls llm-proxy-tls \
  --cert=tls.crt \
  --key=tls.key \
  -n llm-proxy
```

## Scaling

### Horizontal Pod Autoscaling

The HPA automatically scales pods based on:
- CPU utilization (target: 70%)
- Memory utilization (target: 80%)
- Request rate (target: 1000 RPS)

### Manual scaling

```bash
kubectl scale deployment llm-proxy --replicas=5 -n llm-proxy
```

## Monitoring

### Prometheus Integration

The deployment includes:
- ServiceMonitor for Prometheus scraping
- Custom Prometheus rules for alerting
- Metrics endpoint on port 9090

### Grafana Dashboards

Import the provided Grafana dashboard from `deployment/grafana/dashboards/`

## Security

### Security Contexts

- Non-root user (UID 1000)
- Read-only root filesystem
- Dropped capabilities
- Seccomp profile

### Network Policies

- Restricts ingress to nginx controller
- Restricts egress to DNS and HTTPS
- Prevents lateral movement

## Upgrades

### Rolling Update

```bash
# Update image
kubectl set image deployment/llm-proxy \
  llm-proxy=go-llm-proxy:v1.1.0 \
  -n llm-proxy

# Check rollout status
kubectl rollout status deployment/llm-proxy -n llm-proxy
```

### Rollback

```bash
kubectl rollout undo deployment/llm-proxy -n llm-proxy
```

## Troubleshooting

### Check logs

```bash
kubectl logs -f deployment/llm-proxy -n llm-proxy
```

### Check events

```bash
kubectl get events -n llm-proxy --sort-by='.lastTimestamp'
```

### Port forward for testing

```bash
kubectl port-forward svc/llm-proxy 8080:80 -n llm-proxy
```

## Resources

- **CPU Request**: 100m
- **CPU Limit**: 500m
- **Memory Request**: 25Mi
- **Memory Limit**: 50Mi
- **Storage**: 35Gi total (config: 10Gi, logs: 5Gi, backup: 20Gi)

## High Availability

- **Replicas**: 2 (configurable)
- **PodDisruptionBudget**: Ensures at least 1 pod available
- **PodAntiAffinity**: Spreads pods across nodes
- **RollingUpdate**: Zero-downtime deployments
