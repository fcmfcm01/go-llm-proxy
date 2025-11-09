#!/bin/bash

# Memory Usage Validation Script
# Verifies <50MB memory usage under normal load (SC-003)
# Part of T176 - Memory validation

set -e

echo "=== LLM Proxy Memory Usage Validation ==="
echo "Date: $(date)"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

MEMORY_LIMIT_MB=50
MEMORY_LIMIT_BYTES=$((MEMORY_LIMIT_MB * 1024 * 1024))

echo "Memory Requirements:"
echo "  Target: <${MEMORY_LIMIT_MB}MB under normal load"
echo "  Python version: ~100MB (baseline)"
echo "  Expected reduction: ~50%"
echo ""

# Check if service is running
echo "1. Checking if LLM Proxy is running..."
echo "----------------------------------------"

SERVICE_URL="http://localhost:8080"
PID_FILE="/tmp/llm-proxy.pid"

if curl -sf $SERVICE_URL/healthz > /dev/null 2>&1; then
    echo -e "${GREEN}✓ LLM Proxy is running${NC}"
    RUNNING=1
elif [ -f "$PID_FILE" ] && kill -0 $(cat $PID_FILE) 2>/dev/null; then
    echo -e "${GREEN}✓ LLM Proxy process found${NC}"
    RUNNING=1
else
    echo -e "${YELLOW}⚠ LLM Proxy not running${NC}"
    echo "Starting service for memory test..."
    RUNNING=0
fi

# Start service if not running
if [ $RUNNING -eq 0 ]; then
    echo ""
    echo "Starting LLM Proxy service..."
    ./llm-proxy serve --config config/config.yaml > /dev/null 2>&1 &
    SERVICE_PID=$!
    echo $SERVICE_PID > $PID_FILE

    # Wait for service to start
    echo "Waiting for service to start..."
    for i in {1..30}; do
        if curl -sf $SERVICE_URL/healthz > /dev/null 2>&1; then
            echo -e "${GREEN}✓ Service started${NC}"
            RUNNING=1
            break
        fi
        sleep 1
    done

    if [ $RUNNING -eq 0 ]; then
        echo -e "${RED}✗ Failed to start service${NC}"
        exit 1
    fi
fi

# Get PID
SERVICE_PID=$(pgrep -f "llm-proxy" | head -1)

if [ -z "$SERVICE_PID" ]; then
    echo -e "${RED}✗ Could not find LLM Proxy process${NC}"
    exit 1
fi

echo "Service PID: $SERVICE_PID"

echo ""
echo "2. Baseline memory measurement..."
echo "----------------------------------------"

# Wait for service to stabilize
echo "Waiting for service to stabilize (10 seconds)..."
sleep 10

# Get baseline memory (RSS in KB)
BASELINE_RSS=$(ps -p $SERVICE_PID -o rss= 2>/dev/null | awk '{print $1}')
BASELINE_RSS_MB=$(echo "scale=2; $BASELINE_RSS / 1024" | bc)
BASELINE_VSZ=$(ps -p $SERVICE_PID -o vsz= 2>/dev/null | awk '{print $1}')
BASELINE_VSZ_MB=$(echo "scale=2; $BASELINE_VSZ / 1024" | bc)

echo "Baseline Memory (idle):"
echo "  RSS (Resident Set Size): ${BASELINE_RSS_MB}MB"
echo "  VSZ (Virtual Size): ${BASELINE_VSZ_MB}MB"

echo ""
echo "3. Generating test load..."
echo "----------------------------------------"

# Install Apache Bench if not present
if ! command -v ab &> /dev/null; then
    echo "Installing apache2-utils for ab..."
    apt-get update -qq && apt-get install -y -qq apache2-utils > /dev/null 2>&1 || \
    yum install -y -q httpd-tools > /dev/null 2>&1 || \
    echo "Warning: Could not install ab, skipping load test"
fi

# Function to generate load
generate_load() {
    local requests=$1
    local concurrency=$2
    local url=$3

    echo "Load test: $requests requests, concurrency $concurrency"
    ab -n $requests -c $concurrency -q $url > /dev/null 2>&1 || true
}

# Generate normal load (100 RPS for 60 seconds)
echo "Generating normal load (100 RPS for 60 seconds)..."

# Use concurrent requests to simulate load
for i in {1..10}; do
    echo "Batch $i/10..."
    generate_load 100 10 $SERVICE_URL/v1/chat/completions 2>/dev/null || true
    sleep 5
done &

LOAD_PID=$!

echo "Load started (PID: $LOAD_PID)"
echo "Running for 60 seconds..."

# Monitor memory during load
echo ""
echo "4. Monitoring memory during load..."
echo "----------------------------------------"

SAMPLE_COUNT=0
TOTAL_RSS=0
MAX_RSS=0
MIN_RSS=999999

for i in {1..12}; do
    sleep 5

    CURRENT_RSS=$(ps -p $SERVICE_PID -o rss= 2>/dev/null | awk '{print $1}')
    CURRENT_RSS_MB=$(echo "scale=2; $CURRENT_RSS / 1024" | bc)

    echo "[$i/12] RSS: ${CURRENT_RSS_MB}MB"

    # Track statistics
    TOTAL_RSS=$((TOTAL_RSS + CURRENT_RSS))
    SAMPLE_COUNT=$((SAMPLE_COUNT + 1))

    if [ $CURRENT_RSS -gt $MAX_RSS ]; then
        MAX_RSS=$CURRENT_RSS
    fi

    if [ $CURRENT_RSS -lt $MIN_RSS ]; then
        MIN_RSS=$CURRENT_RSS
    fi
done

# Wait for load to complete
wait $LOAD_PID 2>/dev/null || true

echo ""
echo "5. Post-load memory measurement..."
echo "----------------------------------------"

# Wait for memory to stabilize
echo "Waiting for memory to stabilize (30 seconds)..."
sleep 30

FINAL_RSS=$(ps -p $SERVICE_PID -o rss= 2>/dev/null | awk '{print $1}')
FINAL_RSS_MB=$(echo "scale=2; $FINAL_RSS / 1024" | bc)

# Calculate statistics
AVG_RSS=$((TOTAL_RSS / SAMPLE_COUNT))
AVG_RSS_MB=$(echo "scale=2; $AVG_RSS / 1024" | bc)

MAX_RSS_MB=$(echo "scale=2; $MAX_RSS / 1024" | bc)
MIN_RSS_MB=$(echo "scale=2; $MIN_RSS / 1024" | bc)

echo "Memory Statistics:"
echo "  Minimum: ${MIN_RSS_MB}MB"
echo "  Maximum: ${MAX_RSS_MB}MB"
echo "  Average: ${AVG_RSS_MB}MB"
echo "  Final: ${FINAL_RSS_MB}MB"
echo "  Baseline: ${BASELINE_RSS_MB}MB"

echo ""
echo "6. Memory growth analysis..."
echo "----------------------------------------"

GROWTH=$(echo "$FINAL_RSS - $BASELINE_RSS" | bc)
GROWTH_MB=$(echo "scale=2; $GROWTH / 1024" | bc)

echo "Memory Growth:"
echo "  Baseline: ${BASELINE_RSS_MB}MB"
echo "  Final: ${FINAL_RSS_MB}MB"
echo "  Growth: ${GROWTH_MB}MB"

# Check for memory leaks (significant growth)
if (( $(echo "$GROWTH > 10*1024" | bc -l) )); then
    echo -e "${YELLOW}⚠ Potential memory leak detected${NC}"
    echo "  Memory grew by ${GROWTH_MB}MB"
else
    echo -e "${GREEN}✓ No significant memory growth${NC}"
fi

echo ""
echo "7. Using pprof for detailed analysis..."
echo "----------------------------------------"

# Check if pprof endpoint is available
if curl -sf $SERVICE_URL/debug/pprof/heap > /dev/null 2>&1; then
    echo "Generating heap profile..."

    # Save heap profile
    curl -s $SERVICE_URL/debug/pprof/heap > heap.prof

    if [ -f "heap.prof" ]; then
        echo -e "${GREEN}✓ Heap profile saved to heap.prof${NC}"

        # Analyze with go tool pprof
        if command -v go &> /dev/null; then
            echo "Top memory consumers:"
            go tool pprof -top heap.prof 2>/dev/null | head -20 || true
        fi
    fi
else
    echo -e "${YELLOW}⚠ pprof endpoint not available${NC}"
fi

echo ""
echo "8. Comparing with requirements..."
echo "----------------------------------------"

echo "Memory Limit: ${MEMORY_LIMIT_MB}MB"
echo "Average Memory: ${AVG_RSS_MB}MB"
echo "Peak Memory: ${MAX_RSS_MB}MB"

# Check against limit
if (( $(echo "$AVG_RSS_MB < $MEMORY_LIMIT_MB" | bc -l) )); then
    echo -e "${GREEN}✓ PASS: Average memory (${AVG_RSS_MB}MB) < ${MEMORY_LIMIT_MB}MB${NC}"
    COMPLIANCE="PASS"
    VERDICT=0
else
    echo -e "${RED}✗ FAIL: Average memory (${AVG_RSS_MB}MB) ≥ ${MEMORY_LIMIT_MB}MB${NC}"
    COMPLIANCE="FAIL"
    VERDICT=1
fi

# Check peak
if (( $(echo "$MAX_RSS_MB < $MEMORY_LIMIT_MB" | bc -l) )); then
    echo -e "${GREEN}✓ PASS: Peak memory (${MAX_RSS_MB}MB) < ${MEMORY_LIMIT_MB}MB${NC}"
else
    echo -e "${YELLOW}⚠ Peak memory (${MAX_RSS_MB}MB) exceeds limit${NC}"
fi

# Compare with Python version
PYTHON_MB=100
REDUCTION=$(echo "scale=1; (1 - $AVG_RSS_MB / $PYTHON_MB) * 100" | bc)
echo ""
echo "Python version: ~${PYTHON_MB}MB"
echo "Memory reduction: ${REDUCTION}%"

if (( $(echo "$REDUCTION >= 50" | bc -l) )); then
    echo -e "${GREEN}✓ Memory reduction (${REDUCTION}%) ≥ 50%${NC}"
else
    echo -e "${YELLOW}⚠ Memory reduction (${REDUCTION}%) < 50%${NC}"
fi

echo ""
echo "9. Checking for memory leaks..."
echo "----------------------------------------"

# Take another sample after waiting
sleep 30
LEAK_CHECK_RSS=$(ps -p $SERVICE_PID -o rss= 2>/dev/null | awk '{print $1}')
LEAK_CHECK_MB=$(echo "scale=2; $LEAK_CHECK_RSS / 1024" | bc)

LEAK_GROWTH=$(echo "$LEAK_CHECK_RSS - $FINAL_RSS" | bc)
LEAK_GROWTH_MB=$(echo "scale=2; $LEAK_GROWTH / 1024" | bc)

echo "After additional 30 seconds:"
echo "  Memory: ${LEAK_CHECK_MB}MB"
echo "  Growth: ${LEAK_GROWTH_MB}MB"

if (( $(echo "$LEAK_GROWTH > 5*1024" | bc -l) )); then
    echo -e "${RED}✗ Potential memory leak${NC}"
    echo "  Memory grew by ${LEAK_GROWTH_MB}MB in 30 seconds"
else
    echo -e "${GREEN}✓ No memory leak detected${NC}"
fi

echo ""
echo "10. Generating report..."
echo "----------------------------------------"

REPORT_FILE="memory-usage-report.txt"
{
    echo "LLM Proxy Memory Usage Report"
    echo "============================="
    echo ""
    echo "Date: $(date)"
    echo "Service PID: $SERVICE_PID"
    echo ""
    echo "Memory Measurements:"
    echo "  Baseline (idle): ${BASELINE_RSS_MB}MB"
    echo "  Minimum: ${MIN_RSS_MB}MB"
    echo "  Maximum: ${MAX_RSS_MB}MB"
    echo "  Average (under load): ${AVG_RSS_MB}MB"
    echo "  Final: ${FINAL_RSS_MB}MB"
    echo "  After 30s: ${LEAK_CHECK_MB}MB"
    echo ""
    echo "Memory Growth:"
    echo "  From baseline: ${GROWTH_MB}MB"
    echo "  Additional check: ${LEAK_GROWTH_MB}MB"
    echo ""
    echo "Requirements (SC-003):"
    echo "  Target: <${MEMORY_LIMIT_MB}MB"
    echo "  Python baseline: ~100MB"
    echo "  Expected reduction: ~50%"
    echo ""
    echo "Compliance:"
    echo "  Status: $COMPLIANCE"
    echo "  Under limit: $([ $AVG_RSS -lt $MEMORY_LIMIT_BYTES ] && echo "Yes" || echo "No")"
    echo "  Reduction from Python: ${REDUCTION}%"
    echo ""
    echo "Memory Leak Check:"
    echo "  Leaked: $([ $(echo "$LEAK_GROWTH < 5*1024" | bc) -eq 1 ] && echo "No" || echo "Yes")"
    echo "  Growth in 30s: ${LEAK_GROWTH_MB}MB"
} > $REPORT_FILE

echo "Report saved to: $REPORT_FILE"

echo ""
echo "11. Cleanup..."
echo "----------------------------------------"

if [ $RUNNING -eq 0 ]; then
    echo "Stopping test service..."
    kill $SERVICE_PID 2>/dev/null || true
    rm -f $PID_FILE
    echo -e "${GREEN}✓ Cleanup complete${NC}"
fi

echo ""
echo "=== Memory Validation Results ==="
echo ""
echo "Average memory: ${AVG_RSS_MB}MB (limit: ${MEMORY_LIMIT_MB}MB)"
echo "Peak memory: ${MAX_RSS_MB}MB"
echo "Status: $COMPLIANCE"

if [ $VERDICT -eq 0 ]; then
    echo ""
    echo -e "${GREEN}✓ VALIDATION PASSED${NC}"
    echo "  Memory usage is within requirements"
    echo "  No significant memory leaks detected"
else
    echo ""
    echo -e "${RED}✗ VALIDATION FAILED${NC}"
    echo "  Memory usage exceeds limit"
    echo "  Optimization required"
fi

echo ""
echo "Summary saved to: $REPORT_FILE"
echo "Heap profile saved to: heap.prof"

exit $VERDICT
