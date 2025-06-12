# var-sync Makefile for test automation and CI/CD

.PHONY: all build test test-unit test-integration test-performance test-memory test-race-conditions test-coverage clean install deps lint fmt vet security help

# Build variables
BINARY_NAME=var-sync
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# Test variables
TEST_TIMEOUT=10m
COVERAGE_OUT=coverage.out
COVERAGE_HTML=coverage.html

# Default target
all: deps lint vet test build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) .

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Run all tests
test: test-unit test-integration

# Run unit tests only
test-unit:
	@echo "Running unit tests..."
	go test -v -timeout $(TEST_TIMEOUT) ./internal/... ./pkg/...

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	go test -v -timeout $(TEST_TIMEOUT) -run "TestIntegration" ./tests/

# Run performance tests (benchmarks)
test-performance:
	@echo "Running performance tests..."
	go test -v -timeout $(TEST_TIMEOUT) -bench=. -benchmem -run=^$$ ./tests/
	@echo "Running performance tests with CPU profiling..."
	go test -timeout $(TEST_TIMEOUT) -bench=. -benchmem -cpuprofile=cpu.prof -run=^$$ ./tests/
	@echo "Running performance tests with memory profiling..."
	go test -timeout $(TEST_TIMEOUT) -bench=. -benchmem -memprofile=mem.prof -run=^$$ ./tests/

# Run memory leak tests
test-memory:
	@echo "Running memory leak tests..."
	go test -v -timeout $(TEST_TIMEOUT) -run "TestMemoryLeak" ./tests/

# Run race condition tests
test-race-conditions:
	@echo "Running race condition tests..."
	go test -v -timeout $(TEST_TIMEOUT) -run "TestRace|TestRealWorld|TestSafe" ./tests/

# Run performance stress tests
test-stress:
	@echo "Running stress tests..."
	go test -v -timeout 30m -run "TestPerformance" ./tests/

# Generate test coverage report
test-coverage:
	@echo "Generating test coverage report..."
	go test -coverprofile=$(COVERAGE_OUT) -covermode=atomic ./internal/... ./pkg/... ./tests/
	go tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_HTML)
	go tool cover -func=$(COVERAGE_OUT)
	@echo "Coverage report saved to $(COVERAGE_HTML)"

# Run tests with race detection
test-race:
	@echo "Running tests with race detection..."
	go test -race -timeout $(TEST_TIMEOUT) ./internal/... ./pkg/... ./tests/

# Run all tests in short mode (for CI)
test-short:
	@echo "Running tests in short mode..."
	go test -short -timeout 5m ./internal/... ./pkg/... ./tests/

# Run tests with verbose output and coverage
test-verbose:
	@echo "Running verbose tests with coverage..."
	go test -v -coverprofile=$(COVERAGE_OUT) -covermode=atomic ./internal/... ./pkg/... ./tests/
	go tool cover -func=$(COVERAGE_OUT)

# Lint code
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "Running goimports..."
	@which goimports > /dev/null || (echo "Installing goimports..." && go install golang.org/x/tools/cmd/goimports@latest)
	goimports -w .

# Vet code
vet:
	@echo "Running go vet..."
	go vet ./...

# Security audit
security:
	@echo "Running security audit..."
	@which gosec > /dev/null || (echo "Installing gosec..." && go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest)
	gosec ./...
	@echo "Checking for known vulnerabilities..."
	@which govulncheck > /dev/null || (echo "Installing govulncheck..." && go install golang.org/x/vuln/cmd/govulncheck@latest)
	govulncheck ./...

# Profile CPU performance
profile-cpu:
	@echo "Running CPU profiling..."
	go test -timeout $(TEST_TIMEOUT) -bench=. -cpuprofile=cpu.prof -run=^$$ ./tests/
	@echo "View profile with: go tool pprof cpu.prof"

# Profile memory performance
profile-memory:
	@echo "Running memory profiling..."
	go test -timeout $(TEST_TIMEOUT) -bench=. -memprofile=mem.prof -run=^$$ ./tests/
	@echo "View profile with: go tool pprof mem.prof"

# Install the application
install: build
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) .

# Clean build artifacts and test files
clean:
	@echo "Cleaning up..."
	rm -f $(BINARY_NAME)
	rm -f $(COVERAGE_OUT) $(COVERAGE_HTML)
	rm -f cpu.prof mem.prof
	rm -f *.test
	go clean ./...

# Development setup
dev-setup: deps
	@echo "Setting up development environment..."
	@which pre-commit > /dev/null || (echo "Installing pre-commit..." && pip install pre-commit)
	pre-commit install
	@echo "Development environment ready!"

# CI pipeline (used by continuous integration)
ci: deps lint vet security test-race test-coverage
	@echo "CI pipeline completed successfully!"

# Quick check (for pre-commit hooks)
quick-check: fmt vet test-short

# Generate documentation
docs:
	@echo "Generating documentation..."
	@which godoc > /dev/null || (echo "Installing godoc..." && go install golang.org/x/tools/cmd/godoc@latest)
	@echo "Documentation server available at: http://localhost:6060/pkg/var-sync/"
	@echo "Run 'godoc -http=:6060' to start documentation server"

# Release build (optimized)
release:
	@echo "Building release version..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -a -installsuffix cgo -o $(BINARY_NAME)-linux-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -a -installsuffix cgo -o $(BINARY_NAME)-darwin-amd64 .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -a -installsuffix cgo -o $(BINARY_NAME)-windows-amd64.exe .
	@echo "Release binaries created!"

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t var-sync:latest .
	docker build -t var-sync:$(VERSION) .

# Docker test
docker-test:
	@echo "Running tests in Docker..."
	docker run --rm -v $(PWD):/app -w /app golang:1.22 make test

# Check test dependencies
check-deps:
	@echo "Checking test dependencies..."
	@go list -m all | grep -E "(testify|mock|gomega|ginkgo)" || echo "No test frameworks found"
	@echo "Go version: $(shell go version)"
	@echo "Available test tools:"
	@which golangci-lint && echo "  ✓ golangci-lint" || echo "  ✗ golangci-lint (run 'make lint' to install)"
	@which gosec && echo "  ✓ gosec" || echo "  ✗ gosec (run 'make security' to install)"
	@which govulncheck && echo "  ✓ govulncheck" || echo "  ✗ govulncheck (run 'make security' to install)"

# Display help
help:
	@echo "Available targets:"
	@echo "  build           - Build the application"
	@echo "  test            - Run all tests (unit + integration)"
	@echo "  test-unit       - Run unit tests only"
	@echo "  test-integration- Run integration tests only"
	@echo "  test-performance- Run performance benchmarks"
	@echo "  test-memory     - Run memory leak tests"
	@echo "  test-race-conditions - Run race condition tests"
	@echo "  test-coverage   - Generate test coverage report"
	@echo "  test-race       - Run tests with race detection"
	@echo "  test-short      - Run tests in short mode"
	@echo "  test-stress     - Run stress tests"
	@echo "  lint            - Run linter"
	@echo "  fmt             - Format code"
	@echo "  vet             - Run go vet"
	@echo "  security        - Run security audit"
	@echo "  clean           - Clean build artifacts"
	@echo "  install         - Install the application"
	@echo "  deps            - Install dependencies"
	@echo "  ci              - Run CI pipeline"
	@echo "  dev-setup       - Set up development environment"
	@echo "  release         - Build release binaries"
	@echo "  docker-build    - Build Docker image"
	@echo "  docker-test     - Run tests in Docker"
	@echo "  profile-cpu     - Run CPU profiling"
	@echo "  profile-memory  - Run memory profiling"
	@echo "  docs            - Generate documentation"
	@echo "  check-deps      - Check test dependencies"
	@echo "  help            - Show this help message"