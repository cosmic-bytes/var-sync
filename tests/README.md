# Test Suites

This directory contains different categories of tests for the var-sync project.

## Regular Tests

Run the standard test suite (unit tests, integration tests):

```bash
go test ./...
```

This runs all the core functionality tests and basic integration tests.

## Extended Test Suites

For more comprehensive testing, you can run specialized test suites:

### Race Condition Tests
Tests concurrent access scenarios and race condition handling:
```bash
go test -tags race ./tests/
```

### Performance Tests  
Tests performance under heavy load and concurrent access:
```bash
go test -tags performance ./tests/
```

### Memory Leak Tests
Tests for memory leaks during long-running operations:
```bash
go test -tags memory ./tests/
```

### All Extended Tests
Run all extended test suites:
```bash
go test -tags "race performance memory" ./tests/
```

## Test Categories

- **Unit Tests**: Test individual components (parser, config, logger, etc.)
- **Integration Tests**: Test component interaction and file operations
- **Race Condition Tests**: Test concurrent access and corruption detection
- **Performance Tests**: Test performance under load
- **Memory Tests**: Test memory usage and leak detection

## Notes

- Race condition and performance tests may intentionally cause file corruption to test error handling
- Extended test suites are designed for CI/CD environments and comprehensive testing
- Regular `go test ./...` runs the essential tests needed for development