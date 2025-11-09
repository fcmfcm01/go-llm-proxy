#!/bin/bash

# LLM Proxy Deployment Script
# This script automates the deployment of LLM Proxy to various environments

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
IMAGE_NAME="go-llm-proxy"
VERSION="${VERSION:-latest}"
NAMESPACE="llm-proxy"
KUBE_CONTEXT="${KUBE_CONTEXT:-}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Help function
show_help() {
    cat << EOF
LLM Proxy Deployment Script

Usage: $0 [OPTIONS] COMMAND

Commands:
    build           Build Docker image
    docker          Deploy using Docker Compose
    kubernetes      Deploy to Kubernetes
    systemd         Deploy using systemd
    rollback        Rollback to previous version
    status          Check deployment status
    cleanup         Clean up resources

Options:
    -v, --version VERSION    Specify version (default: latest)
    -n, --namespace NS       Kubernetes namespace (default: llm-proxy)
    -c, --context CTX        Kubernetes context
    -e, --env ENV            Environment (dev, staging, prod)
    -h, --help               Show this help message

Examples:
    $0 build
    $0 --version v1.0.0 docker
    $0 --namespace llm-proxy --env prod kubernetes
    $0 rollback --version v0.9.0

EOF
}

# Parse command line arguments
ENVIRONMENT="dev"
COMMAND=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -n|--namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        -c|--context)
            KUBE_CONTEXT="$2"
            shift 2
            ;;
        -e|--env)
            ENVIRONMENT="$2"
            shift 2
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        build|docker|kubernetes|systemd|rollback|status|cleanup)
            COMMAND="$1"
            shift
            ;;
        *)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Check if command is provided
if [ -z "$COMMAND" ]; then
    log_error "No command specified"
    show_help
    exit 1
fi

# Pre-deployment checks
check_dependencies() {
    log_info "Checking dependencies..."

    case $COMMAND in
        build)
            command -v docker >/dev/null 2>&1 || { log_error "docker is required"; exit 1; }
            ;;
        docker)
            command -v docker >/dev/null 2>&1 || { log_error "docker is required"; exit 1; }
            command -v docker-compose >/dev/null 2>&1 || { log_error "docker-compose is required"; exit 1; }
            ;;
        kubernetes)
            command -v kubectl >/dev/null 2>&1 || { log_error "kubectl is required"; exit 1; }
            command -v kustomize >/dev/null 2>&1 || { log_warn "kustomize not found, using kubectl apply"; }
            ;;
        systemd)
            command -v systemctl >/dev/null 2>&1 || { log_error "systemd is required"; exit 1; }
            ;;
    esac

    log_info "Dependencies check passed"
}

# Build Docker image
build_image() {
    log_info "Building Docker image: ${IMAGE_NAME}:${VERSION}"

    cd "$PROJECT_DIR"

    # Build with BuildKit for better performance
    export DOCKER_BUILDKIT=1

    docker build \
        --tag "${IMAGE_NAME}:${VERSION}" \
        --tag "${IMAGE_NAME}:latest" \
        .

    log_info "Image built successfully: ${IMAGE_NAME}:${VERSION}"
}

# Deploy with Docker Compose
deploy_docker() {
    log_info "Deploying with Docker Compose (env: $ENVIRONMENT)"

    cd "$PROJECT_DIR/deployment"

    # Load environment file
    if [ -f ".env.${ENVIRONMENT}" ]; then
        log_info "Loading environment file: .env.${ENVIRONMENT}"
        export $(cat .env.${ENVIRONMENT} | grep -v '^#' | xargs)
    fi

    # Deploy
    case $ENVIRONMENT in
        prod)
            docker-compose -f docker-compose.prod.yml up -d
            ;;
        staging)
            docker-compose -f docker-compose.staging.yml up -d
            ;;
        *)
            docker-compose up -d
            ;;
    esac

    log_info "Waiting for services to be ready..."
    sleep 10

    # Health check
    if curl -f http://localhost:8080/healthz >/dev/null 2>&1; then
        log_info "Deployment successful! Services are healthy"
    else
        log_error "Health check failed"
        docker-compose logs
        exit 1
    fi
}

# Deploy to Kubernetes
deploy_kubernetes() {
    log_info "Deploying to Kubernetes (env: $ENVIRONMENT, namespace: $NAMESPACE)"

    cd "$PROJECT_DIR/deployment/k8s"

    # Set context if specified
    if [ -n "$KUBE_CONTEXT" ]; then
        kubectl config use-context "$KUBE_CONTEXT"
    fi

    # Create namespace if it doesn't exist
    kubectl get namespace "$NAMESPACE" >/dev/null 2>&1 || {
        log_info "Creating namespace: $NAMESPACE"
        kubectl create namespace "$NAMESPACE"
    }

    # Deploy using kustomize or kubectl
    if command -v kustomize >/dev/null 2>&1; then
        log_info "Deploying with kustomize"
        kubectl apply -k .
    else
        log_info "Deploying with kubectl apply"
        kubectl apply -f .
    fi

    log_info "Waiting for deployment to be ready..."
    kubectl wait --for=condition=available --timeout=300s deployment/llm-proxy -n "$NAMESPACE"

    # Check pod status
    kubectl get pods -n "$NAMESPACE"
    kubectl get svc -n "$NAMESPACE"

    log_info "Kubernetes deployment complete"
}

# Deploy with systemd
deploy_systemd() {
    log_info "Deploying with systemd"

    # Check if running as root
    if [ "$EUID" -ne 0 ]; then
        log_error "systemd deployment requires root privileges"
        exit 1
    fi

    # Create user if it doesn't exist
    id llm-proxy >/dev/null 2>&1 || {
        log_info "Creating system user: llm-proxy"
        useradd -r -s /bin/false -d /opt/llm-proxy -M llm-proxy
    }

    # Create directories
    mkdir -p /opt/llm-proxy/{bin,data,config}
    mkdir -p /var/log/llm-proxy
    mkdir -p /etc/llm-proxy

    # Install binary
    log_info "Installing binary to /opt/llm-proxy/bin/llm-proxy"
    cp "$PROJECT_DIR/llm-proxy" /opt/llm-proxy/bin/
    chmod +x /opt/llm-proxy/bin/llm-proxy
    chown -R llm-proxy:llm-proxy /opt/llm-proxy

    # Install service file
    log_info "Installing systemd service"
    cp "$PROJECT_DIR/deployment/systemd/llm-proxy.service" /etc/systemd/system/
    systemctl daemon-reload

    # Install environment file
    if [ ! -f /etc/default/llm-proxy ]; then
        log_info "Creating environment file"
        cp "$PROJECT_DIR/deployment/systemd/llm-proxy.env" /etc/default/llm-proxy
        chmod 640 /etc/default/llm-proxy
        chown root:llm-proxy /etc/default/llm-proxy
    fi

    # Install configuration
    if [ -f "$PROJECT_DIR/config/production.yaml" ]; then
        cp "$PROJECT_DIR/config/production.yaml" /opt/llm-proxy/config/
        chown -R llm-proxy:llm-proxy /opt/llm-proxy/config
    fi

    # Start and enable service
    log_info "Starting and enabling service"
    systemctl enable llm-proxy
    systemctl start llm-proxy

    # Check status
    sleep 5
    if systemctl is-active --quiet llm-proxy; then
        log_info "systemd deployment successful"
        systemctl status llm-proxy
    else
        log_error "systemd deployment failed"
        journalctl -u llm-proxy -n 50
        exit 1
    fi
}

# Rollback deployment
rollback() {
    log_warn "Rolling back to version: $VERSION"

    case $COMMAND in
        kubernetes)
            if [ -n "$KUBE_CONTEXT" ]; then
                kubectl config use-context "$KUBE_CONTEXT"
            fi

            log_info "Rolling back Kubernetes deployment"
            kubectl rollout undo deployment/llm-proxy -n "$NAMESPACE"
            kubectl rollout status deployment/llm-proxy -n "$NAMESPACE"
            ;;
        systemd)
            if [ "$EUID" -ne 0 ]; then
                log_error "systemd rollback requires root privileges"
                exit 1
            fi

            log_warn "Manual rollback required for systemd"
            log_info "Please manually restore the binary and restart the service"
            ;;
    esac

    log_info "Rollback complete"
}

# Check deployment status
check_status() {
    case $COMMAND in
        kubernetes)
            if [ -n "$KUBE_CONTEXT" ]; then
                kubectl config use-context "$KUBE_CONTEXT"
            fi

            log_info "Checking Kubernetes status"
            kubectl get all -n "$NAMESPACE"
            ;;
        systemd)
            log_info "Checking systemd status"
            systemctl status llm-proxy
            ;;
        docker)
            log_info "Checking Docker Compose status"
            cd "$PROJECT_DIR/deployment"
            docker-compose ps
            ;;
    esac
}

# Cleanup resources
cleanup() {
    log_warn "This will remove all LLM Proxy resources"

    read -p "Are you sure? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Cleanup cancelled"
        exit 0
    fi

    case $COMMAND in
        kubernetes)
            if [ -n "$KUBE_CONTEXT" ]; then
                kubectl config use-context "$KUBE_CONTEXT"
            fi

            log_info "Cleaning up Kubernetes resources"
            kubectl delete -f . -n "$NAMESPACE" 2>/dev/null || true
            ;;
        docker)
            log_info "Cleaning up Docker Compose resources"
            cd "$PROJECT_DIR/deployment"
            docker-compose down -v
            ;;
        systemd)
            if [ "$EUID" -ne 0 ]; then
                log_error "systemd cleanup requires root privileges"
                exit 1
            fi

            log_info "Cleaning up systemd service"
            systemctl stop llm-proxy 2>/dev/null || true
            systemctl disable llm-proxy 2>/dev/null || true
            rm -f /etc/systemd/system/llm-proxy.service
            systemctl daemon-reload
            ;;
    esac

    log_info "Cleanup complete"
}

# Main execution
main() {
    log_info "Starting LLM Proxy deployment"
    log_info "Command: $COMMAND, Version: $VERSION, Environment: $ENVIRONMENT"

    check_dependencies

    case $COMMAND in
        build)
            build_image
            ;;
        docker)
            build_image
            deploy_docker
            ;;
        kubernetes)
            build_image
            deploy_kubernetes
            ;;
        systemd)
            deploy_systemd
            ;;
        rollback)
            rollback
            ;;
        status)
            check_status
            ;;
        cleanup)
            cleanup
            ;;
        *)
            log_error "Unknown command: $COMMAND"
            show_help
            exit 1
            ;;
    esac

    log_info "Deployment script completed successfully"
}

# Run main function
main
