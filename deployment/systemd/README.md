# Systemd Service Deployment

This directory contains systemd service files for deploying LLM Proxy on Linux systems using systemd.

## Installation

### 1. Create system user

```bash
sudo useradd -r -s /bin/false -d /opt/llm-proxy -M llm-proxy
```

### 2. Create directories

```bash
sudo mkdir -p /opt/llm-proxy/{bin,data,config}
sudo mkdir -p /var/log/llm-proxy
sudo mkdir -p /etc/llm-proxy
```

### 3. Install binary

```bash
sudo cp llm-proxy /opt/llm-proxy/bin/
sudo chmod +x /opt/llm-proxy/bin/llm-proxy
sudo chown -R llm-proxy:llm-proxy /opt/llm-proxy
```

### 4. Install systemd service

```bash
sudo cp llm-proxy.service /etc/systemd/system/
sudo systemctl daemon-reload
```

### 5. Configure environment

```bash
sudo cp llm-proxy.env /etc/default/llm-proxy
sudo chmod 640 /etc/default/llm-proxy
sudo chown root:llm-proxy /etc/default/llm-proxy
# Edit the file and set your secrets
sudo nano /etc/default/llm-proxy
```

### 6. Install configuration

```bash
sudo cp config/production.yaml /opt/llm-proxy/config/
sudo chown -R llm-proxy:llm-proxy /opt/llm-proxy/config
```

### 7. Start and enable service

```bash
sudo systemctl enable llm-proxy
sudo systemctl start llm-proxy
sudo systemctl status llm-proxy
```

## Management

### Start the service

```bash
sudo systemctl start llm-proxy
```

### Stop the service

```bash
sudo systemctl stop llm-proxy
```

### Restart the service

```bash
sudo systemctl restart llm-proxy
```

### Reload configuration

```bash
sudo systemctl reload llm-proxy
```

### View status

```bash
sudo systemctl status llm-proxy
```

### View logs

```bash
# Systemd journal
sudo journalctl -u llm-proxy -f

# Log files
sudo tail -f /var/log/llm-proxy/app.log
```

### Enable automatic start on boot

```bash
sudo systemctl enable llm-proxy
```

### Disable automatic start

```bash
sudo systemctl disable llm-proxy
```

## Configuration

### Environment File

The service uses `/etc/default/llm-proxy` for environment variables. Key settings:

- **Session Secret**: Generate a secure secret key
- **TLS Certificates**: Path to SSL certificates
- **Rate Limiting**: Requests per minute
- **Logging**: Log level and format

Example:

```bash
sudo nano /etc/default/llm-proxy
# Update LLM_PROXY_SECURITY_SESSION_SECRET
# Update paths to certificates
```

### Configuration File

The service uses `/opt/llm-proxy/config/production.yaml` for application configuration.

### TLS Certificates

Install TLS certificates in standard locations:

```bash
# Copy certificates
sudo cp your-cert.crt /etc/ssl/certs/llm-proxy.crt
sudo cp your-cert.key /etc/ssl/private/llm-proxy.key
sudo chmod 600 /etc/ssl/private/llm-proxy.key
sudo chown root:ssl-cert /etc/ssl/private/llm-proxy.key
```

## Security Features

The service file includes:

- **Non-privilege execution**: Runs as dedicated `llm-proxy` user
- **No new privileges**: Prevents privilege escalation
- **Private /tmp**: Isolated temporary directory
- **Protected system files**: Read-only system directories
- **Resource limits**: Memory and file descriptor limits
- **Restart policies**: Automatic restart on failure

## Troubleshooting

### Check service status

```bash
sudo systemctl status llm-proxy
```

### Check logs

```bash
sudo journalctl -u llm-proxy -n 100 -l
```

### Check configuration

```bash
sudo -u llm-proxy /opt/llm-proxy/bin/llm-proxy validate-config
```

### Test health endpoint

```bash
curl http://localhost:8080/healthz
```

### Verify binary

```bash
sudo -u llm-proxy /opt/llm-proxy/bin/llm-proxy version
```

## Upgrades

### Manual upgrade

```bash
# Stop service
sudo systemctl stop llm-proxy

# Backup current binary
sudo mv /opt/llm-proxy/bin/llm-proxy /opt/llm-proxy/bin/llm-proxy.bak

# Install new binary
sudo cp new-llm-proxy /opt/llm-proxy/bin/
sudo chmod +x /opt/llm-proxy/bin/llm-proxy
sudo chown llm-proxy:llm-proxy /opt/llm-proxy/bin/llm-proxy

# Start service
sudo systemctl start llm-proxy
```

### Check for issues

```bash
sudo systemctl status llm-proxy
sudo journalctl -u llm-proxy -n 50
```

## Performance Tuning

### Adjust limits in service file

Edit `/etc/systemd/system/llm-proxy.service`:

```ini
# Increase memory limit if needed
MemoryLimit=128M

# Increase file descriptor limit
LimitNOFILE=131072
```

Reload systemd and restart:

```bash
sudo systemctl daemon-reload
sudo systemctl restart llm-proxy
```

## Monitoring

### Create logrotate configuration

```bash
sudo cat > /etc/logrotate.d/llm-proxy << 'EOF'
/var/log/llm-proxy/*.log {
    daily
    missingok
    rotate 7
    compress
    delaycompress
    notifempty
    create 640 llm-proxy llm-proxy
    postrotate
        systemctl reload llm-proxy > /dev/null 2>&1 || true
    endscript
}
EOF
```

### Setup log monitoring

The service logs to systemd journal by default. Configure your log aggregation to collect from:

- `journalctl -u llm-proxy`
- `/var/log/llm-proxy/app.log`
