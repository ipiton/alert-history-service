#!/bin/bash

# Performance Baseline Test Runner for Go Alert History Service
# This script runs load tests and collects performance profiles

set -euo pipefail

# Configuration
BASE_URL="${BASE_URL:-http://localhost:8080}"
GO_APP_DIR="${GO_APP_DIR:-../../../go-app}"
RESULTS_DIR="./results"
PROFILES_DIR="./profiles"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up..."
    if [[ -n "${GO_APP_PID:-}" ]]; then
        log_info "Stopping Go application (PID: $GO_APP_PID)"
        kill $GO_APP_PID 2>/dev/null || true
        wait $GO_APP_PID 2>/dev/null || true
    fi
}

# Set trap for cleanup
trap cleanup EXIT

# Check dependencies
check_dependencies() {
    log_info "Checking dependencies..."

    # Check if k6 is installed
    if ! command -v k6 &> /dev/null; then
        log_error "k6 is not installed. Please install k6: https://k6.io/docs/getting-started/installation/"
        exit 1
    fi

    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed. Please install Go: https://golang.org/doc/install"
        exit 1
    fi

    # Check if curl is available
    if ! command -v curl &> /dev/null; then
        log_error "curl is not installed. Please install curl."
        exit 1
    fi

    log_success "All dependencies are available"
}

# Build Go application
build_go_app() {
    log_info "Building Go application..."

    cd "$GO_APP_DIR"
    if ! go build -o bin/alert-history ./cmd/server; then
        log_error "Failed to build Go application"
        exit 1
    fi
    cd - > /dev/null

    log_success "Go application built successfully"
}

# Start Go application
start_go_app() {
    log_info "Starting Go application in MOCK MODE..."

    cd "$GO_APP_DIR"
    export MOCK_MODE=true
    export PERFORMANCE_TEST=true
    ./bin/alert-history &
    GO_APP_PID=$!
    cd - > /dev/null

    # Wait for application to start
    log_info "Waiting for application to start..."
    for i in {1..30}; do
        if curl -s "$BASE_URL/healthz" > /dev/null 2>&1; then
            log_success "Go application is running (PID: $GO_APP_PID)"
            return 0
        fi
        sleep 1
    done

    log_error "Failed to start Go application or health check failed"
    exit 1
}

# Create directories
create_directories() {
    log_info "Creating result directories..."
    mkdir -p "$RESULTS_DIR"
    mkdir -p "$PROFILES_DIR"
    log_success "Directories created"
}

# Run webhook load test
run_webhook_test() {
    log_info "Running webhook load test..."

    local output_file="$RESULTS_DIR/webhook_test_${TIMESTAMP}.json"

    if k6 run --out json="$output_file" \
        --env BASE_URL="$BASE_URL" \
        k6-webhook-test.js; then
        log_success "Webhook load test completed successfully"
        log_info "Results saved to: $output_file"
    else
        log_error "Webhook load test failed"
        return 1
    fi
}

# Run history API load test
run_history_test() {
    log_info "Running history API load test..."

    local output_file="$RESULTS_DIR/history_test_${TIMESTAMP}.json"

    if k6 run --out json="$output_file" \
        --env BASE_URL="$BASE_URL" \
        k6-history-test.js; then
        log_success "History API load test completed successfully"
        log_info "Results saved to: $output_file"
    else
        log_error "History API load test failed"
        return 1
    fi
}

# Collect CPU profile
collect_cpu_profile() {
    log_info "Collecting CPU profile..."

    local profile_file="$PROFILES_DIR/cpu_profile_${TIMESTAMP}.prof"

    # Start CPU profiling in background
    curl -s "$BASE_URL/debug/pprof/profile?seconds=30" > "$profile_file" &
    local profile_pid=$!

    # Run some load during profiling
    log_info "Running light load during CPU profiling..."
    k6 run --duration 30s --vus 10 \
        --env BASE_URL="$BASE_URL" \
        k6-webhook-test.js > /dev/null 2>&1 || true

    # Wait for profile to complete
    wait $profile_pid

    if [[ -f "$profile_file" && -s "$profile_file" ]]; then
        log_success "CPU profile collected: $profile_file"
    else
        log_error "Failed to collect CPU profile"
        return 1
    fi
}

# Collect memory profile
collect_memory_profile() {
    log_info "Collecting memory profile..."

    local profile_file="$PROFILES_DIR/memory_profile_${TIMESTAMP}.prof"

    if curl -s "$BASE_URL/debug/pprof/heap" > "$profile_file"; then
        if [[ -f "$profile_file" && -s "$profile_file" ]]; then
            log_success "Memory profile collected: $profile_file"
        else
            log_error "Memory profile is empty"
            return 1
        fi
    else
        log_error "Failed to collect memory profile"
        return 1
    fi
}

# Collect goroutine profile
collect_goroutine_profile() {
    log_info "Collecting goroutine profile..."

    local profile_file="$PROFILES_DIR/goroutine_profile_${TIMESTAMP}.prof"

    if curl -s "$BASE_URL/debug/pprof/goroutine" > "$profile_file"; then
        if [[ -f "$profile_file" && -s "$profile_file" ]]; then
            log_success "Goroutine profile collected: $profile_file"
        else
            log_error "Goroutine profile is empty"
            return 1
        fi
    else
        log_error "Failed to collect goroutine profile"
        return 1
    fi
}

# Generate summary report
generate_summary() {
    log_info "Generating summary report..."

    local summary_file="$RESULTS_DIR/summary_${TIMESTAMP}.md"

    cat > "$summary_file" << EOF
# Performance Baseline Test Summary

**Test Run:** $TIMESTAMP
**Base URL:** $BASE_URL
**Go Version:** $(go version)
**k6 Version:** $(k6 version)

## Test Results

### Webhook Load Test
- Test file: webhook_test_${TIMESTAMP}.json
- Scenarios: constant_load, ramp_up, spike_test
- Target RPS: 1000-2000

### History API Load Test
- Test file: history_test_${TIMESTAMP}.json
- Scenarios: constant_load, ramp_up, pagination_stress
- Target RPS: 500-1000

## Performance Profiles

### CPU Profile
- File: cpu_profile_${TIMESTAMP}.prof
- Duration: 30 seconds
- Collected during load test

### Memory Profile
- File: memory_profile_${TIMESTAMP}.prof
- Heap snapshot

### Goroutine Profile
- File: goroutine_profile_${TIMESTAMP}.prof
- Goroutine stack traces

## Analysis Commands

\`\`\`bash
# Analyze CPU profile
go tool pprof profiles/cpu_profile_${TIMESTAMP}.prof

# Analyze memory profile
go tool pprof profiles/memory_profile_${TIMESTAMP}.prof

# Analyze goroutine profile
go tool pprof profiles/goroutine_profile_${TIMESTAMP}.prof

# View k6 results
k6 run --out json=results/webhook_test_${TIMESTAMP}.json k6-webhook-test.js
k6 run --out json=results/history_test_${TIMESTAMP}.json k6-history-test.js
\`\`\`

## Next Steps

1. Analyze profiles using go tool pprof
2. Compare results with Python version baseline
3. Identify performance bottlenecks
4. Document findings in performance-baseline.md
EOF

    log_success "Summary report generated: $summary_file"
}

# Main execution
main() {
    log_info "Starting performance baseline tests for Go Alert History Service"
    log_info "Timestamp: $TIMESTAMP"
    log_info "Base URL: $BASE_URL"

    # Check dependencies
    check_dependencies

    # Create directories
    create_directories

    # Build and start Go application
    build_go_app
    start_go_app

    # Wait a bit for the application to stabilize
    sleep 5

    # Run load tests
    log_info "=== Running Load Tests ==="
    run_webhook_test
    sleep 10  # Brief pause between tests
    run_history_test

    # Collect performance profiles
    log_info "=== Collecting Performance Profiles ==="
    collect_cpu_profile
    sleep 5
    collect_memory_profile
    sleep 5
    collect_goroutine_profile

    # Generate summary
    generate_summary

    log_success "Performance baseline tests completed successfully!"
    log_info "Results are available in: $RESULTS_DIR"
    log_info "Profiles are available in: $PROFILES_DIR"
}

# Run main function
main "$@"
