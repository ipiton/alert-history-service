# Alert History Service - Go Version

## Project Structure

This Go application follows standard Go project layout and hexagonal architecture principles.

### Directory Structure

```
go-app/
├── cmd/
│   └── server/
│       └── main.go         # Main entry point
├── internal/
│   ├── api/                # HTTP handlers and routes
│   ├── core/               # Business logic and domain models
│   ├── infrastructure/     # External adapters (DB, Redis, HTTP clients)
│   └── config/             # Configuration management
├── pkg/
│   ├── logger/             # Shared logging utilities
│   ├── metrics/            # Shared metrics utilities
│   └── utils/              # Common utilities
├── go.mod                  # Go module definition
├── go.sum                  # Dependency checksums
└── README.md              # This file
```

### Package Descriptions

#### `cmd/`
Contains main applications for this project. Each subdirectory represents a separate application:
- `cmd/server/` - Main HTTP server application

#### `internal/`
Private application and library code. This is the code you don't want others importing in their applications or libraries.

- `internal/api/` - HTTP handlers, routes, middleware
- `internal/core/` - Domain models, business logic, services
- `internal/infrastructure/` - Adapters for external systems (PostgreSQL, Redis, HTTP clients)
- `internal/config/` - Configuration loading and validation

#### `pkg/`
Library code that's OK to use by external applications. Other projects will import these packages expecting them to work, so think twice before putting something here.

- `pkg/logger/` - Structured logging utilities
- `pkg/metrics/` - Prometheus metrics utilities
- `pkg/utils/` - Common utility functions

## Development

### Prerequisites
- Go 1.21+ (currently using 1.24.6)
- Git

### Building
```bash
# Using Go directly
go build ./cmd/server

# Using Makefile (recommended)
make build
```

### Running
```bash
# Using Go directly
go run ./cmd/server

# Using Makefile
make run

# Development mode with hot reload (requires air)
make dev
```

### Testing
```bash
# Basic tests
go test ./...
make test

# With coverage report
make test-coverage
```

### Development Commands
```bash
# Format code
make fmt

# Run linter (requires golangci-lint)
make lint

# Run static analysis
make vet

# Clean build artifacts
make clean

# Install development tools
make install-tools

# Show all available commands
make help
```

## Architecture Principles

This project follows:
- **Hexagonal Architecture** (Ports & Adapters)
- **Dependency Injection** for testability
- **Structured Logging** with slog
- **12-Factor App** compliance
- **Standard Go Project Layout**

## Migration from Python

This Go version is being developed alongside the existing Python version to ensure:
- 100% API compatibility
- Feature parity
- Performance improvements
- Better scalability

See `../tasks/go-migration-analysis/` for detailed migration plan.
