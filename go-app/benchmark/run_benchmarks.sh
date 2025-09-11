#!/bin/bash

# HTTP Framework Benchmark Script
# Compares Fiber v2 vs Gin performance

set -e

echo "🚀 HTTP Framework Benchmark: Fiber vs Gin"
echo "=========================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
FIBER_PORT=8080
GIN_PORT=8081
RESULTS_DIR="./results"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# Create results directory
mkdir -p "$RESULTS_DIR"

echo -e "${BLUE}Configuration:${NC}"
echo "  Fiber server port: $FIBER_PORT"
echo "  Gin server port: $GIN_PORT"
echo "  Results directory: $RESULTS_DIR"
echo "  Timestamp: $TIMESTAMP"
echo ""

# Function to check if port is in use
check_port() {
    local port=$1
    if lsof -Pi :"$port" -sTCP:LISTEN -t >/dev/null ; then
        echo -e "${RED}❌ Port $port is already in use${NC}"
        return 1
    else
        echo -e "${GREEN}✅ Port $port is available${NC}"
        return 0
    fi
}

# Function to wait for server to be ready
wait_for_server() {
    local url=$1
    local max_attempts=30
    local attempt=1

    echo -e "${YELLOW}Waiting for server at $url...${NC}"

    while [ $attempt -le $max_attempts ]; do
        if curl -s --max-time 1 "$url" > /dev/null 2>&1; then
            echo -e "${GREEN}✅ Server is ready${NC}"
            return 0
        fi
        echo -e "${YELLOW}Attempt $attempt/$max_attempts...${NC}"
        sleep 1
        ((attempt++))
    done

    echo -e "${RED}❌ Server failed to start${NC}"
    return 1
}

# Function to run benchmark with hey
run_hey_benchmark() {
    local url=$1
    local name=$2
    local concurrency=$3
    local requests=$4
    local output_file=$5

    echo -e "${BLUE}Running hey benchmark for $name...${NC}"
    echo "  URL: $url"
    echo "  Concurrency: $concurrency"
    echo "  Requests: $requests"
    echo "  Output: $output_file"
    echo ""

    hey -n "$requests" -c "$concurrency" -m GET "$url" > "$output_file" 2>&1

    echo -e "${GREEN}✅ Benchmark completed for $name${NC}"
    echo ""
}

# Function to run benchmark with wrk
run_wrk_benchmark() {
    local url=$1
    local name=$2
    local duration=$3
    local threads=$4
    local connections=$5
    local output_file=$6

    echo -e "${BLUE}Running wrk benchmark for $name...${NC}"
    echo "  URL: $url"
    echo "  Duration: ${duration}s"
    echo "  Threads: $threads"
    echo "  Connections: $connections"
    echo "  Output: $output_file"
    echo ""

    wrk -t"$threads" -c"$connections" -d"${duration}"s "$url" > "$output_file" 2>&1

    echo -e "${GREEN}✅ Benchmark completed for $name${NC}"
    echo ""
}

# Check ports
echo -e "${BLUE}Checking ports...${NC}"
check_port $FIBER_PORT || exit 1
check_port $GIN_PORT || exit 1
echo ""

# Build applications
echo -e "${BLUE}Building applications...${NC}"

echo "Building Fiber app..."
cd fiber-app
go build -o fiber-server .
cd ..

echo "Building Gin app..."
cd gin-app
go build -o gin-server .
cd ..

echo -e "${GREEN}✅ Applications built${NC}"
echo ""

# Start servers in background
echo -e "${BLUE}Starting servers...${NC}"

# Start Fiber server
echo "Starting Fiber server on port $FIBER_PORT..."
./fiber-app/fiber-server &
FIBER_PID=$!

# Start Gin server
echo "Starting Gin server on port $GIN_PORT..."
./gin-app/gin-server &
GIN_PID=$!

# Wait for servers to be ready
wait_for_server "http://localhost:$FIBER_PORT/health" || exit 1
wait_for_server "http://localhost:$GIN_PORT/health" || exit 1

echo -e "${GREEN}✅ Both servers are running${NC}"
echo ""

# Run benchmarks
echo -e "${BLUE}Running benchmarks...${NC}"

# Test scenarios
SCENARIOS=(
    "health:10:1000"
    "alerts:50:5000"
    "single-alert:100:10000"
)

for scenario in "${SCENARIOS[@]}"; do
    IFS=':' read -r endpoint concurrency requests <<< "$scenario"

    echo -e "${YELLOW}=== Testing /$endpoint endpoint ===${NC}"

    # Fiber benchmark
    run_hey_benchmark \
        "http://localhost:$FIBER_PORT/$endpoint" \
        "Fiber-$endpoint" \
        "$concurrency" \
        "$requests" \
        "$RESULTS_DIR/fiber_${endpoint}_${TIMESTAMP}.txt"

    # Gin benchmark
    run_hey_benchmark \
        "http://localhost:$GIN_PORT/$endpoint" \
        "Gin-$endpoint" \
        "$concurrency" \
        "$requests" \
        "$RESULTS_DIR/gin_${endpoint}_${TIMESTAMP}.txt"
done

# Run wrk benchmarks for sustained load
echo -e "${YELLOW}=== Running sustained load tests ===${NC}"

run_wrk_benchmark \
    "http://localhost:$FIBER_PORT/api/alerts" \
    "Fiber-sustained" \
    30 \
    4 \
    200 \
    "$RESULTS_DIR/fiber_sustained_${TIMESTAMP}.txt"

run_wrk_benchmark \
    "http://localhost:$GIN_PORT/api/alerts" \
    "Gin-sustained" \
    30 \
    4 \
    200 \
    "$RESULTS_DIR/gin_sustained_${TIMESTAMP}.txt"

# Stop servers
echo -e "${BLUE}Stopping servers...${NC}"
kill $FIBER_PID $GIN_PID 2>/dev/null || true

# Wait for servers to stop
sleep 2

echo -e "${GREEN}✅ Benchmarks completed!${NC}"
echo ""
echo -e "${BLUE}Results saved to: $RESULTS_DIR${NC}"
echo "Files:"
ls -la "$RESULTS_DIR"/*"$TIMESTAMP"*

echo ""
echo -e "${YELLOW}To analyze results, run:${NC}"
echo "python3 analyze_results.py $RESULTS_DIR"
echo ""
echo -e "${GREEN}🎉 Benchmark complete!${NC}"
