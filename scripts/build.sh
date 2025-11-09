#!/bin/bash

# Build script for LLM Proxy
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Build variables
BUILD_DIR=build
VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
BUILD_TIME=${BUILD_TIME:-$(date -u '+%Y-%m-%d_%H:%M:%S')}
LDFLAGS="-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"
BINARY_NAME=go-llm-proxy

# Function to print colored output
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to show usage
show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Build LLM Proxy

OPTIONS:
    -h, --help          Show this help message
    -v, --version       Build version (default: ${VERSION})
    -t, --target        Build target: linux, darwin, windows (default: linux)
    -a, --arch          Build architecture: amd64, arm64 (default: amd64)
    -o, --output        Output directory (default: ${BUILD_DIR})
    -d, --debug         Build with debug symbols
    -c, --clean         Clean build directory before building
    -l, --ldflags       Custom LDFLAGS (overrides default)
    --race              Enable race detector
    --static            Build static binary
    --docker            Build Docker image

EXAMPLES:
    $0                          # Build for linux/amd64
    $0 --target darwin          # Build for darwin/amd64
    $0 --arch arm64             # Build for linux/arm64
    $0 --clean --debug          # Clean and build with debug
    $0 --docker                 # Build Docker image

EOF
}

# Parse command line arguments
CLEAN=false
DEBUG=false
ENABLE_RACE=false
STATIC=false
BUILD_DOCKER=false
TARGET=linux
ARCH=amd64
OUTPUT=${BUILD_DIR}
CUSTOM_LDFLAGS=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_usage
            exit 0
            ;;
        -v|--version)
            VERSION="$2"
            shift 2
            ;;
        -t|--target)
            TARGET="$2"
            shift 2
            ;;
        -a|--arch)
            ARCH="$2"
            shift 2
            ;;
        -o|--output)
            OUTPUT="$2"
            shift 2
            ;;
        -d|--debug)
            DEBUG=true
            shift
            ;;
        -c|--clean)
            CLEAN=true
            shift
            ;;
        -l|--ldflags)
            CUSTOM_LDFLAGS="$2"
            shift 2
            ;;
        --race)
            ENABLE_RACE=true
            shift
            ;;
        --static)
            STATIC=true
            shift
            ;;
        --docker)
            BUILD_DOCKER=true
            shift
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Build Docker image
if [ "$BUILD_DOCKER" = true ]; then
    print_info "Building Docker image..."
    docker build -t ${BINARY_NAME}:${VERSION} -t ${BINARY_NAME}:latest -f deployment/Dockerfile ..
    print_info "Docker image built successfully"
    print_info "Image: ${BINARY_NAME}:${VERSION}"
    print_info "Image: ${BINARY_NAME}:latest"
    exit 0
fi

# Clean build directory
if [ "$CLEAN" = true ]; then
    print_info "Cleaning build directory..."
    rm -rf ${OUTPUT}
fi

# Create build directory
mkdir -p ${OUTPUT}

# Prepare LDFLAGS
if [ -z "$CUSTOM_LDFLAGS" ]; then
    LDFLAGS="-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"
    if [ "$DEBUG" = true ]; then
        LDFLAGS="${LDFLAGS} -linkmode external -extldflags '-static'"
    fi
else
    LDFLAGS="$CUSTOM_LDFLAGS"
fi

# Set GO environment variables
export GOOS=${TARGET}
export GOARCH=${ARCH}

# Build flags
BUILD_FLAGS="-a"
if [ "$DEBUG" = false ]; then
    BUILD_FLAGS="${BUILD_FLAGS} -ldflags '${LDFLAGS} -w -s'"  # Strip debug info
else
    BUILD_FLAGS="${BUILD_FLAGS} -ldflags '${LDFLAGS}'"
fi

if [ "$STATIC" = true ]; then
    export CGO_ENABLED=0
fi

# Run tests
print_info "Running tests..."
go test -v ./... || {
    print_error "Tests failed"
    exit 1
}

# Build binary
print_info "Building ${BINARY_NAME} for ${TARGET}/${ARCH}..."
print_info "Version: ${VERSION}"
print_info "Build Time: ${BUILD_TIME}"
print_info "Output: ${OUTPUT}"

if [ "$ENABLE_RACE" = true ]; then
    print_warn "Building with race detector (slower)"
    BUILD_FLAGS="${BUILD_FLAGS} -race"
fi

# Execute build
eval "go build ${BUILD_FLAGS} -o ${OUTPUT}/${BINARY_NAME} cmd/proxy/main.go"

# Verify binary
if [ -f "${OUTPUT}/${BINARY_NAME}" ]; then
    BINARY_SIZE=$(du -h "${OUTPUT}/${BINARY_NAME}" | cut -f1)
    print_info "Build successful!"
    print_info "Binary: ${OUTPUT}/${BINARY_NAME}"
    print_info "Size: ${BINARY_SIZE}"

    # Check if static binary
    if command -v file &> /dev/null; then
        FILE_TYPE=$(file "${OUTPUT}/${BINARY_NAME}")
        print_info "Type: ${FILE_TYPE}"
    fi
else
    print_error "Build failed"
    exit 1
fi

print_info "Build completed successfully"
