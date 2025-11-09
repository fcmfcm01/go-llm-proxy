#!/bin/bash

# Container Image Size Validation Script
# Validates <50MB container image size (SC-005)
# Part of T175 - Container size validation

set -e

echo "=== LLM Proxy Container Image Size Validation ==="
echo "Date: $(date)"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

SIZE_LIMIT_MB=50
SIZE_LIMIT_BYTES=$((SIZE_LIMIT_MB * 1024 * 1024))

echo "Container Size Requirements:"
echo "  Target: <${SIZE_LIMIT_MB}MB"
echo "  Python version: ~500MB (baseline)"
echo "  Expected reduction: ~90%"
echo ""

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo -e "${YELLOW}⚠ Docker not found${NC}"
    echo "Cannot build or check container image"
    echo "Please install Docker to run this validation"
    exit 1
fi

echo "1. Checking Dockerfile..."
echo "----------------------------------------"

if [ -f "Dockerfile" ]; then
    echo -e "${GREEN}✓ Dockerfile found${NC}"

    # Check for multi-stage build
    if grep -q "FROM.*AS builder" Dockerfile; then
        echo -e "${GREEN}✓ Multi-stage build detected${NC}"
    else
        echo -e "${YELLOW}⚠ Not using multi-stage build${NC}"
        echo "  Consider multi-stage build for smaller image"
    fi

    # Check base image
    if grep -q "alpine" Dockerfile; then
        echo -e "${GREEN}✓ Using Alpine base image (lightweight)${NC}"
    elif grep -q "debian\|ubuntu" Dockerfile; then
        echo -e "${YELLOW}⚠ Using Debian/Ubuntu base (larger than Alpine)${NC}"
    fi

    # Check for CGO disabled
    if grep -q "CGO_ENABLED=0" Dockerfile; then
        echo -e "${GREEN}✓ CGO disabled (static binary)${NC}"
    else
        echo -e "${YELLOW}⚠ CGO may be enabled${NC}"
    fi

    # Check for unnecessary packages
    if grep -q "curl\|wget\|bash" Dockerfile; then
        echo -e "${YELLOW}⚠ Debug tools found in image${NC}"
        echo "  Consider removing curl/wget/bash from final image"
    fi

    # Check for non-root user
    if grep -q "USER.*[0-9]" Dockerfile; then
        echo -e "${GREEN}✓ Non-root user configured${NC}"
    else
        echo -e "${YELLOW}⚠ Running as root${NC}"
        echo "  Consider adding USER directive for security"
    fi
else
    echo -e "${RED}✗ Dockerfile not found${NC}"
    exit 1
fi

echo ""
echo "2. Building container image..."
echo "----------------------------------------"

# Build the image
IMAGE_NAME="go-llm-proxy:size-test"
echo "Building image: $IMAGE_NAME"

docker build -t $IMAGE_NAME . 2>&1 | tee build-output.log

if [ ${PIPESTATUS[0]} -eq 0 ]; then
    echo -e "${GREEN}✓ Build successful${NC}"
else
    echo -e "${RED}✗ Build failed${NC}"
    exit 1
fi

echo ""
echo "3. Analyzing image layers..."
echo "----------------------------------------"

# Show image history
echo "Image layers:"
docker history $IMAGE_NAME --human=true

echo ""
echo "4. Checking image size..."
echo "----------------------------------------"

# Get image size
IMAGE_SIZE=$(docker images $IMAGE_NAME --format "{{.Size}}")
IMAGE_BYTES=$(docker inspect $IMAGE_NAME --format='{{.Size}}')

echo "Image size: $IMAGE_SIZE"
echo "Image size (bytes): $IMAGE_BYTES"

# Convert to MB for display
IMAGE_SIZE_MB=$(echo "scale=2; $IMAGE_BYTES / 1024 / 1024" | bc)

echo ""
echo "5. Comparing with requirements..."
echo "----------------------------------------"

# Check against limit
echo "Size limit: ${SIZE_LIMIT_MB}MB"
echo "Image size: ${IMAGE_SIZE_MB}MB"

# Check if below limit
if [ $IMAGE_BYTES -lt $SIZE_LIMIT_BYTES ]; then
    echo -e "${GREEN}✓ PASS: Image size (${IMAGE_SIZE_MB}MB) is < ${SIZE_LIMIT_MB}MB${NC}"
    COMPLIANCE="PASS"
    VERDICT=0
else
    echo -e "${RED}✗ FAIL: Image size (${IMAGE_SIZE_MB}MB) is ≥ ${SIZE_LIMIT_MB}MB${NC}"
    echo "FAILED: Container image exceeds size limit"
    COMPLIANCE="FAIL"
    VERDICT=1
fi

# Compare with Python version
PYTHON_SIZE_MB=500
REDUCTION=$(echo "scale=1; (1 - $IMAGE_SIZE_MB / $PYTHON_SIZE_MB) * 100" | bc)
echo ""
echo "Python version: ~${PYTHON_SIZE_MB}MB"
echo "Size reduction: ${REDUCTION}%"

if (( $(echo "$REDUCTION >= 90" | bc -l) )); then
    echo -e "${GREEN}✓ Size reduction (${REDUCTION}%) ≥ 90%${NC}"
else
    echo -e "${YELLOW}⚠ Size reduction (${REDUCTION}%) < 90%${NC}"
fi

echo ""
echo "6. Comparing with common images..."
echo "----------------------------------------"

# List image sizes for comparison
echo "Comparison with common base images:"
echo "  Alpine: ~5MB"
echo "  Ubuntu: ~80MB"
echo "  Python (3.11-slim): ~120MB"
echo "  Node.js (alpine): ~40MB"
echo "  This image: ${IMAGE_SIZE_MB}MB"

echo ""
echo "7. Layer analysis..."
echo "----------------------------------------"

# Get layer sizes
echo "Largest layers:"
docker history $IMAGE_NAME --format "table {{.Size}}\t{{.CreatedBy}}" --no-trunc | head -10

echo ""
echo "8. Generating report..."
echo "----------------------------------------"

# Create detailed report
REPORT_FILE="container-size-report.txt"
{
    echo "LLM Proxy Container Image Size Report"
    echo "======================================"
    echo ""
    echo "Date: $(date)"
    echo "Image: $IMAGE_NAME"
    echo ""
    echo "Size Analysis:"
    echo "  Raw bytes: $IMAGE_BYTES"
    echo "  Human readable: $IMAGE_SIZE"
    echo "  Size in MB: ${IMAGE_SIZE_MB}MB"
    echo "  Size limit: ${SIZE_LIMIT_MB}MB"
    echo ""
    echo "Compliance:"
    echo "  Status: $COMPLIANCE"
    echo "  Under limit: $([ $IMAGE_BYTES -lt $SIZE_LIMIT_BYTES ] && echo "Yes" || echo "No")"
    echo "  Reduction from Python: ${REDUCTION}%"
    echo ""
    echo "Requirements (SC-005):"
    echo "  Target: <${SIZE_LIMIT_MB}MB"
    echo "  Python baseline: ~500MB"
    echo "  Expected reduction: ~90%"
    echo ""
    echo "Build Details:"
    echo "  Multi-stage: $(grep -q "FROM.*AS builder" Dockerfile && echo "Yes" || echo "No")"
    echo "  Base image: $(grep "^FROM" Dockerfile | head -1 | awk '{print $2}')"
    echo "  CGO disabled: $(grep -q "CGO_ENABLED=0" Dockerfile && echo "Yes" || echo "No")"
    echo ""
} > $REPORT_FILE

echo "Report saved to: $REPORT_FILE"

echo ""
echo "9. Optimization recommendations..."
echo "----------------------------------------"

if [ $IMAGE_BYTES -ge $SIZE_LIMIT_BYTES ]; then
    echo "Recommendations to reduce image size:"
    echo ""
    echo "1. Use multi-stage build (already done if present)"
    echo "   - Build in builder image, copy binary to alpine"
    echo ""
    echo "2. Use Alpine Linux base"
    echo "   - Change FROM golang:1.21-alpine AS builder"
    echo "   - Change FROM alpine:latest"
    echo ""
    echo "3. Remove unnecessary tools"
    echo "   - Don't include curl, wget, git in final image"
    echo "   - Only include ca-certificates for HTTPS"
    echo ""
    echo "4. Disable CGO"
    echo "   - Set CGO_ENABLED=0 in build"
    echo ""
    echo "5. Strip binary"
    echo "   - Add: RUN strip /app/llm-proxy"
    echo ""
    echo "6. Use .dockerignore"
    echo "   - Exclude .git, tests, docs from build context"
    echo ""
fi

# Show .dockerignore if exists
if [ -f ".dockerignore" ]; then
    echo -e "${GREEN}✓ .dockerignore found${NC}"
    echo "Contents:"
    cat .dockerignore
else
    echo -e "${YELLOW}⚠ No .dockerignore file${NC}"
    echo "Create .dockerignore to reduce build context:"
    cat << 'EOF'
.dockerignore
.git
.gitignore
README.md
docs/
tests/
*.md
Dockerfile*
docker-compose*
EOF
fi

echo ""
echo "10. Cleanup..."
echo "----------------------------------------"

echo "Removing test image: $IMAGE_NAME"
docker rmi $IMAGE_NAME > /dev/null 2>&1 || true

echo -e "${GREEN}✓ Cleanup complete${NC}"

echo ""
echo "=== Container Size Validation Results ==="
echo ""
echo "Image size: ${IMAGE_SIZE_MB}MB"
echo "Limit: ${SIZE_LIMIT_MB}MB"
echo "Status: $COMPLIANCE"

if [ $VERDICT -eq 0 ]; then
    echo ""
    echo -e "${GREEN}✓ VALIDATION PASSED${NC}"
    echo "  Container image is within size requirements"
    echo "  Ready for deployment"
else
    echo ""
    echo -e "${RED}✗ VALIDATION FAILED${NC}"
    echo "  Container image exceeds size limit"
    echo "  Please optimize before deployment"
fi

echo ""
echo "Summary saved to: $REPORT_FILE"

exit $VERDICT
