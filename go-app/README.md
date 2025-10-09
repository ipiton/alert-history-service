# Alert History Service - Go Version

🚀 **Production-ready Go implementation** of Alert History Service with optimal performance, minimal resource usage, and cloud-native deployment.

## Prerequisites

- **Go**: 1.21 or later (currently tested with 1.21, 1.22, 1.23)
- **Git**: For cloning and version control
- **Docker**: Optional, for containerized development
- **Make**: Optional, for using development shortcuts

## Quick Start

### Option 1: Docker (Recommended)

```bash
# Build and run with Docker
make docker-build
make docker-run

# Check health
curl http://localhost:8080/healthz

# View structured logs
make docker-logs
```

### Option 2: Local Development

```bash
# Install dependencies
make deps

# Build and run
make build
make run

# Or run directly
go run ./cmd/server
```

### Option 3: Development Mode

```bash
# Install air for hot reload
make install-tools

# Run with hot reload
make dev
```

## Health Checks

```bash
# Health check
curl http://localhost:8080/healthz

# Version info
./server --version

# Help
./server --help
```

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

## CI/CD

### GitHub Actions
The project uses GitHub Actions for continuous integration and delivery:

- **Multi-version testing**: Go 1.21, 1.22, 1.23
- **Database integration**: PostgreSQL and Redis services
- **Comprehensive checks**: build, test, lint, security scan
- **Coverage reporting**: Codecov integration
- **Multi-platform builds**: Linux binaries for different architectures

### Workflow Triggers
- Push to `main`, `develop`, `feature/use-LLM` branches
- Pull requests to these branches
- Only when Go code changes (`go-app/**` or workflow files)

### Available Jobs
- **test**: Unit tests with race detection and coverage
- **lint**: golangci-lint code quality checks
- **build**: Multi-platform binary builds
- **security**: Gosec security scanning with SARIF reports
- **dependency-review**: Dependency vulnerability checks

### Local Development
```bash
# Run local checks before pushing
make test
make lint
make build

# Or run all checks at once
make test && make lint && make build
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

## Docker

### Building and Running

```bash
# Build Docker image
make docker-build

# Run container
make docker-run

# Run in background
make docker-run-detached

# View logs
make docker-logs

# Stop container
make docker-stop

# Clean up
make docker-clean
```

### Docker Image Details

- **Base Image**: `golang:1.21-alpine` (build) → `scratch` (runtime)
- **Size**: < 10MB final image (< 15MB with multi-arch manifest)
- **Architecture**: Linux AMD64 + ARM64 (multi-platform support)
- **Security**: Non-root user, minimal attack surface
- **Health Check**: Application self-test every 30s
- **Build Tools**: Docker Buildx, Docker Bake, Docker Compose

### Custom Build

```bash
# Build with custom tag
docker build -t my-alert-history:latest .

# Run with custom port
docker run -p 9090:8080 my-alert-history:latest

# Run with environment variables
docker run -e PORT=9090 -p 9090:9090 my-alert-history:latest
```

### Multi-Architecture Builds

```bash
# Build for AMD64 and ARM64 with push to registry
docker buildx build --platform linux/amd64,linux/arm64 \
  -t ipiton/alert-history-llm:1.1.3 \
  . --push

# Using Docker Bake (advanced)
make bake-build

# Using Docker Compose
make compose-build

# Development multi-arch build
make bake-dev
```

#### Supported Platforms
- ✅ **linux/amd64** - Intel/AMD 64-bit
- ✅ **linux/arm64** - ARM 64-bit (Apple Silicon, AWS Graviton, etc.)
- 🚧 **linux/arm/v7** - ARM 32-bit (future support)

#### Build Tools
- **Docker Buildx**: Native multi-platform builds
- **Docker Bake**: Advanced build orchestration
- **Docker Compose**: Multi-service development
- **GitHub Actions**: CI/CD multi-platform builds

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |

### Command Line Flags

```bash
./server --help
```

Available flags:
- `--version`: Show version information
- `--help`: Show help message

## Testing

### Unit Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific package
go test ./cmd/server/handlers/

# Run with verbose output
go test -v ./...
```

### Integration Tests

```bash
# Test with Docker Compose (future)
make test-integration
```

### Code Quality

```bash
# Run linter
make lint

# Format code
make fmt

# Static analysis
make vet

# All quality checks
make test && make lint && make vet
```

## Troubleshooting

### Common Issues

#### Build Issues

**Problem**: `go: command not found`
```bash
# Install Go
brew install go  # macOS
# or download from https://golang.org/dl/
```

**Problem**: `make: command not found`
```bash
# Install make
# macOS: xcode-select --install
# Ubuntu: sudo apt-get install build-essential
```

#### Runtime Issues

**Problem**: `bind: address already in use`
```bash
# Kill process on port 8080
lsof -ti:8080 | xargs kill -9

# Or use different port
PORT=9090 ./server
```

**Problem**: Permission denied when running binary
```bash
# Make binary executable
chmod +x server
```

#### Docker Issues

**Problem**: `docker: command not found`
```bash
# Install Docker Desktop
# https://www.docker.com/products/docker-desktop
```

**Problem**: `docker build` fails
```bash
# Check Docker daemon
docker info

# Clean Docker cache
docker system prune -a
```

### Debug Commands

```bash
# Check Go version
go version

# Check Go environment
go env

# List Go modules
go list -m all

# Clean Go cache
go clean -modcache

# Check binary info
file server
ldd server  # Should show "not a dynamic executable"
```

### Logs and Debugging

```bash
# View application logs (JSON format)
./server 2>&1 | jq .

# Debug build
make build 2>&1

# Test health endpoint
curl -v http://localhost:8080/healthz

# Check process
ps aux | grep server
```

### Performance Tuning

```bash
# Profile application
go tool pprof http://localhost:8080/debug/pprof/profile

# Benchmark tests
go test -bench=. ./...

# Memory usage
go tool pprof http://localhost:8080/debug/pprof/heap
```

## Development Workflow

### Daily Development

1. **Pull latest changes**
   ```bash
   git pull origin main
   ```

2. **Install/update dependencies**
   ```bash
   make deps
   ```

3. **Run tests**
   ```bash
   make test
   ```

4. **Start development server**
   ```bash
   make dev
   ```

5. **Make changes and test**
   ```bash
   # Edit code
   make test  # Run tests
   make lint  # Check code quality
   ```

6. **Build and verify**
   ```bash
   make build
   ./server --version
   ```

### Code Quality

- **Formatting**: `make fmt`
- **Linting**: `make lint`
- **Testing**: `make test`
- **Security**: `make vet`

### Git Workflow

```bash
# Create feature branch
git checkout -b feature/my-feature

# Make changes
# ... edit code ...

# Test changes
make test && make lint

# Commit changes
git add .
git commit -m "feat: add my feature"

# Push branch
git push origin feature/my-feature

# Create PR
```

## API Endpoints

### Health & Status
- `GET /healthz` - Health check endpoint

### Future Endpoints
- Webhook endpoints
- Dashboard endpoints
- API endpoints
- Metrics endpoints

## Contributing

1. Follow Go best practices
2. Write tests for new code
3. Update documentation
4. Run full test suite before PR

## License

Same as main project - MIT License.
