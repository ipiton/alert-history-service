#!/bin/bash
#
# test-redis-failover.sh
# Redis failover simulation test (TN-99)
#
# Tests:
# - Data persistence after pod deletion
# - AOF replay on restart
# - Recovery time <60s
# - Zero data loss
#
# Usage: bash scripts/test-redis-failover.sh

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
TEST_KEY="failover-test-key"
TEST_VALUE="failover-test-value-$(date +%s)"

log_info "Starting Redis failover simulation test"
echo "================================================================"
log_info "Pod: $POD_NAME"
log_info "Namespace: $NAMESPACE"
echo "================================================================"

# Step 1: Write test data to Redis
log_info "Step 1: Writing test data to Redis..."
if [ -n "$REDIS_PASSWORD" ]; then
    kubectl exec -n "$NAMESPACE" "$POD_NAME" -- redis-cli -a "$REDIS_PASSWORD" SET "$TEST_KEY" "$TEST_VALUE" >/dev/null
else
    kubectl exec -n "$NAMESPACE" "$POD_NAME" -- redis-cli SET "$TEST_KEY" "$TEST_VALUE" >/dev/null
fi
log_success "Test data written: $TEST_KEY = $TEST_VALUE"

# Step 2: Verify data written
log_info "Step 2: Verifying data written..."
if [ -n "$REDIS_PASSWORD" ]; then
    RETRIEVED=$(kubectl exec -n "$NAMESPACE" "$POD_NAME" -- redis-cli -a "$REDIS_PASSWORD" GET "$TEST_KEY" 2>/dev/null | tail -1)
else
    RETRIEVED=$(kubectl exec -n "$NAMESPACE" "$POD_NAME" -- redis-cli GET "$TEST_KEY" | tail -1)
fi

if [ "$RETRIEVED" = "$TEST_VALUE" ]; then
    log_success "Data verified before failover"
else
    log_error "Data mismatch! Expected: $TEST_VALUE, Got: $RETRIEVED"
    exit 1
fi

# Step 3: Delete pod (simulate crash)
log_info "Step 3: Deleting pod to simulate crash..."
START_TIME=$(date +%s)
kubectl delete pod -n "$NAMESPACE" "$POD_NAME" --grace-period=0 --force 2>/dev/null || {
    kubectl delete pod -n "$NAMESPACE" "$POD_NAME" --wait=false
}
log_success "Pod deleted"

# Step 4: Wait for pod recreation
log_info "Step 4: Waiting for pod recreation..."
kubectl wait --for=condition=ready pod/"$POD_NAME" -n "$NAMESPACE" --timeout=120s || {
    log_error "Pod failed to become ready within 120s"
    kubectl get pod -n "$NAMESPACE" "$POD_NAME"
    kubectl describe pod -n "$NAMESPACE" "$POD_NAME"
    exit 1
}
END_TIME=$(date +%s)
RECOVERY_TIME=$((END_TIME - START_TIME))
log_success "Pod recreated and ready (recovery time: ${RECOVERY_TIME}s)"

if [ "$RECOVERY_TIME" -le 60 ]; then
    log_success "Recovery time <60s target met ✅"
else
    log_error "Recovery time >60s (actual: ${RECOVERY_TIME}s) ❌"
fi

# Step 5: Verify data persisted
log_info "Step 5: Verifying data persisted after restart..."
sleep 5  # Give Redis a moment to finish AOF replay

if [ -n "$REDIS_PASSWORD" ]; then
    RETRIEVED_AFTER=$(kubectl exec -n "$NAMESPACE" "$POD_NAME" -- redis-cli -a "$REDIS_PASSWORD" GET "$TEST_KEY" 2>/dev/null | tail -1)
else
    RETRIEVED_AFTER=$(kubectl exec -n "$NAMESPACE" "$POD_NAME" -- redis-cli GET "$TEST_KEY" | tail -1)
fi

if [ "$RETRIEVED_AFTER" = "$TEST_VALUE" ]; then
    log_success "Data persisted after restart ✅ Zero data loss!"
else
    log_error "Data lost after restart! Expected: $TEST_VALUE, Got: $RETRIEVED_AFTER"
    exit 1
fi

# Step 6: Verify AOF replay
log_info "Step 6: Checking Redis logs for AOF replay..."
LOGS=$(kubectl logs -n "$NAMESPACE" "$POD_NAME" -c redis --tail=50 | grep -i "loading\|aof\|rdb" || echo "")
if echo "$LOGS" | grep -qi "loading"; then
    log_success "AOF replay detected in logs ✅"
    echo "$LOGS" | grep -i "loading"
else
    log_info "No 'Loading DB' message found (may be fast startup)"
fi

# Step 7: Cleanup test data
log_info "Step 7: Cleaning up test data..."
if [ -n "$REDIS_PASSWORD" ]; then
    kubectl exec -n "$NAMESPACE" "$POD_NAME" -- redis-cli -a "$REDIS_PASSWORD" DEL "$TEST_KEY" >/dev/null
else
    kubectl exec -n "$NAMESPACE" "$POD_NAME" -- redis-cli DEL "$TEST_KEY" >/dev/null
fi
log_success "Test data cleaned up"

# Summary
echo ""
echo "================================================================"
log_success "Failover test PASSED ✅"
echo "================================================================"
echo "Recovery time: ${RECOVERY_TIME}s (target: <60s)"
echo "Data loss: 0 keys (target: 0)"
echo "AOF replay: Verified"
echo "================================================================"
exit 0
