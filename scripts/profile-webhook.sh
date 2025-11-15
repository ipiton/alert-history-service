#!/bin/bash
# TN-061: Webhook Endpoint Performance Profiling Script
#
# This script profiles the webhook endpoint using pprof
# Profiles: CPU, Memory, Goroutines, Block, Mutex
#
# Usage: ./scripts/profile-webhook.sh [profile_type] [duration]
# Example: ./scripts/profile-webhook.sh cpu 30s

set -e

# Configuration
PROFILE_TYPE="${1:-cpu}"
DURATION="${2:-30s}"
SERVICE_URL="${SERVICE_URL:-http://localhost:8080}"
OUTPUT_DIR="${OUTPUT_DIR:-./profiles}"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Webhook Endpoint Profiling ===${NC}"
echo "Profile type: ${PROFILE_TYPE}"
echo "Duration: ${DURATION}"
echo "Service URL: ${SERVICE_URL}"
echo ""

# Create output directory
mkdir -p "${OUTPUT_DIR}"

# Check if service is running
echo -e "${YELLOW}Checking service health...${NC}"
if ! curl -sf "${SERVICE_URL}/healthz" > /dev/null; then
    echo -e "${RED}Error: Service is not responding at ${SERVICE_URL}${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Service is healthy${NC}"
echo ""

# Function to run CPU profile
profile_cpu() {
    echo -e "${YELLOW}Starting CPU profile (${DURATION})...${NC}"
    OUTPUT_FILE="${OUTPUT_DIR}/cpu_${TIMESTAMP}.prof"

    # Start profiling in background
    curl -sf "${SERVICE_URL}/debug/pprof/profile?seconds=${DURATION//s/}" > "${OUTPUT_FILE}" &
    PROFILE_PID=$!

    # Generate load during profiling
    echo -e "${YELLOW}Generating load...${NC}"
    for i in {1..1000}; do
        curl -sf -X POST "${SERVICE_URL}/webhook" \
            -H "Content-Type: application/json" \
            -d '{"alerts":[{"status":"firing","labels":{"alertname":"ProfileTest"}}]}' \
            > /dev/null 2>&1 &
    done

    # Wait for profiling to complete
    wait $PROFILE_PID

    echo -e "${GREEN}✓ CPU profile saved: ${OUTPUT_FILE}${NC}"
    echo ""
    echo "View with: go tool pprof -http=:8081 ${OUTPUT_FILE}"
    echo "Or: go tool pprof ${OUTPUT_FILE}"
}

# Function to run memory profile
profile_memory() {
    echo -e "${YELLOW}Starting memory profile...${NC}"
    OUTPUT_FILE="${OUTPUT_DIR}/memory_${TIMESTAMP}.prof"

    # Generate some load first
    echo -e "${YELLOW}Generating load...${NC}"
    for i in {1..5000}; do
        curl -sf -X POST "${SERVICE_URL}/webhook" \
            -H "Content-Type: application/json" \
            -d '{"alerts":[{"status":"firing","labels":{"alertname":"ProfileTest"}}]}' \
            > /dev/null 2>&1
    done

    # Take memory snapshot
    curl -sf "${SERVICE_URL}/debug/pprof/heap" > "${OUTPUT_FILE}"

    echo -e "${GREEN}✓ Memory profile saved: ${OUTPUT_FILE}${NC}"
    echo ""
    echo "View with: go tool pprof -http=:8081 ${OUTPUT_FILE}"
    echo "Check allocations: go tool pprof -alloc_space ${OUTPUT_FILE}"
    echo "Check in-use: go tool pprof -inuse_space ${OUTPUT_FILE}"
}

# Function to run goroutine profile
profile_goroutines() {
    echo -e "${YELLOW}Starting goroutine profile...${NC}"
    OUTPUT_FILE="${OUTPUT_DIR}/goroutine_${TIMESTAMP}.prof"

    # Generate concurrent load
    echo -e "${YELLOW}Generating concurrent load...${NC}"
    for i in {1..100}; do
        curl -sf -X POST "${SERVICE_URL}/webhook" \
            -H "Content-Type: application/json" \
            -d '{"alerts":[{"status":"firing","labels":{"alertname":"ProfileTest"}}]}' \
            > /dev/null 2>&1 &
    done

    sleep 2

    # Take goroutine snapshot
    curl -sf "${SERVICE_URL}/debug/pprof/goroutine" > "${OUTPUT_FILE}"

    echo -e "${GREEN}✓ Goroutine profile saved: ${OUTPUT_FILE}${NC}"
    echo ""
    echo "View with: go tool pprof -http=:8081 ${OUTPUT_FILE}"
}

# Function to run block profile
profile_block() {
    echo -e "${YELLOW}Starting block profile (${DURATION})...${NC}"
    OUTPUT_FILE="${OUTPUT_DIR}/block_${TIMESTAMP}.prof"

    # Enable block profiling first (requires restart or runtime call)
    echo -e "${YELLOW}Note: Block profiling should be enabled in the service${NC}"

    # Generate load
    echo -e "${YELLOW}Generating load...${NC}"
    for i in {1..2000}; do
        curl -sf -X POST "${SERVICE_URL}/webhook" \
            -H "Content-Type: application/json" \
            -d '{"alerts":[{"status":"firing","labels":{"alertname":"ProfileTest"}}]}' \
            > /dev/null 2>&1 &
    done

    sleep 5

    # Take block profile
    curl -sf "${SERVICE_URL}/debug/pprof/block" > "${OUTPUT_FILE}"

    echo -e "${GREEN}✓ Block profile saved: ${OUTPUT_FILE}${NC}"
    echo ""
    echo "View with: go tool pprof -http=:8081 ${OUTPUT_FILE}"
}

# Function to run mutex profile
profile_mutex() {
    echo -e "${YELLOW}Starting mutex profile...${NC}"
    OUTPUT_FILE="${OUTPUT_DIR}/mutex_${TIMESTAMP}.prof"

    echo -e "${YELLOW}Note: Mutex profiling should be enabled in the service${NC}"

    # Generate concurrent load
    for i in {1..2000}; do
        curl -sf -X POST "${SERVICE_URL}/webhook" \
            -H "Content-Type: application/json" \
            -d '{"alerts":[{"status":"firing","labels":{"alertname":"ProfileTest"}}]}' \
            > /dev/null 2>&1 &
    done

    sleep 5

    # Take mutex profile
    curl -sf "${SERVICE_URL}/debug/pprof/mutex" > "${OUTPUT_FILE}"

    echo -e "${GREEN}✓ Mutex profile saved: ${OUTPUT_FILE}${NC}"
    echo ""
    echo "View with: go tool pprof -http=:8081 ${OUTPUT_FILE}"
}

# Function to run all profiles
profile_all() {
    echo -e "${YELLOW}Running all profiles...${NC}"
    echo ""

    profile_cpu
    sleep 2

    profile_memory
    sleep 2

    profile_goroutines
    sleep 2

    profile_block
    sleep 2

    profile_mutex

    echo ""
    echo -e "${GREEN}=== All profiles complete ===${NC}"
    echo "Profiles saved in: ${OUTPUT_DIR}"
    ls -lh "${OUTPUT_DIR}"
}

# Main
case "${PROFILE_TYPE}" in
    cpu)
        profile_cpu
        ;;
    memory|heap)
        profile_memory
        ;;
    goroutine|goroutines)
        profile_goroutines
        ;;
    block)
        profile_block
        ;;
    mutex)
        profile_mutex
        ;;
    all)
        profile_all
        ;;
    *)
        echo -e "${RED}Error: Unknown profile type: ${PROFILE_TYPE}${NC}"
        echo ""
        echo "Usage: $0 [profile_type] [duration]"
        echo ""
        echo "Profile types:"
        echo "  cpu        - CPU profile (default)"
        echo "  memory     - Memory/heap profile"
        echo "  goroutine  - Goroutine profile"
        echo "  block      - Block profile"
        echo "  mutex      - Mutex profile"
        echo "  all        - Run all profiles"
        echo ""
        echo "Example: $0 cpu 30s"
        exit 1
        ;;
esac

echo ""
echo -e "${GREEN}=== Profiling Complete ===${NC}"
echo ""
echo "Next steps:"
echo "1. Analyze profiles with: go tool pprof -http=:8081 <profile_file>"
echo "2. Look for hot spots (CPU), allocations (memory), goroutine leaks"
echo "3. Check blocking operations (block profile)"
echo "4. Identify mutex contention (mutex profile)"
echo ""
echo "Tip: Compare profiles over time to track improvements"
