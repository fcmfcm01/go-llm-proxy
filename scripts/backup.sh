#!/bin/bash

# Backup script for LLM Proxy

set -euo pipefail

# Configuration
BACKUP_DIR="${BACKUP_DIR:-/var/backups/llm-proxy}"
DATA_DIR="${DATA_DIR:-/opt/llm-proxy/data}"
CONFIG_DIR="${CONFIG_DIR:-/opt/llm-proxy/config}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_NAME="llm-proxy-backup-${TIMESTAMP}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Logging
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Create backup directory
setup() {
    if [ ! -d "$BACKUP_DIR" ]; then
        log_info "Creating backup directory: $BACKUP_DIR"
        mkdir -p "$BACKUP_DIR"
    fi
}

# Backup configuration files
backup_config() {
    log_info "Backing up configuration files..."

    if [ -d "$CONFIG_DIR" ]; then
        tar -czf "${BACKUP_DIR}/${BACKUP_NAME}-config.tar.gz" -C "$CONFIG_DIR" .
        log_info "Configuration backed up to: ${BACKUP_NAME}-config.tar.gz"
    else
        log_warn "Configuration directory not found: $CONFIG_DIR"
    fi
}

# Backup data files
backup_data() {
    log_info "Backing up data files..."

    if [ -d "$DATA_DIR" ]; then
        tar -czf "${BACKUP_DIR}/${BACKUP_NAME}-data.tar.gz" -C "$DATA_DIR" .
        log_info "Data backed up to: ${BACKUP_NAME}-data.tar.gz"
    else
        log_warn "Data directory not found: $DATA_DIR"
    fi
}

# Backup systemd service file
backup_service() {
    log_info "Backing up service file..."

    if [ -f /etc/systemd/system/llm-proxy.service ]; then
        cp /etc/systemd/system/llm-proxy.service "${BACKUP_DIR}/${BACKUP_NAME}-service.service"
        log_info "Service file backed up"
    else
        log_warn "Service file not found"
    fi
}

# Backup environment file
backup_env() {
    log_info "Backing up environment file..."

    if [ -f /etc/default/llm-proxy ]; then
        cp /etc/default/llm-proxy "${BACKUP_DIR}/${BACKUP_NAME}-env"
        chmod 600 "${BACKUP_DIR}/${BACKUP_NAME}-env"
        log_info "Environment file backed up"
    else
        log_warn "Environment file not found"
    fi
}

# Compress all backups
compress_backup() {
    log_info "Creating compressed archive..."

    cd "$BACKUP_DIR"
    tar -czf "${BACKUP_NAME}.tar.gz" \
        "${BACKUP_NAME}"-config.tar.gz \
        "${BACKUP_NAME}"-data.tar.gz \
        "${BACKUP_NAME}"-service.service \
        "${BACKUP_NAME}"-env 2>/dev/null || true

    # Remove individual files
    rm -f "${BACKUP_NAME}"-config.tar.gz \
          "${BACKUP_NAME}"-data.tar.gz \
          "${BACKUP_NAME}"-service.service \
          "${BACKUP_NAME}"-env

    log_info "Backup archive created: ${BACKUP_NAME}.tar.gz"
}

# Calculate backup size
show_size() {
    if [ -f "${BACKUP_DIR}/${BACKUP_NAME}.tar.gz" ]; then
        SIZE=$(du -h "${BACKUP_DIR}/${BACKUP_NAME}.tar.gz" | cut -f1)
        log_info "Backup size: $SIZE"
    fi
}

# Clean old backups
cleanup_old() {
    log_info "Cleaning up old backups (older than $RETENTION_DAYS days)..."

    find "$BACKUP_DIR" -name "llm-proxy-backup-*.tar.gz" -mtime +$RETENTION_DAYS -delete || true
    find "$BACKUP_DIR" -name "llm-proxy-backup-*" -type d -mtime +$RETENTION_DAYS -delete || true

    log_info "Old backups cleaned up"
}

# List backups
list_backups() {
    log_info "Available backups:"

    if ls "$BACKUP_DIR"/llm-proxy-backup-*.tar.gz 1> /dev/null 2>&1; then
        ls -lh "$BACKUP_DIR"/llm-proxy-backup-*.tar.gz | while read -r line; do
            echo "  $line"
        done
    else
        log_warn "No backups found"
    fi
}

# Show usage
show_help() {
    cat << EOF
LLM Proxy Backup Script

Usage: $0 [COMMAND]

Commands:
    backup     Create a new backup (default)
    list       List available backups
    cleanup    Remove old backups
    help       Show this help

Environment variables:
    BACKUP_DIR       Backup directory (default: /var/backups/llm-proxy)
    DATA_DIR         Data directory (default: /opt/llm-proxy/data)
    CONFIG_DIR       Config directory (default: /opt/llm-proxy/config)
    RETENTION_DAYS   Days to keep backups (default: 7)

Examples:
    $0 backup
    BACKUP_DIR=/custom/path $0 backup
    $0 list
    RETENTION_DAYS=30 $0 cleanup

EOF
}

# Main function
main() {
    COMMAND="${1:-backup}"

    case $COMMAND in
        backup)
            setup
            backup_config
            backup_data
            backup_service
            backup_env
            compress_backup
            show_size
            cleanup_old
            log_info "Backup completed successfully"
            ;;
        list)
            list_backups
            ;;
        cleanup)
            cleanup_old
            log_info "Cleanup completed"
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            log_error "Unknown command: $COMMAND"
            show_help
            exit 1
            ;;
    esac
}

main "$@"
