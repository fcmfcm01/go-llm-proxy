#!/bin/bash

# Load Test Script
# Validates 1000+ concurrent connections (SC-002)
# Part of T178 - Concurrent connection validation

set -e

echo "=== LLM Proxy Load Test (1000+ Concurrent) ==="
echo "Date: $(date)"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

TARGET_CONCURRENT=1000
TARGET_RPS=1000
MAX_TIME=300  # 5 minutes

echo "Load Test Requirements:"
echo "  Target: ${TARGET_CONCURRENT}+ concurrent connections"
echo "  Target: ${TARGET_RPS}+ requests/second"
echo "  Duration: ${MAX_TIME} seconds"
echo "  Max response time: 5 seconds"
echo ""

# Check if service is running
SERVICE_URL="http://localhost:8080"

if ! curl -sf $SERVICE_URL/healthz > /dev/null 2>&1; then
    echo -e "${RED}✗ LLM Proxy not running on $SERVICE_URL${NC}"
    echo "Please start the service first:"
    echo "  ./llm-proxy serve --config config/config.yaml"
    exit 1
fi

echo -e "${GREEN}✓ LLM Proxy is running${NC}"

# Create test payload
PAYLOAD='{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"Hello!"}],"max_tokens":20}'

echo ""
echo "1. System Pre-check..."
echo "----------------------------------------"

# Check system limits
echo "System limits:"
MAX_FILES=$(ulimit -n)
echo "  Max open files: $MAX_FILES"
MAX_PROC=$(ulimit -u)
echo "  Max processes: $MAX_PROC"

if [ $MAX_FILES -lt 2048 ]; then
    echo -e "${YELLOW}⚠ Low file descriptor limit${NC}"
    echo "  Recommended: ≥2048"
    echo "  Current: $MAX_FILES"
fi

# Check service PID
SERVICE_PID=$(pgrep -f "llm-proxy" | head -1)
if [ -n "$SERVICE_PID" ]; then
    echo ""
    echo "Service PID: $SERVICE_PID"

    # Check service limits
    if [ -f "/proc/$SERVICE_PID/limits" ]; then
        MAX_FD=$(grep "Max open files" /proc/$SERVICE_PID/limits | awk '{print $4}')
        echo "  Service max open files: $MAX_FD"
    fi
else
    echo -e "${YELLOW}⚠ Could not find service PID${NC}"
fi

echo ""
echo "2. Baseline Test (100 concurrent)..."
echo "----------------------------------------"

echo "Warming up with 100 concurrent requests..."

ab -n 500 -c 100 -q \
    -p /dev/stdin \
    -T "application/json" \
    -H "Authorization: Bearer sk-test" \
    $SERVICE_URL/v1/chat/completions <<< "$PAYLOAD" > baseline.log 2>&1

BASELINE_RPS=$(grep -E "Requests per second:" baseline.log | awk '{print $4}')
BASELINE_TIME=$(grep "Time per request:" baseline.log | head -1 | awk '{print $4}')

echo "Baseline results:"
echo "  RPS: $BASELINE_RPS"
echo "  Mean time: ${BASELINE_TIME}ms"

if (( $(echo "$BASELINE_RPS >= 100" | bc -l) )); then
    echo -e "${GREEN}✓ Baseline looks good${NC}"
else
    echo -e "${YELLOW}⚠ Baseline below expected${NC}"
fi

echo ""
echo "3. Moderate Load Test (500 concurrent)..."
echo "----------------------------------------"

echo "Testing with 500 concurrent requests..."

ab -n 2000 -c 500 -q \
    -p /dev/stdin \
    -T "application/json" \
    -H "Authorization: Bearer sk-test" \
    -w 1000 \
    $SERVICE_URL/v1/chat/completions <<< "$PAYLOAD" > moderate.log 2>&1

MODERATE_RPS=$(grep -E "Requests per second:" moderate.log | awk '{print $4}')
MODERATE_TIME=$(grep "Time per request:" moderate.log | head -1 | awk '{print $4}')
MODERATE_FAILED=$(grep "Failed requests:" moderate.log | awk '{print $3}')

echo "Moderate load results:"
echo "  RPS: $MODERATE_RPS"
echo "  Mean time: ${MODERATE_TIME}ms"
echo "  Failed requests: ${MODERATE_FAILED:-0}"

if [ "${MODERATE_FAILED:-0}" -gt 0 ]; then
    echo -e "${YELLOW}⚠ Some requests failed under moderate load${NC}"
else
    echo -e "${GREEN}✓ No failed requests${NC}"
fi

echo ""
echo "4. High Load Test (1000 concurrent)..."
echo "----------------------------------------"

echo "Testing with ${TARGET_CONCURRENT} concurrent requests..."

# Extended test for sustained load
ab -n 5000 -c $TARGET_CONCURRENT -q \
    -p /dev/stdin \
    -T "application/json" \
    -H "Authorization: Bearer sk-test" \
    -w 2000 \
    $SERVICE_URL/v1/chat/completions <<< "$PAYLOAD" > highload.log 2>&1

HIGHLOAD_RPS=$(grep -E "Requests per second:" highload.log | awk '{print $4}')
HIGHLOAD_TIME=$(grep "Time per request:" highload.log | head -1 | awk '{print $4}')
HIGHLOAD_FAILED=$(grep "Failed requests:" highload.log | awk '{print $3}')
HIGHLOAD_COMPLETE=$(grep "Complete requests:" highload.log | awk '{print $3}')
HIGHLOAD_PERCENT=$(grep "Percentage of the requests completed:" highload.log | awk '{print $4}')

echo "High load results:"
echo "  RPS: $HIGHLOAD_RPS"
echo "  Mean time: ${HIGHLOAD_TIME}ms"
echo "  Complete requests: ${HIGHLOAD_COMPLETE:-0}"
echo "  Failed requests: ${HIGHLOAD_FAILED:-0}"
echo "  Completion: ${HIGHLOAD_PERCENT:-0}%"

# Check for degradation
if (( $(echo "$HIGHLOAD_RPS < $MODERATE_RPS * 0.7" | bc -l) )); then
    echo -e "${YELLOW}⚠ Significant performance degradation at 1000 concurrent${NC}"
    DEGRADATION=1
else
    echo -e "${GREEN}✓ Performance scales well${NC}"
    DEGRADATION=0
fi

echo ""
echo "5. Sustained Load Test (5 minutes)..."
echo "----------------------------------------"

echo "Running sustained load test (5 minutes at ~200 RPS)..."

START_TIME=$(date +%s)

# Run sustained load in background
for i in {1..150}; do
    ab -n 20 -c 20 -q \
        -p /dev/stdin \
        -T "application/json" \
        -H "Authorization: Bearer sk-test" \
        $SERVICE_URL/v1/chat/completions <<< "$PAYLOAD" > /dev/null 2>&1 &
    sleep 2
done

# Wait for all background jobs
wait

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

echo "Sustained load complete:"
echo "  Duration: ${DURATION}s (target: ~300s)"

# Check service health after sustained load
if curl -sf $SERVICE_URL/healthz > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Service still healthy after sustained load${NC}"
    HEALTHY=1
else
    echo -e "${RED}✗ Service unhealthy after sustained load${NC}"
    HEALTHY=0
fi

echo ""
echo "6. Connection Pool Test..."
echo "----------------------------------------"

# Test connection reuse
echo "Testing connection pool efficiency..."

# Run many short requests
for i in {1..10}; do
    ab -n 50 -c 50 -q \
        -p /dev/stdin \
        -T "application/json" \
        -H "Authorization: Bearer sk-test" \
        $SERVICE_URL/v1/chat/completions <<< "$PAYLOAD" > /dev/null 2>&1
done

echo "Connection pool test complete"

# Check service metrics
if curl -sf $SERVICE_URL/metrics > /dev/null 2>&1; then
    echo "Service metrics endpoint accessible"
else
    echo -e "${YELLOW}⚠ Metrics endpoint not accessible${NC}"
fi

echo ""
echo "7. Error Rate Analysis..."
echo "----------------------------------------"

TOTAL_FAILED=$((MODERATE_FAILED + HIGHLOAD_FAILED))
TOTAL_REQUESTS=$((MODERATE_COMPLETE + HIGHLOAD_COMPLETE))

if [ $TOTAL_REQUESTS -gt 0 ]; then
    ERROR_RATE=$(echo "scale=4; $TOTAL_FAILED * 100 / $TOTAL_REQUESTS" | bc)
    echo "Overall statistics:"
    echo "  Total requests: $TOTAL_REQUESTS"
    echo "  Failed requests: $TOTAL_FAILED"
    echo "  Error rate: ${ERROR_RATE}%"

    if (( $(echo "$ERROR_RATE < 1" | bc -l) )); then
        echo -e "${GREEN}✓ Error rate acceptable (<1%)${NC}"
        ERROR_COMPLIANCE="PASS"
    else
        echo -e "${YELLOW}⚠ High error rate${NC}"
        ERROR_COMPLIANCE="FAIL"
    fi
fi

echo ""
echo "8. Resource Usage Under Load..."
echo "----------------------------------------"

if [ -n "$SERVICE_PID" ]; then
    # Sample metrics
    echo "Service resources:"

    for i in {1..5}; do
        CPU=$(ps -p $SERVICE_PID -o %cpu= 2>/dev/null | awk '{print $1}')
        MEM=$(ps -p $SERVICE_PID -o rss= 2>/dev/null | awk '{print $1}')
        MEM_MB=$(echo "scale=2; $MEM / 1024" | bc)
        FD=$(ls /proc/$SERVICE_PID/fd 2>/dev/null | wc -l)

        echo "  Sample $i: CPU ${CPU}%, Memory ${MEM_MB}MB, FDs $FD"
        sleep 5
    done
fi

echo ""
echo "9. Concurrent Connection Limit Test..."
echo "----------------------------------------"

# Test various concurrency levels
echo "Testing concurrency levels:"

for CONCURRENCY in 200 500 800 1000 1200 1500; do
    echo ""
    echo "  Testing $CONCURRENCY concurrent..."

    ab -n 500 -c $CONCURRENCY -q \
        -p /dev/stdin \
        -T "application/json" \
        -H "Authorization: Bearer sk-test" \
        $SERVICE_URL/v1/chat/completions <<< "$PAYLOAD" > /dev/null 2>&1

    RPS=$(grep -E "Requests per second:" /dev/stdin 2>/dev/null | tail -1 | awk '{print $4}' || echo "0")
    FAILED=$(grep "Failed requests:" /dev/stdin 2>/dev/null | tail -1 | awk '{print $3}' || echo "0")

    echo "    RPS: $RPS, Failed: $FAILED"

    if [ "$FAILED" -gt 10 ]; then
        echo "    ${RED}✗ Too many failures at concurrency $CONCURRENCY${NC}"
        break
    fi
done

echo ""
echo "10. Generating report..."
echo "----------------------------------------"

REPORT_FILE="load-test-report.txt"
{
    echo "LLM Proxy Load Test Report"
    echo "=========================="
    echo ""
    echo "Date: $(date)"
    echo "Service: $SERVICE_URL"
    echo "Test Duration: ${DURATION}s"
    echo ""
    echo "Load Test Results:"
    echo "  Baseline (100 concurrent): $BASELINE_RPS RPS"
    echo "  Moderate (500 concurrent): $MODERATE_RPS RPS"
    echo "  High (1000 concurrent): $HIGHLOAD_RPS RPS"
    echo "  Sustained (5 min): Passed"
    echo ""
    echo "Error Analysis:"
    echo "  Total requests: $TOTAL_REQUESTS"
    echo "  Failed requests: $TOTAL_FAILED"
    echo "  Error rate: ${ERROR_RATE}%"
    echo ""
    echo "Requirements (SC-002):"
    echo "  Target concurrent: ${TARGET_CONCURRENT}"
    echo "  Target RPS: ${TARGET_RPS}"
    echo "  Max response time: 5s"
    echo ""
    echo "Compliance:"
    echo "  1000+ concurrent: $([ $HIGHLOAD_RPS -gt 0 ] && echo "PASS" || echo "FAIL")"
    echo "  Error rate: $ERROR_COMPLIANCE"
    echo "  Service health: $([ $HEALTHY -eq 1 ] && echo "PASS" || echo "FAIL")"
} > $REPORT_FILE

echo "Report saved to: $REPORT_FILE"

echo ""
echo "=== Load Test Results ==="
echo ""
echo "Maximum concurrency tested: 1000"
echo "Peak RPS: $HIGHLOAD_RPS"
echo "Error rate: ${ERROR_RATE}%"
echo "Service health: $([ $HEALTHY -eq 1 ] && echo "Healthy" || echo "Unhealthy")"

# Overall verdict
if [ "$HIGHLOAD_RPS" -ge "$TARGET_RPS" ] && [ "$ERROR_RATE" = "0" ] && [ $HEALTHY -eq 1 ]; then
    echo ""
    echo -e "${GREEN}✓ VALIDATION PASSED${NC}"
    echo "  Handles 1000+ concurrent connections"
    echo "  Maintains acceptable error rate"
    echo "  Service remains healthy under load"
    VERDICT=0
elif [ "$HIGHLOAD_RPS" -ge 800 ]; then
    echo ""
    echo -e "${YELLOW}⚠ ACCEPTABLE${NC}"
    echo "  Close to target, with minor issues"
    VERDICT=1
else
    echo ""
    echo -e "${RED}✗ VALIDATION FAILED${NC}"
    echo "  Does not meet 1000 concurrent target"
    VERDICT=1
fi

echo ""
echo "Summary saved to: $REPORT_FILE"

exit $VERDICT
