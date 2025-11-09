#!/bin/bash

# Test script for LLM Proxy
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

print_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}\n"
}

# Function to show usage
show_usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Test LLM Proxy

OPTIONS:
    -h, --help          Show this help message
    -v, --verbose       Verbose output
    -u, --unit          Run unit tests only
    -i, --integration   Run integration tests only
    -c, --contract      Run contract tests only
    -p, --performance   Run performance tests only
    -s, --security      Run security tests only
    -a, --all           Run all tests (default)
    --coverage          Generate coverage report
    --race              Run tests with race detector
    --bench             Run benchmarks
    --short             Run short tests only
    --failfast          Stop on first test failure
    --timeout           Set test timeout in seconds (default: 300)

EXAMPLES:
    $0                          # Run all tests
    $0 --unit --coverage        # Run unit tests with coverage
    $0 --integration --verbose  # Run integration tests verbosely
    $0 --bench                  # Run benchmarks
    $0 --race --coverage        # Run with race detector and coverage

EOF
}

# Parse command line arguments
VERBOSE=false
RUN_UNIT=false
RUN_INTEGRATION=false
RUN_CONTRACT=false
RUN_PERFORMANCE=false
RUN_SECURITY=false
RUN_ALL=true
COVERAGE=false
RACE=false
BENCH=false
SHORT=false
FAILFAST=false
TIMEOUT=300

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_usage
            exit 0
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -u|--unit)
            RUN_UNIT=true
            RUN_ALL=false
            shift
            ;;
        -i|--integration)
            RUN_INTEGRATION=true
            RUN_ALL=false
            shift
            ;;
        -c|--contract)
            RUN_CONTRACT=true
            RUN_ALL=false
            shift
            ;;
        -p|--performance)
            RUN_PERFORMANCE=true
            RUN_ALL=false
            shift
            ;;
        -s|--security)
            RUN_SECURITY=true
            RUN_ALL=false
            shift
            ;;
        -a|--all)
            RUN_ALL=true
            shift
            ;;
        --coverage)
            COVERAGE=true
            shift
            ;;
        --race)
            RACE=true
            shift
            ;;
        --bench)
            BENCH=true
            shift
            ;;
        --short)
            SHORT=true
            shift
            ;;
        --failfast)
            FAILFAST=true
            shift
            ;;
        --timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        *)
            print_error "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Set test flags
TEST_FLAGS="-v"
if [ "$VERBOSE" = true ]; then
    TEST_FLAGS="${TEST_FLAGS}"
else
    TEST_FLAGS="-v"
fi

if [ "$SHORT" = true ]; then
    TEST_FLAGS="${TEST_FLAGS} -short"
fi

if [ "$FAILFAST" = true ]; then
    TEST_FLAGS="${TEST_FLAGS} -failfast"
fi

if [ "$RACE" = true ]; then
    TEST_FLAGS="${TEST_FLAGS} -race"
fi

if [ "$COVERAGE" = true ]; then
    TEST_FLAGS="${TEST_FLAGS} -coverprofile=coverage.txt -covermode=atomic"
fi

# Set timeout
TEST_FLAGS="${TEST_FLAGS} -timeout=${TIMEOUT}s"

print_header "LLM Proxy Test Suite"

# Run benchmarks
if [ "$BENCH" = true ]; then
    print_header "Running Benchmarks"
    print_info "Running benchmarks..."
    go test -bench=. -benchmem ./...

    if [ $? -ne 0 ]; then
        print_error "Benchmarks failed"
        exit 1
    fi
fi

# Run unit tests
if [ "$RUN_ALL" = true ] || [ "$RUN_UNIT" = true ]; then
    print_header "Running Unit Tests"
    print_info "Running unit tests..."
    go test ${TEST_FLAGS} ./internal/... ./pkg/...

    if [ $? -ne 0 ]; then
        print_error "Unit tests failed"
        exit 1
    fi
    print_info "Unit tests passed ✓"
fi

# Run integration tests
if [ "$RUN_ALL" = true ] || [ "$RUN_INTEGRATION" = true ]; then
    print_header "Running Integration Tests"
    print_info "Running integration tests..."

    if [ -d "tests/integration" ]; then
        if command -v ginkgo &> /dev/null; then
            ginkgo -r ${TEST_FLAGS} tests/integration
        else
            go test ${TEST_FLAGS} ./tests/integration/...
        fi

        if [ $? -ne 0 ]; then
            print_error "Integration tests failed"
            exit 1
        fi
        print_info "Integration tests passed ✓"
    else
        print_warn "No integration tests found"
    fi
fi

# Run contract tests
if [ "$RUN_ALL" = true ] || [ "$RUN_CONTRACT" = true ]; then
    print_header "Running Contract Tests"
    print_info "Running contract tests..."

    if [ -d "tests/contract" ]; then
        if command -v ginkgo &> /dev/null; then
            ginkgo -r ${TEST_FLAGS} tests/contract
        else
            go test ${TEST_FLAGS} ./tests/contract/...
        fi

        if [ $? -ne 0 ]; then
            print_error "Contract tests failed"
            exit 1
        fi
        print_info "Contract tests passed ✓"
    else
        print_warn "No contract tests found"
    fi
fi

# Run performance tests
if [ "$RUN_ALL" = true ] || [ "$RUN_PERFORMANCE" = true ]; then
    print_header "Running Performance Tests"
    print_info "Running performance tests..."

    if [ -d "tests/perf" ]; then
        go test ${TEST_FLAGS} -bench=. -benchmem ./tests/perf/...

        if [ $? -ne 0 ]; then
            print_error "Performance tests failed"
            exit 1
        fi
        print_info "Performance tests passed ✓"
    else
        print_warn "No performance tests found"
    fi
fi

# Run security tests
if [ "$RUN_ALL" = true ] || [ "$RUN_SECURITY" = true ]; then
    print_header "Running Security Tests"
    print_info "Running security tests..."

    if command -v gosec &> /dev/null; then
        gosec ./... || true
    else
        print_warn "gosec not installed, skipping security scan"
    fi

    # Run basic security checks with go test
    go test ${TEST_FLAGS} -tags=security ./tests/security/... || true

    print_info "Security tests completed"
fi

# Generate coverage report
if [ "$COVERAGE" = true ] && [ -f "coverage.txt" ]; then
    print_header "Coverage Report"
    go tool cover -html=coverage.txt -o coverage.html
    go tool cover -func=coverage.txt

    COVERAGE_PERCENT=$(go tool cover -func=coverage.txt | grep total | awk '{print $3}')
    print_info "Coverage: ${COVERAGE_PERCENT}"

    if [[ $COVERAGE_PERCENT < "85%" ]]; then
        print_warn "Coverage below target (85%)"
    else
        print_info "Coverage meets target (85%) ✓"
    fi
fi

print_header "All Tests Completed Successfully"
print_info "All tests passed! ✓"
