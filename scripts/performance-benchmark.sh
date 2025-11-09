#!/bin/bash

# Performance Benchmark Script
# Validates 25% performance improvement over Python version (SC-004)
# Part of T177 - Performance benchmark validation

set -e

echo "=== LLM Proxy Performance Benchmark ==="
echo "Date: $(date)"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

IMPROVEMENT_TARGET=25
PYTHON_LATENCY=120  # ms (baseline)
GO_LATENCY_TARGET=$(echo "scale=2; $PYTHON_LATENCY * (100 - $IMPROVEMENT_TARGET) / 100" | bc)

echo "Performance Requirements:"
echo "  Target: ${IMPROVEMENT_TARGET}% faster than Python"
echo "  Python baseline: ${PYTHON_LATENCY}ms p95"
echo "  Go target: <${GO_LATENCY_TARGET}ms p95"
echo "  Goal: <1.5s total response time"
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

# Install Apache Bench if needed
if ! command -v ab &> /dev/null; then
    echo "Installing apache2-utils..."
    apt-get update -qq && apt-get install -y -qq apache2-utils > /dev/null 2>&1 || \
    yum install -y -q httpd-tools > /dev/null 2>&1
fi

# Create test payload
PAYLOAD='{"model":"gpt-3.5-turbo","messages":[{"role":"user","content":"Hello! How are you?"}],"max_tokens":50}'
PAYLOAD_SIZE=$(echo -n $PAYLOAD | wc -c)

echo ""
echo "1. Latency Test (Single Request)..."
echo "----------------------------------------"

# Test single request latency
echo "Testing single request latency..."

LATENCIES=()
for i in {1..10}; do
    LATENCY=$(curl -o /dev/null -s -w "%{time_total}\n" \
        -X POST $SERVICE_URL/v1/chat/completions \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer sk-test" \
        -d "$PAYLOAD" | awk '{print $1 * 1000}')  # Convert to ms

    LATENCIES+=($LATENCY)
    echo "  Request $i: ${LATENCY}ms"
done

# Calculate statistics
echo ""
echo "Latency Statistics:"
AVG_LATENCY=$(echo "scale=2; $(IFS=+; echo "$((${LATENCIES[*]}))") / ${#LATENCIES[@]}" | bc)
echo "  Average: ${AVG_LATENCY}ms"

# Sort latencies for percentile calculation
SORTED_LATENCIES=($(printf '%s\n' "${LATENCIES[@]}" | sort -n))
P95_INDEX=$(( ${#SORTED_LATENCIES[@]} * 95 / 100 ))
P99_INDEX=$(( ${#SORTED_LATENCIES[@]} * 99 / 100 ))
P50_LATENCY=${SORTED_LATENCIES[$(( ${#SORTED_LATENCIES[@]} * 50 / 100 ))]}
P95_LATENCY=${SORTED_LATENCIES[$P95_INDEX]}
P99_LATENCY=${SORTED_LATENCIES[$P99_INDEX]}

echo "  Median (p50): ${P50_LATENCY}ms"
echo "  p95: ${P95_LATENCY}ms"
echo "  p99: ${P99_LATENCY}ms"
echo "  Max: ${SORTED_LATENCIES[-1]}ms"

# Check against target
if (( $(echo "$P95_LATENCY < $GO_LATENCY_TARGET" | bc -l) )); then
    echo -e "${GREEN}✓ PASS: p95 latency (${P95_LATENCY}ms) < target (${GO_LATENCY_TARGET}ms)${NC}"
    LATENCY_COMPLIANCE="PASS"
else
    echo -e "${RED}✗ FAIL: p95 latency (${P95_LATENCY}ms) ≥ target (${GO_LATENCY_TARGET}ms)${NC}"
    LATENCY_COMPLIANCE="FAIL"
fi

echo ""
echo "2. Throughput Test (100 requests)..."
echo "----------------------------------------"

# Test throughput
echo "Running 100 requests with concurrency 10..."

START_TIME=$(date +%s%3N)
ab -n 100 -c 10 -q \
    -p /dev/stdin \
    -T "application/json" \
    -H "Authorization: Bearer sk-test" \
    $SERVICE_URL/v1/chat/completions <<< "$PAYLOAD" > throughput.log 2>&1

END_TIME=$(date +%s%3N)
TOTAL_TIME=$((END_TIME - START_TIME))

# Extract metrics from ab output
RPS=$(grep -E "Requests per second:" throughput.log | awk '{print $4}')
MEAN_TIME=$(grep "Time per request:" throughput.log | head -1 | awk '{print $4}')

echo "Results:"
echo "  Total time: ${TOTAL_TIME}ms"
echo "  Requests per second: $RPS"
echo "  Mean time per request: ${MEAN_TIME}ms"

echo ""
echo "3. Concurrent Load Test (50 concurrent)..."
echo "----------------------------------------"

echo "Running 500 requests with concurrency 50..."

ab -n 500 -c 50 -q \
    -p /dev/stdin \
    -T "application/json" \
    -H "Authorization: Bearer sk-test" \
    $SERVICE_URL/v1/chat/completions <<< "$PAYLOAD" > concurrent.log 2>&1

RPS_CONCURRENT=$(grep -E "Requests per second:" concurrent.log | awk '{print $4}')
TIME_CONCURRENT=$(grep "Time per request:" concurrent.log | head -1 | awk '{print $4}')

echo "Results:"
echo "  Requests per second: $RPS_CONCURRENT"
echo "  Mean time per request: ${TIME_CONCURRENT}ms"

# Check for degradation
if (( $(echo "$RPS_CONCURRENT < $RPS * 0.5" | bc -l) )); then
    echo -e "${YELLOW}⚠ Significant performance degradation under load${NC}"
else
    echo -e "${GREEN}✓ Performance consistent under load${NC}"
fi

echo ""
echo "4. Format Conversion Benchmark..."
echo "----------------------------------------"

# Test format conversion performance
if [ -d "tests/benchmark" ]; then
    echo "Running format conversion benchmarks..."
    go test -bench=. ./tests/benchmark/... 2>&1 | tee conversion-bench.log

    echo ""
    echo "Conversion benchmark results saved to conversion-bench.log"
else
    echo -e "${YELLOW}⚠ Benchmark tests not found${NC}"
fi

echo ""
echo "5. Resource Usage During Load..."
echo "----------------------------------------"

# Monitor resource usage
SERVICE_PID=$(pgrep -f "llm-proxy" | head -1)

if [ -n "$SERVICE_PID" ]; then
    echo "Monitoring PID: $SERVICE_PID"

    # Get baseline CPU and memory
    CPU_BEFORE=$(ps -p $SERVICE_PID -o %cpu= 2>/dev/null | awk '{print $1}')
    MEM_BEFORE=$(ps -p $SERVICE_PID -o rss= 2>/dev/null | awk '{print $1}')
    MEM_BEFORE_MB=$(echo "scale=2; $MEM_BEFORE / 1024" | bc)

    # Run load
    ab -n 100 -c 20 -q -p /dev/stdin -T "application/json" \
        -H "Authorization: Bearer sk-test" \
        $SERVICE_URL/v1/chat/completions <<< "$PAYLOAD" > /dev/null 2>&1

    # Get post-load metrics
    CPU_AFTER=$(ps -p $SERVICE_PID -o %cpu= 2>/dev/null | awk '{print $1}')
    MEM_AFTER=$(ps -p $SERVICE_PID -o rss= 2>/dev/null | awk '{print $1}')
    MEM_AFTER_MB=$(echo "scale=2; $MEM_AFTER / 1024" | bc)

    echo "Resource Usage:"
    echo "  CPU: ${CPU_BEFORE}% → ${CPU_AFTER}%"
    echo "  Memory: ${MEM_BEFORE_MB}MB → ${MEM_AFTER_MB}MB"

    # Check for excessive CPU
    if (( $(echo "$CPU_AFTER > 80" | bc -l) )); then
        echo -e "${YELLOW}⚠ High CPU usage: ${CPU_AFTER}%${NC}"
    else
        echo -e "${GREEN}✓ CPU usage acceptable${NC}"
    fi
else
    echo -e "${YELLOW}⚠ Could not find service PID${NC}"
fi

echo ""
echo "6. Comparison with Python Version..."
echo "----------------------------------------"

echo "Performance Comparison:"
echo "  Python p95 latency: ${PYTHON_LATENCY}ms"
echo "  Go p95 latency: ${P95_LATENCY}ms"
echo "  Go target: <${GO_LATENCY_TARGET}ms"

IMPROVEMENT=$(echo "scale=1; (1 - $P95_LATENCY / $PYTHON_LATENCY) * 100" | bc)
echo "  Improvement: ${IMPROVEMENT}%"

if (( $(echo "$IMPROVEMENT >= $IMPROVEMENT_TARGET" | bc -l) )); then
    echo -e "${GREEN}✓ PASS: Improvement (${IMPROVEMENT}%) ≥ ${IMPROVEMENT_TARGET}%${NC}"
    IMPROVEMENT_COMPLIANCE="PASS"
else
    echo -e "${YELLOW}⚠ Improvement (${IMPROVEMENT}%) < ${IMPROVEMENT_TARGET}%${NC}"
    IMPROVEMENT_COMPLIANCE="INSUFFICIENT"
fi

echo ""
echo "7. Response Time Distribution..."
echo "----------------------------------------"

# Create histogram
echo "Response Time Distribution:"
echo ""
for i in {0..10}; do
    THRESHOLD=$((i * 20))
    COUNT=0
    for latency in "${LATENCIES[@]}"; do
        if (( $(echo "$latency < $THRESHOLD" | bc -l) )); then
            COUNT=$((COUNT + 1))
        fi
    done
    PERCENT=$(echo "scale=1; $COUNT * 100 / ${#LATENCIES[@]}" | bc)
    BAR=$(printf "%${PERCENT}s" | tr ' ' '#')
    echo "  <${THRESHOLD}ms: $PERCENT% $BAR"
done

echo ""
echo "8. Generating report..."
echo "----------------------------------------"

REPORT_FILE="performance-benchmark-report.txt"
{
    echo "LLM Proxy Performance Benchmark Report"
    echo "======================================"
    echo ""
    echo "Date: $(date)"
    echo "Service: $SERVICE_URL"
    echo ""
    echo "Latency Metrics:"
    echo "  Average: ${AVG_LATENCY}ms"
    echo "  p50 (median): ${P50_LATENCY}ms"
    echo "  p95: ${P95_LATENCY}ms"
    echo "  p99: ${P99_LATENCY}ms"
    echo "  Max: ${SORTED_LATENCIES[-1]}ms"
    echo ""
    echo "Throughput Metrics:"
    echo "  RPS (single): $RPS"
    echo "  RPS (concurrent): $RPS_CONCURRENT"
    echo "  Mean time: ${MEAN_TIME}ms"
    echo ""
    echo "Comparison:"
    echo "  Python baseline: ${PYTHON_LATENCY}ms"
    echo "  Go measured: ${P95_LATENCY}ms"
    echo "  Target: <${GO_LATENCY_TARGET}ms"
    echo "  Improvement: ${IMPROVEMENT}%"
    echo ""
    echo "Requirements (SC-004):"
    echo "  Target improvement: ${IMPROVEMENT_TARGET}%"
    echo "  Target latency: <${GO_LATENCY_TARGET}ms"
    echo "  Goal: <1.5s total"
    echo ""
    echo "Compliance:"
    echo "  Latency: $LATENCY_COMPLIANCE"
    echo "  Improvement: $IMPROVEMENT_COMPLIANCE"
} > $REPORT_FILE

echo "Report saved to: $REPORT_FILE"

echo ""
echo "=== Performance Benchmark Results ==="
echo ""
echo "Latency (p95): ${P95_LATENCY}ms (target: <${GO_LATENCY_TARGET}ms)"
echo "Throughput: $RPS RPS"
echo "Improvement: ${IMPROVEMENT}% (target: ≥${IMPROVEMENT_TARGET}%)"

# Overall verdict
if [ "$LATENCY_COMPLIANCE" = "PASS" ] && [ "$IMPROVEMENT_COMPLIANCE" = "PASS" ]; then
    echo ""
    echo -e "${GREEN}✓ VALIDATION PASSED${NC}"
    echo "  Performance meets all requirements"
    VERDICT=0
else
    echo ""
    echo -e "${YELLOW}⚼ PARTIAL COMPLIANCE${NC}"
    if [ "$LATENCY_COMPLIANCE" != "PASS" ]; then
        echo "  Latency target not met"
    fi
    if [ "$IMPROVEMENT_COMPLIANCE" != "PASS" ]; then
        echo "  Improvement target not met"
    fi
    VERDICT=1
fi

echo ""
echo "Summary saved to: $REPORT_FILE"

exit $VERDICT
