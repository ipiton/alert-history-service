# Makefile for Alert History Service (Go version)
.PHONY: build test lint run clean help deps fmt vet mod-tidy

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Main package path
MAIN_PATH=./cmd/server
BINARY_NAME=server
BINARY_UNIX=$(BINARY_NAME)_unix

# Build the project
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) -o $(BINARY_NAME) -v $(MAIN_PATH)
	@echo "Build complete: $(BINARY_NAME)"

# Build for Linux (for Docker)
build-linux:
	@echo "Building $(BINARY_NAME) for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v $(MAIN_PATH)
	@echo "Linux build complete: $(BINARY_UNIX)"

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Run linter (requires golangci-lint to be installed)
lint:
	@echo "Running linter..."
	golangci-lint run

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Run vet (Go's static analyzer)
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

# Tidy dependencies
mod-tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download

# Run the application
run:
	@echo "Running application..."
	$(GOCMD) run $(MAIN_PATH)

# Run with hot reload (requires air to be installed)
dev:
	@echo "Running in development mode with hot reload..."
	air

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f coverage.out coverage.html
	@echo "Clean complete"

# Install development tools
install-tools:
	@echo "Installing development tools..."
	$(GOGET) -u github.com/cosmtrek/air
	$(GOGET) -u github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Create new migration (requires goose to be installed)
migrate-create:
	@echo "Creating new migration..."
	@if [ -z "$(name)" ]; then echo "Usage: make migrate-create name=migration_name"; exit 1; fi
	goose -dir migrations create $(name) sql

# Run migrations up (requires goose to be installed)
migrate-up:
	@echo "Running migrations up..."
	goose -dir migrations postgres "$(DATABASE_URL)" up

# Run migrations down (requires goose to be installed)
migrate-down:
	@echo "Running migrations down..."
	goose -dir migrations postgres "$(DATABASE_URL)" down

# Show available commands
help:
	@echo "Available commands:"
	@echo "  build         - Build the application"
	@echo "  build-linux   - Build for Linux (Docker)"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  lint          - Run linter (requires golangci-lint)"
	@echo "  fmt           - Format code"
	@echo "  vet           - Run go vet"
	@echo "  mod-tidy      - Tidy dependencies"
	@echo "  deps          - Download dependencies"
	@echo "  run           - Run the application"
	@echo "  dev           - Run with hot reload (requires air)"
	@echo "  clean         - Clean build artifacts"
	@echo "  install-tools - Install development tools"
	@echo "  migrate-create - Create new migration (requires goose)"
	@echo "  migrate-up    - Run migrations up"
	@echo "  migrate-down  - Run migrations down"
