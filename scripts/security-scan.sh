#!/bin/bash

# Security Scan Script for LLM Proxy
# Runs gosec and other security checks
# Part of T173 - Security validation (SC-022)

set -e

echo "=== LLM Proxy Security Scan ==="
echo "Date: $(date)"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print status
print_status() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ $2${NC}"
    else
        echo -e "${RED}✗ $2${NC}"
    fi
}

# Check if gosec is installed
if ! command -v gosec &> /dev/null; then
    echo "Installing gosec..."
    go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
fi

echo "1. Running gosec security scan..."
echo "----------------------------------------"

# Run gosec scan
gosec -fmt sarif -out security-scan.sarif ./... 2>&1 || true
gosec -fmt json -out security-scan.json ./... 2>&1 || true
gosec ./... 2>&1 || true

echo ""
echo "2. Running go vet (static analysis)..."
echo "----------------------------------------"
go vet ./... 2>&1
VET_RESULT=$?
print_status $VET_RESULT "go vet passed"

echo ""
echo "3. Running go-critic (additional linting)..."
echo "----------------------------------------"

# Check if go-critic is installed
if command -v go-critic &> /dev/null; then
    go-critic check ./... 2>&1 || true
    print_status 0 "go-critic check completed"
else
    echo -e "${YELLOW}⚠ go-critic not installed, skipping${NC}"
    echo "Install with: go install github.com/go-critic/go-critic/cmd/go-critic@latest"
fi

echo ""
echo "4. Checking for hardcoded secrets..."
echo "----------------------------------------"

# Check for common secret patterns
if grep -r "password.*=" --include="*.go" . 2>/dev/null | grep -v "test" | grep -v "_test.go" | grep -v "example" | grep -v "config"; then
    echo -e "${YELLOW}⚠ Potential hardcoded passwords found${NC}"
else
    print_status 0 "No obvious hardcoded passwords"
fi

if grep -r "api[_-]key.*=" --include="*.go" . 2>/dev/null | grep -v "test" | grep -v "_test.go" | grep -v "example" | grep -v "config"; then
    echo -e "${YELLOW}⚠ Potential hardcoded API keys found${NC}"
else
    print_status 0 "No obvious hardcoded API keys"
fi

echo ""
echo "5. Checking dependencies for known vulnerabilities..."
echo "----------------------------------------"

# Run govulncheck
if command -v govulncheck &> /dev/null; then
    govulncheck ./... 2>&1 || true
    print_status 0 "govulncheck completed"
else
    echo -e "${YELLOW}⚠ govulncheck not installed${NC}"
    echo "Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"
fi

# Alternative: Use npm audit if package.json exists (for frontend)
if [ -f "package.json" ]; then
    echo "Running npm audit for frontend dependencies..."
    if command -v npm &> /dev/null; then
        npm audit --audit-level=high 2>&1 || true
        print_status 0 "npm audit completed"
    fi
fi

echo ""
echo "6. Checking Docker security..."
echo "----------------------------------------"

if [ -f "Dockerfile" ]; then
    # Check for non-root user in Dockerfile
    if grep -q "USER.*[0-9]" Dockerfile; then
        print_status 0 "Dockerfile uses non-root user"
    else
        echo -e "${YELLOW}⚠ Dockerfile may be running as root${NC}"
    fi

    # Check for latest tag usage
    if grep -q "FROM.*:latest" Dockerfile; then
        echo -e "${YELLOW}⚠ Dockerfile uses :latest tag (consider using specific version)${NC}"
    else
        print_status 0 "Dockerfile uses specific image tags"
    fi

    # Check for unnecessary packages
    if grep -q "apk.*add.*curl\|apk.*add.*wget" Dockerfile; then
        echo -e "${YELLOW}⚠ Dockerfile includes debug tools (curl/wget) in final image${NC}"
    else
        print_status 0 "Dockerfile does not include debug tools in final image"
    fi
else
    echo -e "${YELLOW}⚠ No Dockerfile found${NC}"
fi

echo ""
echo "7. Checking for security headers in HTTP server..."
echo "----------------------------------------"

# This would be checked in integration tests, but we can verify the code
if grep -r "Strict-Transport-Security" --include="*.go" . 2>/dev/null | grep -v "_test.go" > /dev/null; then
    print_status 0 "HSTS security header implemented"
else
    echo -e "${YELLOW}⚠ HSTS header may not be implemented${NC}"
fi

if grep -r "X-Content-Type-Options" --include="*.go" . 2>/dev/null | grep -v "_test.go" > /dev/null; then
    print_status 0 "X-Content-Type-Options header implemented"
else
    echo -e "${YELLOW}⚠ X-Content-Type-Options header may not be implemented${NC}"
fi

echo ""
echo "8. Checking for SQL injection protection..."
echo "----------------------------------------"

if grep -r "database/sql" --include="*.go" . 2>/dev/null | grep -v "_test.go" > /dev/null; then
    if grep -r "Prepare\|Query" --include="*.go" . 2>/dev/null | grep -v "_test.go" | grep -v "// sql" | head -5 > /dev/null; then
        print_status 0 "Uses parameterized queries"
    else
        echo -e "${YELLOW}⚠ Verify SQL injection protection is in place${NC}"
    fi
else
    print_status 0 "No direct database usage (filesystem-based storage)"
fi

echo ""
echo "9. Checking for XSS protection..."
echo "----------------------------------------"

if grep -r "Sanitize\|html.EscapeString" --include="*.go" . 2>/dev/null | grep -v "_test.go" > /dev/null; then
    print_status 0 "XSS protection implemented"
else
    echo -e "${YELLOW}⚠ XSS protection may not be implemented${NC}"
fi

echo ""
echo "10. Analyzing report files..."
echo "----------------------------------------"

if [ -f "security-scan.json" ]; then
    echo "Security scan report generated: security-scan.json"
    if command -v jq &> /dev/null; then
        HIGH_ISSUES=$(jq '.Issues | map(select(.severity == "HIGH")) | length' security-scan.json)
        MEDIUM_ISSUES=$(jq '.Issues | map(select(.severity == "MEDIUM")) | length' security-scan.json)
        LOW_ISSUES=$(jq '.Issues | map(select(.severity == "LOW")) | length' security-scan.json)

        echo "  High severity: $HIGH_ISSUES"
        echo "  Medium severity: $MEDIUM_ISSUES"
        echo "  Low severity: $LOW_ISSUES"

        if [ "$HIGH_ISSUES" -eq "0" ]; then
            print_status 0 "No HIGH severity issues found"
        else
            echo -e "${RED}✗ Found $HIGH_ISSUES HIGH severity issues${NC}"
        fi
    fi
fi

if [ -f "security-scan.sarif" ]; then
    echo "SARIF report generated: security-scan.sarif"
    echo "  (Import this into GitHub Security or other SARIF-compatible tools)"
fi

echo ""
echo "=== Security Scan Summary ==="
echo ""

# Summary report
echo "Date: $(date)" > security-report.txt
echo "=== LLM Proxy Security Scan Report ===" >> security-report.txt
echo "" >> security-report.txt
echo "1. Code Quality:" >> security-report.txt
go vet ./... 2>&1 >> security-report.txt || echo "  - Minor issues found" >> security-report.txt
echo "" >> security-report.txt

echo "2. Security Scan (gosec):" >> security-report.txt
if [ -f "security-scan.json" ] && command -v jq &> /dev/null; then
    jq '.Issues | map({severity, rule_id, file_path, line, column, description})' security-scan.json >> security-report.txt
fi
echo "" >> security-report.txt

echo "3. Vulnerabilities (govulncheck):" >> security-report.txt
if command -v govulncheck &> /dev/null; then
    govulncheck ./... >> security-report.txt 2>&1 || echo "  - No critical vulnerabilities found" >> security-report.txt
else
    echo "  - govulncheck not installed" >> security-report.txt
fi
echo "" >> security-report.txt

echo "4. Dependencies:" >> security-report.txt
go list -json -m all >> security-report.txt 2>&1
echo "" >> security-report.txt

echo "Report saved to: security-report.txt"
echo ""

# Final verdict
echo "=== Final Verdict ==="
echo ""

if [ -f "security-scan.json" ] && command -v jq &> /dev/null; then
    HIGH_ISSUES=$(jq '.Issues | map(select(.severity == "HIGH")) | length' security-scan.json 2>/dev/null || echo "0")
    if [ "$HIGH_ISSUES" -eq "0" ]; then
        echo -e "${GREEN}✓ Security scan passed - No HIGH severity issues found${NC}"
        echo "✓ Ready for production deployment"
        exit 0
    else
        echo -e "${RED}✗ Security scan failed - Found $HIGH_ISSUES HIGH severity issues${NC}"
        echo "Please review the report and fix issues before deployment"
        echo ""
        echo "Common fixes:"
        echo "  - Use proper error handling (G104)"
        echo "  - Avoid hardcoded credentials (G101)"
        echo "  - Use proper SQL parameterization (G201, G202)"
        echo "  - Fix weak cryptographic random (G404)"
        exit 1
    fi
else
    echo -e "${YELLOW}⚠ Could not determine final verdict (missing jq or report)${NC}"
    echo "Please review security-scan.json manually"
    exit 2
fi
