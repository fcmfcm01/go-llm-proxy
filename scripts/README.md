# Deployment Scripts

This directory contains deployment and management scripts for LLM Proxy.

## Scripts Overview

### deploy.sh

Comprehensive deployment script supporting multiple deployment methods.

**Usage:**

```bash
./deploy.sh [OPTIONS] COMMAND
```

**Commands:**
- `build` - Build Docker image
- `docker` - Deploy using Docker Compose
- `kubernetes` - Deploy to Kubernetes
- `systemd` - Deploy using systemd
- `rollback` - Rollback to previous version
- `status` - Check deployment status
- `cleanup` - Clean up resources

**Options:**
- `-v, --version VERSION` - Specify version
- `-n, --namespace NS` - Kubernetes namespace
- `-c, --context CTX` - Kubernetes context
- `-e, --env ENV` - Environment (dev, staging, prod)

**Examples:**

```bash
# Build Docker image
./deploy.sh build

# Deploy with Docker Compose
./deploy.sh --env prod docker

# Deploy to Kubernetes
./deploy.sh --namespace llm-proxy --env prod kubernetes

# Rollback Kubernetes deployment
./deploy.sh --version v0.9.0 rollback

# Check status
./deploy.sh status
```

### health-check.sh

Health check script for monitoring service availability.

**Usage:**

```bash
./health-check.sh [OPTIONS]
```

**Options:**
- `-h, --host HOST` - Host to check (default: localhost)
- `-p, --port PORT` - Port to check (default: 8080)
- `-t, --timeout SEC` - Timeout in seconds (default: 5)

**Examples:**

```bash
# Check local service
./health-check.sh

# Check remote service
./health-check.sh --host example.com --port 8080

# Check with custom timeout
./health-check.sh --timeout 10
```

**Checks performed:**
- Service availability (/healthz)
- Service readiness (/healthz/ready)
- Metrics endpoint (/metrics)
- Proxy endpoint (/v1/models)
- TLS endpoint (if enabled)

### backup.sh

Backup script for configuration, data, and service files.

**Usage:**

```bash
./backup.sh [COMMAND]
```

**Commands:**
- `backup` - Create a new backup (default)
- `list` - List available backups
- `cleanup` - Remove old backups

**Environment variables:**
- `BACKUP_DIR` - Backup directory (default: /var/backups/llm-proxy)
- `DATA_DIR` - Data directory (default: /opt/llm-proxy/data)
- `CONFIG_DIR` - Config directory (default: /opt/llm-proxy/config)
- `RETENTION_DAYS` - Days to keep backups (default: 7)

**Examples:**

```bash
# Create backup
./backup.sh backup

# List backups
./backup.sh list

# Cleanup old backups
./backup.sh cleanup

# Custom backup location
BACKUP_DIR=/custom/path ./backup.sh backup
```

## Quick Start

### Docker Deployment

```bash
# Build and deploy
./deploy.sh docker

# Check health
./health-check.sh

# Backup data
./backup.sh backup
```

### Kubernetes Deployment

```bash
# Build image
docker build -t go-llm-proxy:v1.0.0 .

# Tag and push to registry
docker tag go-llm-proxy:v1.0.0 myregistry.io/go-llm-proxy:v1.0.0
docker push myregistry.io/go-llm-proxy:v1.0.0

# Deploy to Kubernetes
./deploy.sh --version v1.0.0 --namespace llm-proxy --env prod kubernetes

# Check status
./deploy.sh --namespace llm-proxy status
```

### Systemd Deployment

```bash
# Deploy using systemd
sudo ./deploy.sh systemd

# Check health
./health-check.sh

# Backup
sudo ./backup.sh backup
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Deploy LLM Proxy

on:
  push:
    tags:
      - 'v*'

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Build image
        run: ./deploy.sh build

      - name: Deploy to Kubernetes
        run: |
          ./deploy.sh \
            --version ${{ github.ref_name }} \
            --namespace llm-proxy \
            --env production \
            kubernetes
```

### GitLab CI

```yaml
deploy:
  stage: deploy
  script:
    - ./deploy.sh --version $CI_COMMIT_TAG kubernetes
  only:
    - tags
```

## Monitoring

### Health Checks

Run health checks periodically:

```bash
# Add to crontab
*/5 * * * * /opt/llm-proxy/scripts/health-check.sh || \
  echo "Health check failed" | mail -s "LLM Proxy Alert" admin@example.com
```

### Automated Backups

Schedule daily backups:

```bash
# Add to crontab
0 2 * * * /opt/llm-proxy/scripts/backup.sh
```

## Troubleshooting

### Check deployment status

```bash
# Kubernetes
./deploy.sh status

# Systemd
systemctl status llm-proxy
journalctl -u llm-proxy

# Docker
docker-compose ps
docker-compose logs
```

### Manual health check

```bash
./health-check.sh --host localhost --port 8080
```

### Manual backup

```bash
./backup.sh backup
ls -lh /var/backups/llm-proxy/
```

### Rollback

```bash
# Kubernetes
./deploy.sh --version v0.9.0 rollback

# Systemd (manual)
sudo systemctl stop llm-proxy
sudo cp /opt/llm-proxy/bin/llm-proxy.bak /opt/llm-proxy/bin/llm-proxy
sudo systemctl start llm-proxy
```

## Best Practices

1. **Always test in staging** before production deployment
2. **Keep backups** of configuration and data
3. **Monitor health** with regular health checks
4. **Use versioning** for deployments
5. **Enable TLS** in production
6. **Set up alerts** for health check failures
7. **Rotate logs** regularly
8. **Update regularly** for security patches
