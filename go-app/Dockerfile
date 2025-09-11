# Build stage
FROM golang:1.21-alpine AS builder

# Install ca-certificates (needed for HTTPS)
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies (cached layer)
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -a \
    -installsuffix cgo \
    -ldflags="-w -s -extldflags '-static'" \
    -tags netgo \
    -o server \
    ./cmd/server

# Strip the binary for additional size reduction
RUN strip server

# Runtime stage
FROM scratch

# Import CA certificates from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /app/server /server

# Expose port
EXPOSE 8080

# Health check using the application itself
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/server", "--version"] || exit 1

# Set the binary as entrypoint
ENTRYPOINT ["/server"]

# Default command (can be overridden)
CMD []
