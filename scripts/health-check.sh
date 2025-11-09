#!/bin/bash

# Health check script for LLM Proxy

set -euo pipefail

# Configuration
HOST="${HOST:-localhost}"
PORT="${PORT:-8080}"
TIMEOUT="${TIMEOUT:-5}"
URL="http://${HOST}:${PORT}"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Check if service is available
check_service() {
    echo -n "Checking service availability... "

    if curl -sf --max-time "$TIMEOUT" "$URL/healthz" > /dev/null 2>&1; then
        echo -e "${GREEN}OK${NC}"
        return 0
    else
        echo -e "${RED}FAILED${NC}"
        return 1
    fi
}

# Check if service is ready
check_ready() {
    echo -n "Checking service readiness... "

    if curl -sf --max-time "$TIMEOUT" "$URL/healthz/ready" > /dev/null 2>&1; then
        echo -e "${GREEN}OK${NC}"
        return 0
    else
        echo -e "${RED}FAILED${NC}"
        return 1
    fi
}

# Check metrics endpoint
check_metrics() {
    echo -n "Checking metrics endpoint... "

    if curl -sf --max-time "$TIMEOUT" "$URL/metrics" > /dev/null 2>&1; then
        echo -e "${GREEN}OK${NC}"
        return 0
    else
        echo -e "${YELLOW}WARNING${NC}"
        return 1
    fi
}

# Check proxy endpoint
check_proxy() {
    echo -n "Checking proxy endpoint... "

    if curl -sf --max-time "$TIMEOUT" "$URL/v1/models" > /dev/null 2>&1; then
        echo -e "${GREEN}OK${NC}"
        return 0
    else
        echo -e "${YELLOW}WARNING${NC}"
        return 1
    fi
}

# Check TLS if enabled
check_tls() {
    echo -n "Checking TLS endpoint... "

    TLS_PORT="${TLS_PORT:-8443}"
    TLS_URL="https://${HOST}:${TLS_PORT}"

    if curl -sfk --max-time "$TIMEOUT" "$TLS_URL/healthz" > /dev/null 2>&1; then
        echo -e "${GREEN}OK${NC}"
        return 0
    else
        echo -e "${YELLOW}WARNING${NC}"
        return 1
    fi
}

# Run all checks
main() {
    echo "LLM Proxy Health Check"
    echo "======================"
    echo "Host: $HOST"
    echo "Port: $PORT"
    echo ""

    EXIT_CODE=0

    check_service || EXIT_CODE=1
    check_ready || EXIT_CODE=1
    check_metrics || EXIT_CODE=1
    check_proxy || EXIT_CODE=1
    check_tls || true  # TLS check is optional

    echo ""
    if [ $EXIT_CODE -eq 0 ]; then
        echo -e "${GREEN}All checks passed${NC}"
    else
        echo -e "${RED}Some checks failed${NC}"
    fi

    exit $EXIT_CODE
}

# Parse command line
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--host)
            HOST="$2"
            shift 2
            ;;
        -p|--port)
            PORT="$2"
            shift 2
            ;;
        -t|--timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo "Options:"
            echo "  -h, --host HOST     Host to check (default: localhost)"
            echo "  -p, --port PORT     Port to check (default: 8080)"
            echo "  -t, --timeout SEC   Timeout in seconds (default: 5)"
            echo "  --help              Show this help"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

main
