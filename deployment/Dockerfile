# Multi-stage Dockerfile for Go LLM Proxy
# Target: <50MB final image size

# Stage 1: Build
FROM golang:1.23-alpine AS builder

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates git

# Set working directory
WORKDIR /build

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 for static linking
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -ldflags '-extldflags "-static" -w -s' \
    -installsuffix cgo \
    -o main cmd/proxy/main.go

# Stage 2: Runtime
FROM alpine:3.18

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 -S appgroup && \
    adduser -u 1000 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Create necessary directories
RUN mkdir -p /app/data/config \
    /app/data/backup \
    /app/data/sessions \
    /app/logs \
    && chown -R appuser:appgroup /app

# Copy binary from builder stage
COPY --from=builder /build/main /app/

# Change ownership
RUN chown appuser:appgroup /app/main
RUN chmod +x /app/main

# Switch to non-root user
USER appuser

# Expose ports
EXPOSE 8080 8443

# Set health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

# Run the application
CMD ["/app/main"]
