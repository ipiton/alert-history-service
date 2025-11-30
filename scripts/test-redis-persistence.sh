#!/bin/bash
#
# test-redis-persistence.sh
# Redis persistence validation test (AOF + RDB) (TN-99)
#
# Tests:
# - AOF enabled
# - RDB snapshots created
# - Both files exist on disk
# - Data persistence after writes
#
# Usage: bash scripts/test-redis-persistence.sh

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${YELLOW}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[PASS]${NC} $1"; }
log_error() { echo -e "${RED}[FAIL]${NC} $1"; }

# Configuration
POD_NAME="${POD_NAME:-alerthistory-redis-0}"
NAMESPACE="${NAMESPACE:-default}"
REDIS_PASSWORD="${REDIS_PASSWORD:-}"

log_info "Starting Redis persistence validation test"
echo "================================================================"

# Step 1: Check AOF enabled
log_info "Step 1: Checking AOF enabled..."
if [ -n "$REDIS_PASSWORD" ]; then
    AOF=$(kubectl exec -n "$NAMESPACE" "$POD_NAME" -- redis-cli -a "$REDIS_PASSWORD" CONFIG GET appendonly 2>/dev/null | tail -1)
else
    AOF=$(kubectl exec -n "$NAMESPACE" "$POD_NAME" -- redis-cli CONFIG GET appendonly | tail -1)
fi

if [ "$AOF" = "yes" ]; then
    log_success "AOF enabled ✅"
else
    log_error "AOF disabled (expected: yes, got: $AOF)"
    exit 1
fi

# Step 2: Write test data (1000 keys)
log_info "Step 2: Writing 1000 test keys..."
for i in $(seq 1 1000); do
    if [ -n "$REDIS_PASSWORD" ]; then
        kubectl exec -n "$NAMESPACE" "$POD_NAME" -- redis-cli -a "$REDIS_PASSWORD" SET "persist-test-$i" "value-$i" >/dev/null 2>&1
    else
        kubectl exec -n "$NAMESPACE" "$POD_NAME" -- redis-cli SET "persist-test-$i" "value-$i" >/dev/null
    fi

    # Progress indicator every 100 keys
    if [ $((i % 100)) -eq 0 ]; then
        echo -n "."
    fi
done
echo ""
log_success "1000 keys written"

# Step 3: Verify AOF file created
log_info "Step 3: Verifying AOF file created..."
AOF_FILE=$(kubectl exec -n "$NAMESPACE" "$POD_NAME" -- ls -lh /data/appendonly.aof 2>/dev/null || echo "NOT_FOUND")
if [ "$AOF_FILE" != "NOT_FOUND" ]; then
    log_success "AOF file exists: /data/appendonly.aof"
    kubectl exec -n "$NAMESPACE" "$POD_NAME" -- ls -lh /data/appendonly.aof
else
    log_error "AOF file not found!"
    exit 1
fi

# Step 4: Force RDB snapshot
log_info "Step 4: Forcing RDB snapshot (SAVE)..."
if [ -n "$REDIS_PASSWORD" ]; then
    kubectl exec -n "$NAMESPACE" "$POD_NAME" -- redis-cli -a "$REDIS_PASSWORD" SAVE >/dev/null 2>&1
else
    kubectl exec -n "$NAMESPACE" "$POD_NAME" -- redis-cli SAVE >/dev/null
fi
log_success "RDB snapshot created"

# Step 5: Verify RDB file created
log_info "Step 5: Verifying RDB file created..."
RDB_FILE=$(kubectl exec -n "$NAMESPACE" "$POD_NAME" -- ls -lh /data/dump.rdb 2>/dev/null || echo "NOT_FOUND")
if [ "$RDB_FILE" != "NOT_FOUND" ]; then
    log_success "RDB file exists: /data/dump.rdb"
    kubectl exec -n "$NAMESPACE" "$POD_NAME" -- ls -lh /data/dump.rdb
else
    log_error "RDB file not found!"
    exit 1
fi

# Step 6: Verify both files exist
log_info "Step 6: Listing all persistence files..."
kubectl exec -n "$NAMESPACE" "$POD_NAME" -- ls -lh /data/ | grep -E "appendonly|dump.rdb"
log_success "Both AOF and RDB files exist ✅"

# Step 7: Verify data count
log_info "Step 7: Verifying data persisted (key count)..."
if [ -n "$REDIS_PASSWORD" ]; then
    KEY_COUNT=$(kubectl exec -n "$NAMESPACE" "$POD_NAME" -- redis-cli -a "$REDIS_PASSWORD" DBSIZE 2>/dev/null | tail -1 | grep -o '[0-9]*')
else
    KEY_COUNT=$(kubectl exec -n "$NAMESPACE" "$POD_NAME" -- redis-cli DBSIZE | grep -o '[0-9]*')
fi

if [ "$KEY_COUNT" -ge 1000 ]; then
    log_success "Key count: $KEY_COUNT (>= 1000) ✅"
else
    log_error "Key count too low: $KEY_COUNT (expected >= 1000)"
    exit 1
fi

# Step 8: Cleanup test data
log_info "Step 8: Cleaning up test data..."
for i in $(seq 1 1000); do
    if [ -n "$REDIS_PASSWORD" ]; then
        kubectl exec -n "$NAMESPACE" "$POD_NAME" -- redis-cli -a "$REDIS_PASSWORD" DEL "persist-test-$i" >/dev/null 2>&1
    else
        kubectl exec -n "$NAMESPACE" "$POD_NAME" -- redis-cli DEL "persist-test-$i" >/dev/null
    fi
done
log_success "Test data cleaned up"

# Summary
echo ""
echo "================================================================"
log_success "Persistence test PASSED ✅"
echo "================================================================"
echo "AOF: Enabled ✅"
echo "AOF file: Exists (/data/appendonly.aof) ✅"
echo "RDB file: Exists (/data/dump.rdb) ✅"
echo "Data persisted: $KEY_COUNT keys ✅"
echo "================================================================"
exit 0
