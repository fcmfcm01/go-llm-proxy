#!/bin/bash

# Code Coverage Verification Script
# Verifies ≥85% overall coverage and ≥95% for critical paths
# Part of T174 - Coverage validation (SC-006)

set -e

echo "=== LLM Proxy Coverage Analysis ==="
echo "Date: $(date)"
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

COVERAGE_THRESHOLD=85
CRITICAL_THRESHOLD=95

echo "Coverage Requirements:"
echo "  Overall: ≥${COVERAGE_THRESHOLD}%"
echo "  Critical Paths: ≥${CRITICAL_THRESHOLD}%"
echo ""

# Run tests with coverage
echo "1. Running tests with coverage..."
echo "----------------------------------------"

go test -coverprofile=coverage.out \
        -covermode=atomic \
        -v ./... 2>&1 | tee test-output.log

echo ""
echo "2. Generating coverage report..."
echo "----------------------------------------"

# Generate HTML report
go tool cover -html=coverage.out -o coverage.html

# Generate coverage by function
go tool cover -func=coverage.out -o coverage-by-function.txt

# Display coverage summary
echo "Coverage Summary:"
echo "----------------------------------------"
go tool cover -func=coverage.out

echo ""
echo "3. Analyzing coverage by package..."
echo "----------------------------------------"

# Parse coverage output for each package
echo "Package Coverage:" > coverage-report.txt
echo "Date: $(date)" >> coverage-report.txt
echo "" >> coverage-report.txt
echo "=== Coverage Analysis ===" >> coverage-report.txt
echo "" >> coverage-report.txt

# Get overall coverage
OVERALL_COVERAGE=$(go tool cover -func=coverage.out | tail -1 | awk '{print $3}')

# Remove % sign and convert to number
OVERALL_NUM=$(echo $OVERALL_COVERAGE | sed 's/%//')

echo "Overall Coverage: $OVERALL_COVERAGE" | tee -a coverage-report.txt
echo "" >> coverage-report.txt

# Check if overall coverage meets threshold
if (( $(echo "$OVERALL_NUM >= $COVERAGE_THRESHOLD" | bc -l) )); then
    echo -e "${GREEN}✓ Overall coverage: $OVERALL_COVERAGE (≥${COVERAGE_THRESHOLD}%)${NC}"
else
    echo -e "${RED}✗ Overall coverage: $OVERALL_COVERAGE (<${COVERAGE_THRESHOLD}%)${NC}"
    echo "FAILED: Overall coverage below threshold"
    echo ""
    echo "Packages with low coverage:"
    go tool cover -func=coverage.out | awk -v threshold=$COVERAGE_THRESHOLD '
        $3 ~ /[0-9]+\.[0-9]+%/ {
            gsub(/%/, "", $3);
            if ($3 < threshold) print $1, $2, $3"%"
        }' | head -20
fi

echo ""
echo "4. Critical Path Coverage Analysis..."
echo "----------------------------------------"

# Define critical packages/modules
CRITICAL_PACKAGES=(
    "internal/auth"
    "internal/converter"
    "internal/proxy"
    "internal/config"
    "internal/server/handlers"
)

echo "Critical Packages:" >> coverage-report.txt
echo "==================" >> coverage-report.txt

for pkg in "${CRITICAL_PACKAGES[@]}"; do
    echo "Checking: $pkg"
    echo "" >> coverage-report.txt

    # Get coverage for this package
    PKG_COVERAGE=$(go tool cover -func=coverage.out | grep "$pkg" | awk '{print $3}')

    if [ -n "$PKG_COVERAGE" ]; then
        PKG_NUM=$(echo $PKG_COVERAGE | sed 's/%//')
        echo "  $pkg: $PKG_COVERAGE" | tee -a coverage-report.txt

        if (( $(echo "$PKG_NUM >= $CRITICAL_THRESHOLD" | bc -l) )); then
            echo -e "    ${GREEN}✓ PASS${NC}"
        else
            echo -e "    ${RED}✗ FAIL${NC}"
            echo "    FAILED: Coverage below ${CRITICAL_THRESHOLD}%"
        fi
    else
        echo -e "    ${YELLOW}⚠ No coverage data found${NC}"
    fi
done

echo ""
echo "5. Coverage by File..."
echo "----------------------------------------"

echo "" >> coverage-report.txt
echo "Detailed File Coverage:" >> coverage-report.txt
echo "=======================" >> coverage-report.txt

# Show files with lowest coverage
echo "Files needing attention (lowest coverage):" >> coverage-report.txt
go tool cover -func=coverage.out | sort -k3 -n | head -20 >> coverage-report.txt

echo ""
echo "Top files by coverage (best):"
go tool cover -func=coverage.out | sort -k3 -nr | head -10

echo ""
echo "Files needing attention (lowest):"
go tool cover -func=coverage.out | sort -k3 -n | head -10

echo ""
echo "6. Uncovered Functions..."
echo "----------------------------------------"

echo "Functions with no coverage:" > uncovered-functions.txt
go tool cover -func=coverage.out | grep "0.0%" >> uncovered-functions.txt

if [ -s uncovered-functions.txt ]; then
    echo -e "${YELLOW}⚠ Found functions with 0% coverage${NC}"
    echo "Review uncovered-functions.txt for details"
    echo ""
    echo "Total uncovered functions: $(wc -l < uncovered-functions.txt)"
else
    echo -e "${GREEN}✓ No functions with 0% coverage${NC}"
fi

echo ""
echo "7. Test Quality Analysis..."
echo "----------------------------------------"

# Count test files
TEST_FILES=$(find . -name "*_test.go" -type f | wc -l)
echo "Total test files: $TEST_FILES"

# Count test functions
TEST_FUNCTIONS=$(grep -r "^func Test" . --include="*_test.go" | wc -l)
echo "Total test functions: $TEST_FUNCTIONS"

# Count test suites
TEST_SUITES=$(find tests/ -name "*_test.go" -type f | wc -l)
echo "Test suites: $TEST_SUITES"

echo ""
echo "Test Distribution:" >> coverage-report.txt
echo "==================" >> coverage-report.txt
echo "Total test files: $TEST_FILES" >> coverage-report.txt
echo "Total test functions: $TEST_FUNCTIONS" >> coverage-report.txt
echo "" >> coverage-report.txt

# Test categories
echo "Test Categories:" >> coverage-report.txt
echo "  Unit tests: $(find tests/unit -name "*_test.go" -type f 2>/dev/null | wc -l || echo 0)" >> coverage-report.txt
echo "  Integration tests: $(find tests/integration -name "*_test.go" -type f 2>/dev/null | wc -l || echo 0)" >> coverage-report.txt
echo "  Contract tests: $(find tests/contract -name "*_test.go" -type f 2>/dev/null | wc -l || echo 0)" >> coverage-report.txt
echo "  E2E tests: $(find tests/e2e -name "*_test.go" -type f 2>/dev/null | wc -l || echo 0)" >> coverage-report.txt

echo ""
echo "8. Coverage HTML Report..."
echo "----------------------------------------"

if [ -f "coverage.html" ]; then
    echo -e "${GREEN}✓ HTML coverage report generated: coverage.html${NC}"
    echo "  Open in browser to view detailed coverage"
else
    echo -e "${RED}✗ Failed to generate HTML coverage report${NC}"
fi

echo ""
echo "=== Coverage Verification Results ==="
echo ""

# Final verdict
VERDICT=0

if (( $(echo "$OVERALL_NUM >= $COVERAGE_THRESHOLD" | bc -l) )); then
    echo -e "${GREEN}✓ Overall coverage PASS: $OVERALL_COVERAGE (≥${COVERAGE_THRESHOLD}%)${NC}"
else
    echo -e "${RED}✗ Overall coverage FAIL: $OVERALL_COVERAGE (<${COVERAGE_THRESHOLD}%)${NC}"
    echo "  FAILED"
    VERDICT=1
fi

# Check critical paths
echo ""
echo "Critical Path Coverage:"
for pkg in "${CRITICAL_PACKAGES[@]}"; do
    PKG_COVERAGE=$(go tool cover -func=coverage.out | grep "$pkg" | awk '{print $3}')
    if [ -n "$PKG_COVERAGE" ]; then
        PKG_NUM=$(echo $PKG_COVERAGE | sed 's/%//')
        if (( $(echo "$PKG_NUM >= $CRITICAL_THRESHOLD" | bc -l) )); then
            echo -e "  ${GREEN}✓${NC} $pkg: $PKG_COVERAGE"
        else
            echo -e "  ${RED}✗${NC} $pkg: $PKG_COVERAGE (FAIL)"
            VERDICT=1
        fi
    fi
done

echo ""
echo "=== Report Files ==="
echo "coverage.html - Interactive HTML report"
echo "coverage-by-function.txt - Function-by-function coverage"
echo "coverage-report.txt - This analysis report"
echo "uncovered-functions.txt - Functions with 0% coverage"
echo ""

# Save summary to file
{
    echo "LLM Proxy Coverage Verification"
    echo "==============================="
    echo ""
    echo "Date: $(date)"
    echo "Overall Coverage: $OVERALL_COVERAGE"
    echo "Threshold: ${COVERAGE_THRESHOLD}%"
    echo ""
    echo "Status: $([ $VERDICT -eq 0 ] && echo "PASS" || echo "FAIL")"
    echo ""
} > coverage-summary.txt

# Exit with appropriate code
exit $VERDICT
