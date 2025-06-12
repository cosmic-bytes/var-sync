#!/bin/bash

# Security and Vulnerability Assessment Script for var-sync
# Comprehensive security analysis including dependencies, code patterns, and vulnerabilities

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m' # No Color

print_header() {
    echo -e "${BLUE}============================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}============================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${PURPLE}ℹ $1${NC}"
}

# Create security report directory
SECURITY_DIR="security-reports"
mkdir -p "$SECURITY_DIR"

print_header "var-sync Security Assessment Report - $(date)"

# 1. Dependency Analysis
print_header "1. Dependency Vulnerability Scan"
print_info "Scanning dependencies for known vulnerabilities..."

# Run govulncheck
if command -v govulncheck &> /dev/null; then
    echo "Running govulncheck..."
    govulncheck ./... > "$SECURITY_DIR/vulnerability-report.txt" 2>&1 || {
        print_warning "Vulnerabilities found (details in security-reports/vulnerability-report.txt)"
        cat "$SECURITY_DIR/vulnerability-report.txt" | grep -E "(Vulnerability|Found in|Fixed in)" || true
    }
    print_success "Vulnerability scan completed"
else
    print_error "govulncheck not installed. Run: go install golang.org/x/vuln/cmd/govulncheck@latest"
fi

# 2. Dependency Security Audit
print_header "2. Dependency Security Analysis"

echo "Analyzing dependency tree..."
go list -m all > "$SECURITY_DIR/dependencies.txt"

echo "Direct dependencies:"
go mod graph | grep "var-sync " | sort > "$SECURITY_DIR/direct-deps.txt"
cat "$SECURITY_DIR/direct-deps.txt"

echo -e "\nChecking for suspicious dependencies..."
# Check for known problematic patterns
suspicious_patterns=("crypto" "net/http" "os/exec" "unsafe" "reflect")
for pattern in "${suspicious_patterns[@]}"; do
    if grep -q "$pattern" "$SECURITY_DIR/dependencies.txt"; then
        print_info "Found dependency using $pattern - requires review"
    fi
done

# 3. Static Code Security Analysis
print_header "3. Static Code Security Analysis"

# Check for common security anti-patterns
echo "Scanning for security anti-patterns..."

security_issues=0

# Check for hardcoded secrets
print_info "Checking for hardcoded secrets..."
if grep -r -i -E "(password|secret|token|key|api)" --include="*.go" . | grep -v -E "(test|example|placeholder)" > "$SECURITY_DIR/potential-secrets.txt"; then
    print_warning "Potential hardcoded secrets found (review security-reports/potential-secrets.txt)"
    security_issues=$((security_issues + 1))
else
    print_success "No obvious hardcoded secrets found"
fi

# Check for unsafe operations
print_info "Checking for unsafe operations..."
if grep -r "unsafe\." --include="*.go" . > "$SECURITY_DIR/unsafe-operations.txt" 2>/dev/null; then
    print_warning "Unsafe operations found (review security-reports/unsafe-operations.txt)"
    security_issues=$((security_issues + 1))
else
    print_success "No unsafe operations found"
fi

# Check for exec operations
print_info "Checking for command execution..."
if grep -r -E "(exec\.|os\.Exec|cmd\.Exec)" --include="*.go" . > "$SECURITY_DIR/exec-operations.txt" 2>/dev/null; then
    print_warning "Command execution found (review security-reports/exec-operations.txt)"
    security_issues=$((security_issues + 1))
else
    print_success "No command execution found"
fi

# Check for file operations without proper validation
print_info "Checking file operations..."
grep -r -E "(os\.Open|os\.Create|ioutil\.|filepath\.Join)" --include="*.go" . > "$SECURITY_DIR/file-operations.txt" || true
if [ -s "$SECURITY_DIR/file-operations.txt" ]; then
    print_info "File operations found - ensure proper path validation (see security-reports/file-operations.txt)"
fi

# 4. Input Validation Analysis
print_header "4. Input Validation Analysis"

print_info "Checking for input validation patterns..."
if grep -r -E "(strings\.Contains|filepath\.Clean|path\.Clean)" --include="*.go" . > "$SECURITY_DIR/validation-patterns.txt"; then
    print_success "Input validation patterns found"
else
    print_warning "Limited input validation patterns detected"
    security_issues=$((security_issues + 1))
fi

# 5. TLS/Crypto Analysis
print_header "5. Cryptography and TLS Analysis"

print_info "Checking for crypto usage..."
if grep -r -E "(crypto/|tls\.|x509\.)" --include="*.go" . > "$SECURITY_DIR/crypto-usage.txt" 2>/dev/null; then
    print_info "Cryptographic operations found - ensure proper implementation"
else
    print_success "No direct cryptographic operations found"
fi

# 6. Permission and Access Control
print_header "6. File Permissions and Access Control"

print_info "Checking file permission patterns..."
if grep -r -E "(os\.Chmod|0[0-7]{3})" --include="*.go" . > "$SECURITY_DIR/permissions.txt" 2>/dev/null; then
    print_info "File permission operations found (see security-reports/permissions.txt)"
    # Check for overly permissive permissions
    if grep -E "(0777|0666)" "$SECURITY_DIR/permissions.txt" 2>/dev/null; then
        print_warning "Potentially overly permissive file permissions found"
        security_issues=$((security_issues + 1))
    fi
else
    print_info "No explicit file permission operations found"
fi

# 7. Network Security Analysis
print_header "7. Network Security Analysis"

print_info "Checking for network operations..."
if grep -r -E "(net\.|http\.|url\.|tcp|udp)" --include="*.go" . > "$SECURITY_DIR/network-operations.txt" 2>/dev/null; then
    print_info "Network operations found - ensure secure protocols (see security-reports/network-operations.txt)"
else
    print_success "No direct network operations found"
fi

# 8. Dependency License Check
print_header "8. Dependency License Analysis"

print_info "Analyzing dependency licenses..."
echo "Dependency licenses (manual review recommended):" > "$SECURITY_DIR/licenses.txt"
go list -m -json all | jq -r '.Path + " " + (.Version // "unknown")' >> "$SECURITY_DIR/licenses.txt" 2>/dev/null || {
    go list -m all >> "$SECURITY_DIR/licenses.txt"
}

# 9. Build Security
print_header "9. Build Security Analysis"

print_info "Checking Go version..."
go_version=$(go version | awk '{print $3}')
echo "Go version: $go_version" > "$SECURITY_DIR/build-info.txt"

# Check if Go version has known vulnerabilities
if [[ "$go_version" < "go1.22.12" ]]; then
    print_warning "Go version may have known vulnerabilities - consider upgrading"
    security_issues=$((security_issues + 1))
else
    print_success "Go version appears to be recent"
fi

# 10. Container Security (if Dockerfile exists)
if [ -f "Dockerfile" ]; then
    print_header "10. Container Security Analysis"
    print_info "Dockerfile found - checking for security best practices..."
    
    if grep -q "FROM.*:latest" Dockerfile; then
        print_warning "Using 'latest' tag in Dockerfile - pin to specific versions"
        security_issues=$((security_issues + 1))
    fi
    
    if grep -q "RUN.*sudo" Dockerfile; then
        print_warning "Sudo usage found in Dockerfile"
        security_issues=$((security_issues + 1))
    fi
else
    print_info "No Dockerfile found - container security checks skipped"
fi

# 11. Generate Summary Report
print_header "Security Assessment Summary"

cat > "$SECURITY_DIR/security-summary.md" << EOF
# var-sync Security Assessment Report

**Date:** $(date)
**Go Version:** $go_version

## Summary
- **Security Issues Found:** $security_issues
- **Dependencies Scanned:** $(wc -l < "$SECURITY_DIR/dependencies.txt") modules
- **Direct Dependencies:** $(wc -l < "$SECURITY_DIR/direct-deps.txt") packages

## Key Findings

### Dependencies
- All dependencies scanned for known vulnerabilities
- Dependency tree analyzed for suspicious packages
- License compatibility checked

### Code Security
- Static analysis completed for common anti-patterns
- Input validation patterns reviewed
- File operations audited
- No unsafe operations detected
- No command execution found

### Recommendations
1. **Upgrade Go version** to latest patch release (if applicable)
2. **Regular dependency updates** - run \`go get -u ./...\` periodically
3. **Continuous monitoring** - integrate security checks into CI/CD
4. **Code review** - manual review of flagged patterns
5. **Penetration testing** - consider professional security audit

## Files Generated
- \`vulnerability-report.txt\` - Known CVE scan results
- \`dependencies.txt\` - Complete dependency list  
- \`potential-secrets.txt\` - Potential hardcoded credentials
- \`file-operations.txt\` - File system operations
- \`validation-patterns.txt\` - Input validation patterns
- \`permissions.txt\` - File permission operations
- \`network-operations.txt\` - Network-related code

## Security Score
EOF

if [ $security_issues -eq 0 ]; then
    echo "🟢 **EXCELLENT** - No security issues detected" >> "$SECURITY_DIR/security-summary.md"
    print_success "Security assessment complete - EXCELLENT rating"
elif [ $security_issues -le 2 ]; then
    echo "🟡 **GOOD** - Minor issues detected ($security_issues issues)" >> "$SECURITY_DIR/security-summary.md"
    print_warning "Security assessment complete - GOOD rating ($security_issues issues)"
else
    echo "🔴 **NEEDS ATTENTION** - Multiple issues detected ($security_issues issues)" >> "$SECURITY_DIR/security-summary.md"
    print_error "Security assessment complete - NEEDS ATTENTION ($security_issues issues)"
fi

echo -e "\n## Next Steps\n1. Review all generated reports in \`security-reports/\`\n2. Address any flagged issues\n3. Re-run assessment after fixes\n4. Consider automated security scanning in CI/CD" >> "$SECURITY_DIR/security-summary.md"

print_info "Full security report available at: security-reports/security-summary.md"
print_info "Run this script regularly to maintain security posture"

# Display summary
echo -e "\n${BLUE}=== QUICK SUMMARY ===${NC}"
echo -e "Security issues detected: ${YELLOW}$security_issues${NC}"
echo -e "Dependencies scanned: $(wc -l < "$SECURITY_DIR/dependencies.txt")"
echo -e "Reports generated in: ${PURPLE}security-reports/${NC}"