# Go LLM Proxy - Makefile

.PHONY: all build clean test test-coverage lint run
.PHONY: docker-build docker-run docker-down
.PHONY: admin help

# Build variables
BINARY_NAME=go-llm-proxy
MAIN_PATH=cmd/proxy/main.go
BUILD_DIR=build
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

# Default target
all: clean lint test build

# Build the binary
build:
	@echo "Building ${BINARY_NAME}..."
	@mkdir -p ${BUILD_DIR}
	go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME} ${MAIN_PATH}

# Build for multiple platforms
build-cross:
	@echo "Building for multiple platforms..."
	@mkdir -p ${BUILD_DIR}
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-linux-amd64 ${MAIN_PATH}
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-linux-arm64 ${MAIN_PATH}
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-darwin-amd64 ${MAIN_PATH}
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-darwin-arm64 ${MAIN_PATH}

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf ${BUILD_DIR}
	go clean

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.txt ./...
	go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
test-bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

# Run the application
run:
	@echo "Running ${BINARY_NAME}..."
	go run ${MAIN_PATH}

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t go-llm-proxy:${VERSION} .

# Docker run
docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 -p 8443:8443 go-llm-proxy:${VERSION}

# Docker compose
docker-up:
	@echo "Starting with Docker Compose..."
	docker-compose up -d

docker-down:
	@echo "Stopping Docker Compose..."
	docker-compose down

# Admin CLI
admin:
	@echo "Running admin CLI..."
	go run cmd/admin/main.go

# Generate mocks
mock:
	@echo "Generating mocks..."
	go generate ./...

# Security scan
security:
	@echo "Running security scan..."
	gosec ./...

# Dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Help
help:
	@echo "Available targets:"
	@echo "  all           - Build, test, and lint"
	@echo "  build         - Build the binary"
	@echo "  clean         - Clean build artifacts"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  lint          - Lint code"
	@echo "  fmt           - Format code"
	@echo "  run           - Run the application"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run Docker container"
	@echo "  docker-up     - Start with Docker Compose"
	@echo "  docker-down   - Stop Docker Compose"
	@echo "  help          - Show this help"
