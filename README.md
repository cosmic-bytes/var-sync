# var-sync

A lightweight, cross-format file synchronization tool that watches configuration files (YAML, TOML, JSON) and syncs specific key-value pairs to other files when changes occur.

## Features

- **Cross-format support**: Sync between YAML, TOML, and JSON files
- **Real-time watching**: Automatically detects file changes and syncs values
- **Interactive TUI**: User-friendly terminal interface for configuration
- **Nested key paths**: Support for deep object traversal (e.g., `database.connection.host`)
- **Key selection**: Interactive autocomplete for selecting keys from existing files
- **Persistent configuration**: Rules are saved and restored between sessions
- **Structured logging**: Comprehensive logging with configurable levels
- **Memory efficient**: Lightweight design with minimal resource usage

## Installation

```bash
go build -o var-sync
```

## Usage

### Interactive TUI Mode

Start the interactive terminal interface to configure sync rules:

```bash
./var-sync -tui
```

**TUI Controls:**
- `a`: Add new sync rule
- `Enter`: Edit selected rule
- `d`: Delete selected rule
- `q`: Quit
- `Tab`: Navigate form fields
- `Ctrl+K`: Interactive key selection from file
- `Ctrl+S`: Save rule
- `Esc`: Cancel/Back

### Watch Mode

Start watching configured files for changes:

```bash
./var-sync -watch
```

### Command Line Options

```bash
./var-sync [OPTIONS]

Options:
  -config string     Configuration file path (default "var-sync.json")
  -tui              Start interactive TUI mode
  -watch            Start file watching mode
  -version          Show version
```

## Configuration

The tool uses a JSON configuration file to store sync rules. By default, it looks for `var-sync.json` in the current directory.

### Sample Configuration

```json
{
  "rules": [
    {
      "id": "db-host-sync",
      "name": "Database Host Sync",
      "description": "Sync database host from config to app settings",
      "source_file": "sample-config.yaml",
      "source_key": "database.host",
      "target_file": "sample-target.json",
      "target_key": "config.database.host",
      "enabled": true,
      "created": "2024-01-01T00:00:00Z"
    }
  ],
  "log_file": "var-sync.log",
  "debug": false
}
```

## Example Workflow

1. **Configure sync rules**:
   ```bash
   ./var-sync -tui
   ```
   - Add a rule to sync `database.host` from `config.yaml` to `app.json`
   - Use `Ctrl+K` to interactively select keys from existing files

2. **Start watching**:
   ```bash
   ./var-sync -watch
   ```

3. **Make changes**:
   - Edit `config.yaml` and change the `database.host` value
   - The tool automatically detects the change and updates `app.json`

## Supported File Formats

### YAML (.yaml, .yml)
```yaml
database:
  host: localhost
  port: 5432
```

### TOML (.toml)
```toml
[database]
host = "localhost"
port = 5432
```

### JSON (.json)
```json
{
  "database": {
    "host": "localhost",
    "port": 5432
  }
}
```

## Key Path Syntax

Use dot notation to specify nested keys:
- `database.host` → accesses `database.host` in the file
- `config.db.connection.host` → accesses deeply nested values
- `api.endpoints.users` → accesses array/object values

## Logging

Logs are written to the specified log file (default: `var-sync.log`) and include:
- Sync operations (success/failure)
- File watching events
- Configuration changes
- Error messages

Log levels: DEBUG, INFO, WARN, ERROR

## Testing

var-sync includes a comprehensive test suite with unit tests, integration tests, performance benchmarks, and memory leak detection.

### Quick Test Commands

```bash
# Run all unit tests
make test-unit

# Run integration tests
make test-integration

# Run performance benchmarks
make test-performance

# Run memory leak tests
make test-memory

# Run all tests with coverage
make test-coverage

# Run comprehensive test suite
./test_runner.sh --verbose
```

### Test Categories

#### Unit Tests
- **Config Management**: Configuration loading, saving, and validation
- **File Parser**: JSON/YAML/TOML parsing and manipulation
- **Logger**: Logging functionality and file operations
- **Data Models**: Data structure validation and serialization

#### Integration Tests
- **Full Sync Flow**: End-to-end synchronization testing
- **Multi-format Support**: Cross-format file synchronization
- **Error Handling**: Robust error scenario validation
- **Real-world Scenarios**: Microservice configuration patterns

#### Performance Tests
- **File Operations**: Benchmark parsing performance across formats
- **Concurrent Access**: Multi-threaded operation testing
- **Memory Usage**: Resource consumption monitoring
- **Scalability**: Large dataset handling

#### Memory Leak Tests
- **Parser Operations**: Large-scale file processing
- **Config Management**: Configuration persistence
- **Logger Operations**: Logging system validation
- **Long-running Scenarios**: Production simulation

### Test Automation

The project includes automated testing via:

- **GitHub Actions CI**: Multi-platform testing (Linux, macOS, Windows)
- **Pre-commit Hooks**: Code quality enforcement
- **Makefile**: Local development automation
- **Test Runner Script**: Comprehensive test execution with reporting

### Coverage Reports

Generate detailed coverage reports:

```bash
# Generate HTML coverage report
make test-coverage
open coverage.html

# View coverage summary
go tool cover -func=coverage.out
```

### Performance Profiling

Run performance analysis:

```bash
# CPU profiling
make profile-cpu
go tool pprof cpu.prof

# Memory profiling
make profile-memory
go tool pprof mem.prof
```

### Test Configuration

Customize test behavior with environment variables:

```bash
# Set test timeout
export TEST_TIMEOUT=15m

# Set coverage threshold
export COVERAGE_THRESHOLD=85

# Enable verbose output
export VERBOSE=true

# Run tests
./test_runner.sh
```

## Building from Source

```bash
go mod tidy
go build -o var-sync
```

## Development

### Prerequisites

- Go 1.21 or later
- Make (for build automation)
- Git (for version control)

### Development Setup

```bash
# Clone the repository
git clone <repository-url>
cd var-sync-go

# Install dependencies
make deps

# Set up development environment (includes pre-commit hooks)
make dev-setup

# Run quick checks
make quick-check
```

### Code Quality

The project maintains high code quality through:

- **Linting**: golangci-lint with comprehensive rules
- **Formatting**: Automated code formatting with gofmt and goimports
- **Security**: Security scanning with gosec and govulncheck
- **Testing**: Minimum 80% test coverage requirement
- **Pre-commit Hooks**: Automated quality checks before commits

### Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Run the test suite: `./test_runner.sh`
5. Ensure all checks pass: `make ci`
6. Submit a pull request

## CI/CD

The project uses GitHub Actions for continuous integration:

- **Multi-platform Testing**: Linux, macOS, Windows
- **Multiple Go Versions**: 1.21, 1.22
- **Comprehensive Testing**: Unit, integration, performance, and memory tests
- **Security Scanning**: Dependency and code vulnerability checks
- **Code Coverage**: Automated coverage reporting
- **Build Verification**: Cross-platform build testing

## Dependencies

### Runtime Dependencies
- `github.com/fsnotify/fsnotify` - File system notifications
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/bubbles` - TUI components
- `gopkg.in/yaml.v3` - YAML parsing
- `github.com/BurntSushi/toml` - TOML parsing
- `github.com/google/uuid` - UUID generation

### Development Dependencies
- `github.com/golangci/golangci-lint` - Comprehensive linting
- `github.com/securecodewarrior/gosec` - Security analysis
- `golang.org/x/vuln/cmd/govulncheck` - Vulnerability scanning
- `golang.org/x/tools/cmd/goimports` - Import formatting

## Performance

var-sync is designed for high performance and low resource usage:

- **Lightweight**: Minimal memory footprint
- **Fast Parsing**: Optimized file format parsers
- **Concurrent Safe**: Thread-safe operations
- **Efficient Watching**: Minimal CPU usage during file monitoring
- **Memory Leak Free**: Comprehensive memory leak testing

Benchmark results on typical hardware (Apple M1):
- JSON parsing: ~18μs per operation, 4.5KB, 76 allocs
- YAML parsing: ~38μs per operation, 19.8KB, 287 allocs  
- TOML parsing: ~34μs per operation, 15KB, 256 allocs
- Value retrieval: 37-129ns depending on key depth
- Config operations: 245-379ns per operation
- Concurrent file reads: ~79μs per operation

## License

MIT License