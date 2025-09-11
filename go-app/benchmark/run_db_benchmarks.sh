#!/bin/bash

# Database Driver Benchmark Script
# Compares pgx vs GORM performance

set -e

echo "🗄️  Database Driver Benchmark: pgx vs GORM"
echo "=========================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PGX_PORT=8082
GORM_PORT=8083
RESULTS_DIR="./results"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")

# Create results directory
mkdir -p "$RESULTS_DIR"

echo -e "${BLUE}Configuration:${NC}"
echo "  pgx server port: $PGX_PORT"
echo "  GORM server port: $GORM_PORT"
echo "  Results directory: $RESULTS_DIR"
echo ""

# Function to check if PostgreSQL is available
check_postgres() {
    if ! command -v psql &> /dev/null; then
        echo -e "${RED}❌ PostgreSQL client not found. Please install PostgreSQL.${NC}"
        return 1
    fi

    # Try to connect to database
    if ! psql -h localhost -U postgres -d benchmark_db -c "SELECT 1;" &> /dev/null; then
        echo -e "${YELLOW}⚠️  PostgreSQL not available or database 'benchmark_db' doesn't exist${NC}"
        echo -e "${YELLOW}ℹ️  Please start PostgreSQL and create benchmark_db database${NC}"
        echo -e "${YELLOW}   Commands:${NC}"
        echo -e "${YELLOW}   createdb benchmark_db${NC}"
        return 1
    fi

    echo -e "${GREEN}✅ PostgreSQL is available${NC}"
    return 0
}

# Function to run database benchmark
run_db_benchmark() {
    local url=$1
    local name=$2
    local concurrency=$3
    local requests=$4
    local output_file=$5

    echo -e "${BLUE}Running database benchmark for $name...${NC}"
    echo "  URL: $url"
    echo "  Concurrency: $concurrency"
    echo "  Requests: $requests"
    echo "  Output: $output_file"
    echo ""

    hey -n "$requests" -c "$concurrency" -m GET "$url" > "$output_file" 2>&1

    echo -e "${GREEN}✅ Database benchmark completed for $name${NC}"
    echo ""
}

# Check PostgreSQL availability
if ! check_postgres; then
    echo -e "${YELLOW}⚠️  Skipping database benchmarks${NC}"
    exit 0
fi

# Start database servers
echo -e "${BLUE}Starting database servers...${NC}"

# Start pgx server
echo "Starting pgx server on port $PGX_PORT..."
./db-pgx/pgx-server &
PGX_PID=$!

# Start GORM server
echo "Starting GORM server on port $GORM_PORT..."
./db-gorm/gorm-server &
GORM_PID=$!

# Wait for servers to be ready
sleep 3

# Test scenarios for database operations
SCENARIOS=(
    "health:10:1000"
    "alerts:25:2500"
)

echo -e "${BLUE}Running database benchmarks...${NC}"

for scenario in "${SCENARIOS[@]}"; do
    IFS=':' read -r endpoint concurrency requests <<< "$scenario"

    echo -e "${YELLOW}=== Testing /$endpoint endpoint ===${NC}"

    # pgx benchmark
    run_db_benchmark \
        "http://localhost:$PGX_PORT/$endpoint" \
        "pgx-$endpoint" \
        "$concurrency" \
        "$requests" \
        "$RESULTS_DIR/pgx_${endpoint}_${TIMESTAMP}.txt"

    # GORM benchmark
    run_db_benchmark \
        "http://localhost:$GORM_PORT/$endpoint" \
        "GORM-$endpoint" \
        "$concurrency" \
        "$requests" \
        "$RESULTS_DIR/gorm_${endpoint}_${TIMESTAMP}.txt"
done

# Test bulk insert operations
echo -e "${YELLOW}=== Testing bulk insert operations ===${NC}"

# Generate test data for bulk insert
TEST_DATA='[
    {"title": "Bulk Alert 1", "description": "Test bulk insert 1", "severity": "warning"},
    {"title": "Bulk Alert 2", "description": "Test bulk insert 2", "severity": "error"},
    {"title": "Bulk Alert 3", "description": "Test bulk insert 3", "severity": "info"}
]'

echo "$TEST_DATA" | hey -n 100 -c 10 -m POST \
    -H "Content-Type: application/json" \
    -d @- \
    "http://localhost:$PGX_PORT/api/alerts/bulk" > "$RESULTS_DIR/pgx_bulk_insert_${TIMESTAMP}.txt" 2>&1

echo "$TEST_DATA" | hey -n 100 -c 10 -m POST \
    -H "Content-Type: application/json" \
    -d @- \
    "http://localhost:$GORM_PORT/api/alerts/bulk" > "$RESULTS_DIR/gorm_bulk_insert_${TIMESTAMP}.txt" 2>&1

echo -e "${GREEN}✅ Bulk insert benchmarks completed${NC}"
echo ""

# Stop servers
echo -e "${BLUE}Stopping database servers...${NC}"
kill $PGX_PID $GORM_PID 2>/dev/null || true

# Wait for servers to stop
sleep 2

echo -e "${GREEN}✅ Database benchmarks completed!${NC}"
echo ""
echo -e "${BLUE}Results saved to: $RESULTS_DIR${NC}"
echo "Files:"
ls -la "$RESULTS_DIR"/*"$TIMESTAMP"*

echo ""
echo -e "${YELLOW}To analyze results, run:${NC}"
echo "python3 analyze_db_results.py $RESULTS_DIR"
echo ""
echo -e "${GREEN}🎉 Database benchmark complete!${NC}"
