# Build stage
FROM --platform=$BUILDPLATFORM golang:1.21-alpine AS builder

# Install ca-certificates and air for hot-reload
RUN apk add --no-cache ca-certificates git

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies (cached layer)
RUN go mod download && go mod verify

# Install air for hot-reload
RUN go install github.com/cosmtrek/air@latest

# Copy source code
COPY . .

# Build the binary with optimizations for target platform
ARG TARGETOS TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -a \
    -installsuffix cgo \
    -ldflags="-w -s -extldflags '-static'" \
    -tags netgo \
    -o server \
    ./cmd/server

# Strip the binary for additional size reduction
RUN strip server

# Development stage with air
FROM golang:1.21-alpine AS dev

# Install ca-certificates and air
RUN apk add --no-cache ca-certificates git

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Install air for hot-reload
RUN go install github.com/cosmtrek/air@latest

# Copy source code
COPY . .

# Expose port
EXPOSE 8080

# Use air for hot-reload
CMD ["air", "-c", ".air.toml"]

# Runtime stage
FROM scratch

# Import CA certificates from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /app/server /server

# Expose port
EXPOSE 8080

# Health check using the application itself (no external tools needed)
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/server", "--health-check"] || exit 1

# Set the binary as entrypoint
ENTRYPOINT ["/server"]

# Default command (can be overridden)
CMD []
