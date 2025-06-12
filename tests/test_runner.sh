#!/bin/bash

# Test runner script for var-sync
# This script provides comprehensive testing capabilities with detailed reporting

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
TEST_TIMEOUT=${TEST_TIMEOUT:-10m}
COVERAGE_THRESHOLD=${COVERAGE_THRESHOLD:-80}
VERBOSE=${VERBOSE:-false}

# Directories
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_RESULTS_DIR="${PROJECT_ROOT}/test-results"
COVERAGE_DIR="${PROJECT_ROOT}/coverage"

# Create directories
mkdir -p "${TEST_RESULTS_DIR}"
mkdir -p "${COVERAGE_DIR}"

# Functions
print_header() {
    echo -e "${BLUE}============================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}============================================${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

# Check dependencies
check_dependencies() {
    print_header "Checking Dependencies"
    
    # Check Go version
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed"
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    print_info "Go version: ${GO_VERSION}"
    
    # Check required tools
    local tools=("golangci-lint" "gosec" "govulncheck")
    local missing_tools=()
    
    for tool in "${tools[@]}"; do
        if command -v "$tool" &> /dev/null; then
            print_success "$tool is available"
        else
            print_warning "$tool is not installed (will be installed if needed)"
            missing_tools+=("$tool")
        fi
    done
    
    # Install missing tools if needed
    if [[ ${#missing_tools[@]} -gt 0 ]]; then
        print_info "Installing missing tools..."
        for tool in "${missing_tools[@]}"; do
            case $tool in
                "golangci-lint")
                    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
                    ;;
                "gosec")
                    go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
                    ;;
                "govulncheck")
                    go install golang.org/x/vuln/cmd/govulncheck@latest
                    ;;
            esac
        done
    fi
}

# Install dependencies
install_deps() {
    print_header "Installing Dependencies"
    go mod download
    go mod tidy
    print_success "Dependencies installed"
}

# Format code
format_code() {
    print_header "Formatting Code"
    
    # Run go fmt
    if go fmt ./...; then
        print_success "Code formatted with go fmt"
    else
        print_error "Failed to format code with go fmt"
        return 1
    fi
    
    # Run goimports if available
    if command -v goimports &> /dev/null; then
        if goimports -w .; then
            print_success "Imports formatted with goimports"
        else
            print_warning "Failed to format imports with goimports"
        fi
    else
        print_warning "goimports not available, installing..."
        go install golang.org/x/tools/cmd/goimports@latest
        goimports -w .
    fi
}

# Lint code
lint_code() {
    print_header "Linting Code"
    
    # Run go vet
    if go vet ./...; then
        print_success "go vet passed"
    else
        print_error "go vet failed"
        return 1
    fi
    
    # Run golangci-lint
    if golangci-lint run --timeout=5m; then
        print_success "golangci-lint passed"
    else
        print_error "golangci-lint failed"
        return 1
    fi
}

# Security scan
security_scan() {
    print_header "Security Scanning"
    
    # Run gosec
    if gosec -fmt json -out "${TEST_RESULTS_DIR}/gosec-report.json" ./...; then
        print_success "gosec security scan passed"
    else
        print_warning "gosec found security issues (check ${TEST_RESULTS_DIR}/gosec-report.json)"
    fi
    
    # Run govulncheck
    if govulncheck ./...; then
        print_success "govulncheck passed"
    else
        print_error "govulncheck found vulnerabilities"
        return 1
    fi
}

# Run unit tests
run_unit_tests() {
    print_header "Running Unit Tests"
    
    local test_flags="-v -timeout ${TEST_TIMEOUT}"
    
    if [[ "$VERBOSE" == "true" ]]; then
        test_flags="$test_flags -v"
    fi
    
    # Run unit tests with coverage (from parent directory)
    if (cd .. && go test $test_flags -coverprofile="${COVERAGE_DIR}/unit-coverage.out" -covermode=atomic ./internal/... ./pkg/...); then
        print_success "Unit tests passed"
        
        # Generate coverage report
        go tool cover -func="${COVERAGE_DIR}/unit-coverage.out" > "${COVERAGE_DIR}/unit-coverage.txt"
        go tool cover -html="${COVERAGE_DIR}/unit-coverage.out" -o "${COVERAGE_DIR}/unit-coverage.html"
        
        # Check coverage threshold
        local coverage=$(go tool cover -func="${COVERAGE_DIR}/unit-coverage.out" | grep total | awk '{print $3}' | sed 's/%//')
        print_info "Unit test coverage: ${coverage}%"
        
        if (( $(echo "$coverage >= $COVERAGE_THRESHOLD" | bc -l) )); then
            print_success "Coverage threshold met (${coverage}% >= ${COVERAGE_THRESHOLD}%)"
        else
            print_warning "Coverage below threshold (${coverage}% < ${COVERAGE_THRESHOLD}%)"
        fi
    else
        print_error "Unit tests failed"
        return 1
    fi
}

# Run integration tests
run_integration_tests() {
    print_header "Running Integration Tests"
    
    local test_flags="-v -timeout ${TEST_TIMEOUT}"
    
    if [[ "$VERBOSE" == "true" ]]; then
        test_flags="$test_flags -v"
    fi
    
    if (cd .. && go test $test_flags -coverprofile="${COVERAGE_DIR}/integration-coverage.out" -covermode=atomic -run "TestIntegration" ./tests/); then
        print_success "Integration tests passed"
        
        # Generate coverage report
        go tool cover -func="${COVERAGE_DIR}/integration-coverage.out" > "${COVERAGE_DIR}/integration-coverage.txt"
        go tool cover -html="${COVERAGE_DIR}/integration-coverage.out" -o "${COVERAGE_DIR}/integration-coverage.html"
    else
        print_error "Integration tests failed"
        return 1
    fi
}

# Run performance tests
run_performance_tests() {
    print_header "Running Performance Tests"
    
    # Run benchmarks
    if (cd .. && go test -bench=. -benchmem -timeout ${TEST_TIMEOUT} -run=^$$ ./tests/) > "${TEST_RESULTS_DIR}/benchmark-results.txt"; then
        print_success "Performance benchmarks completed"
        print_info "Results saved to ${TEST_RESULTS_DIR}/benchmark-results.txt"
    else
        print_error "Performance benchmarks failed"
        return 1
    fi
    
    # Run CPU profiling
    if (cd .. && go test -bench=. -cpuprofile="${TEST_RESULTS_DIR}/cpu.prof" -timeout ${TEST_TIMEOUT} -run=^$$ ./tests/); then
        print_success "CPU profiling completed"
        print_info "CPU profile saved to ${TEST_RESULTS_DIR}/cpu.prof"
    else
        print_warning "CPU profiling failed"
    fi
    
    # Run memory profiling
    if (cd .. && go test -bench=. -memprofile="${TEST_RESULTS_DIR}/mem.prof" -timeout ${TEST_TIMEOUT} -run=^$$ ./tests/); then
        print_success "Memory profiling completed"
        print_info "Memory profile saved to ${TEST_RESULTS_DIR}/mem.prof"
    else
        print_warning "Memory profiling failed"
    fi
}

# Run memory leak tests
run_memory_tests() {
    print_header "Running Memory Leak Tests"
    
    local test_flags="-v -timeout ${TEST_TIMEOUT}"
    
    if (cd .. && go test $test_flags -run "TestMemoryLeak" ./tests/); then
        print_success "Memory leak tests passed"
    else
        print_error "Memory leak tests failed"
        return 1
    fi
}

# Run race condition tests
run_race_tests() {
    print_header "Running Race Condition Tests"
    
    local test_flags="-race -v -timeout ${TEST_TIMEOUT}"
    
    if (cd .. && go test $test_flags ./tests/); then
        print_success "Race condition tests passed"
    else
        print_error "Race condition tests failed"
        return 1
    fi
}

# Generate combined coverage report
generate_coverage_report() {
    print_header "Generating Combined Coverage Report"
    
    # Combine coverage files if they exist
    local coverage_files=()
    [[ -f "${COVERAGE_DIR}/unit-coverage.out" ]] && coverage_files+=("${COVERAGE_DIR}/unit-coverage.out")
    [[ -f "${COVERAGE_DIR}/integration-coverage.out" ]] && coverage_files+=("${COVERAGE_DIR}/integration-coverage.out")
    
    if [[ ${#coverage_files[@]} -gt 0 ]]; then
        # Create combined coverage file
        echo "mode: atomic" > "${COVERAGE_DIR}/combined-coverage.out"
        
        for file in "${coverage_files[@]}"; do
            tail -n +2 "$file" >> "${COVERAGE_DIR}/combined-coverage.out"
        done
        
        # Generate reports
        go tool cover -func="${COVERAGE_DIR}/combined-coverage.out" > "${COVERAGE_DIR}/combined-coverage.txt"
        go tool cover -html="${COVERAGE_DIR}/combined-coverage.out" -o "${COVERAGE_DIR}/combined-coverage.html"
        
        local total_coverage=$(go tool cover -func="${COVERAGE_DIR}/combined-coverage.out" | grep total | awk '{print $3}' | sed 's/%//')
        print_success "Combined coverage report generated"
        print_info "Total coverage: ${total_coverage}%"
        print_info "HTML report: ${COVERAGE_DIR}/combined-coverage.html"
    else
        print_warning "No coverage files found to combine"
    fi
}

# Generate test summary
generate_summary() {
    print_header "Test Summary"
    
    local summary_file="${TEST_RESULTS_DIR}/test-summary.txt"
    
    {
        echo "Test Summary - $(date)"
        echo "=================================="
        echo ""
        echo "Project: var-sync"
        echo "Go Version: $(go version)"
        echo "Test Timeout: ${TEST_TIMEOUT}"
        echo "Coverage Threshold: ${COVERAGE_THRESHOLD}%"
        echo ""
        
        if [[ -f "${COVERAGE_DIR}/combined-coverage.txt" ]]; then
            echo "Coverage Results:"
            cat "${COVERAGE_DIR}/combined-coverage.txt"
            echo ""
        fi
        
        if [[ -f "${TEST_RESULTS_DIR}/benchmark-results.txt" ]]; then
            echo "Benchmark Results:"
            head -20 "${TEST_RESULTS_DIR}/benchmark-results.txt"
            echo ""
        fi
        
        echo "Generated files:"
        find "${TEST_RESULTS_DIR}" "${COVERAGE_DIR}" -type f -name "*.out" -o -name "*.html" -o -name "*.txt" -o -name "*.json" -o -name "*.prof" | sort
        
    } > "$summary_file"
    
    print_success "Test summary generated: $summary_file"
    
    # Display summary
    if [[ "$VERBOSE" == "true" ]]; then
        cat "$summary_file"
    fi
}

# Clean up
cleanup() {
    print_header "Cleanup"
    
    # Remove temporary files
    find . -name "*.test" -delete
    
    print_success "Cleanup completed"
}

# Main execution
main() {
    local start_time=$(date +%s)
    
    print_header "var-sync Test Runner"
    print_info "Starting comprehensive test suite..."
    
    # Parse command line arguments
    local run_all=true
    local run_unit=false
    local run_integration=false
    local run_performance=false
    local run_memory=false
    local run_race=false
    local run_lint=false
    local run_security=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --unit)
                run_all=false
                run_unit=true
                shift
                ;;
            --integration)
                run_all=false
                run_integration=true
                shift
                ;;
            --performance)
                run_all=false
                run_performance=true
                shift
                ;;
            --memory)
                run_all=false
                run_memory=true
                shift
                ;;
            --race)
                run_all=false
                run_race=true
                shift
                ;;
            --lint)
                run_all=false
                run_lint=true
                shift
                ;;
            --security)
                run_all=false
                run_security=true
                shift
                ;;
            --verbose)
                VERBOSE=true
                shift
                ;;
            --coverage-threshold)
                COVERAGE_THRESHOLD="$2"
                shift 2
                ;;
            --timeout)
                TEST_TIMEOUT="$2"
                shift 2
                ;;
            --help)
                echo "Usage: $0 [options]"
                echo "Options:"
                echo "  --unit              Run only unit tests"
                echo "  --integration       Run only integration tests"
                echo "  --performance       Run only performance tests"
                echo "  --memory           Run only memory leak tests"
                echo "  --race             Run only race condition tests"
                echo "  --lint             Run only linting"
                echo "  --security         Run only security scans"
                echo "  --verbose          Enable verbose output"
                echo "  --coverage-threshold N  Set coverage threshold (default: 80)"
                echo "  --timeout DURATION Set test timeout (default: 10m)"
                echo "  --help             Show this help message"
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    # Change to project directory
    cd "$PROJECT_ROOT"
    
    # Always check dependencies and install deps
    check_dependencies
    install_deps
    
    local failed_tests=()
    
    # Run selected tests
    if [[ "$run_all" == "true" || "$run_lint" == "true" ]]; then
        format_code || failed_tests+=("format")
        lint_code || failed_tests+=("lint")
    fi
    
    if [[ "$run_all" == "true" || "$run_security" == "true" ]]; then
        security_scan || failed_tests+=("security")
    fi
    
    if [[ "$run_all" == "true" || "$run_unit" == "true" ]]; then
        run_unit_tests || failed_tests+=("unit")
    fi
    
    if [[ "$run_all" == "true" || "$run_integration" == "true" ]]; then
        run_integration_tests || failed_tests+=("integration")
    fi
    
    if [[ "$run_all" == "true" || "$run_race" == "true" ]]; then
        run_race_tests || failed_tests+=("race")
    fi
    
    if [[ "$run_all" == "true" || "$run_performance" == "true" ]]; then
        run_performance_tests || failed_tests+=("performance")
    fi
    
    if [[ "$run_all" == "true" || "$run_memory" == "true" ]]; then
        run_memory_tests || failed_tests+=("memory")
    fi
    
    # Generate reports
    generate_coverage_report
    generate_summary
    
    # Cleanup
    cleanup
    
    # Final results
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    print_header "Final Results"
    
    if [[ ${#failed_tests[@]} -eq 0 ]]; then
        print_success "All tests passed! 🎉"
        print_info "Total time: ${duration}s"
        exit 0
    else
        print_error "Some tests failed:"
        for test in "${failed_tests[@]}"; do
            print_error "  - $test"
        done
        print_info "Total time: ${duration}s"
        exit 1
    fi
}

# Run main function
main "$@"