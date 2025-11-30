#!/bin/bash
#
# test-redis-helm-templates.sh
# Helm template rendering tests for Redis/Valkey StatefulSet (TN-99)
#
# Tests:
# 1. Template renders for Standard Profile
# 2. No Redis for Lite Profile
# 3. ConfigMap rendered correctly
# 4. Services created (3 Redis services)
# 5. ServiceMonitor conditional (with/without monitoring)
#
# Usage: bash scripts/test-redis-helm-templates.sh

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
log_info() {
    echo -e "${YELLOW}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

test_start() {
    TESTS_RUN=$((TESTS_RUN + 1))
    log_info "Test $TESTS_RUN: $1"
}

test_pass() {
    TESTS_PASSED=$((TESTS_PASSED + 1))
    log_success "$1"
}

test_fail() {
    TESTS_FAILED=$((TESTS_FAILED + 1))
    log_error "$1"
}

# Change to project root
cd "$(dirname "$0")/.." || exit 1

# Ensure Helm chart directory exists
if [ ! -d "helm/alert-history" ]; then
    log_error "Helm chart directory not found: helm/alert-history"
    exit 1
fi

log_info "Starting Helm template rendering tests for TN-99 Redis StatefulSet"
echo "================================================================"

# ====================
# Test 1: Template renders for Standard Profile
# ====================
test_start "Template renders for Standard Profile"
OUTPUT=$(helm template alerthistory ./helm/alert-history --set profile=standard 2>&1) || {
    test_fail "Helm template rendering failed for Standard Profile"
    echo "$OUTPUT"
    exit 1
}

if echo "$OUTPUT" | grep -q "kind: StatefulSet"; then
    COUNT=$(echo "$OUTPUT" | grep -c "kind: StatefulSet" || true)
    if [ "$COUNT" -eq 2 ]; then
        test_pass "Standard Profile renders 2 StatefulSets (PostgreSQL + Redis)"
    else
        test_fail "Expected 2 StatefulSets, found $COUNT"
    fi
else
    test_fail "No StatefulSet found in Standard Profile output"
fi

# ====================
# Test 2: No Redis for Lite Profile
# ====================
test_start "No Redis for Lite Profile"
OUTPUT=$(helm template alerthistory ./helm/alert-history --set profile=lite 2>&1) || {
    test_fail "Helm template rendering failed for Lite Profile"
    exit 1
}

# Count StatefulSets (should be 0 for Lite profile)
COUNT=$(echo "$OUTPUT" | grep -c "kind: StatefulSet" || true)
if [ "$COUNT" -eq 0 ]; then
    test_pass "Lite Profile renders 0 StatefulSets (no PostgreSQL, no Redis)"
else
    test_fail "Expected 0 StatefulSets for Lite Profile, found $COUNT"
fi

# ====================
# Test 3: ConfigMap rendered correctly
# ====================
test_start "ConfigMap rendered correctly"
OUTPUT=$(helm template alerthistory ./helm/alert-history --set profile=standard 2>&1)

if echo "$OUTPUT" | grep -q "maxmemory"; then
    # Extract maxmemory value
    MAXMEMORY=$(echo "$OUTPUT" | grep "maxmemory" | head -1 | awk '{print $2}')
    if [ "$MAXMEMORY" = "384mb" ]; then
        test_pass "ConfigMap contains maxmemory 384mb (default)"
    else
        test_fail "Expected maxmemory=384mb, found: $MAXMEMORY"
    fi
else
    test_fail "ConfigMap does not contain 'maxmemory' setting"
fi

# ====================
# Test 4: Services created (3 Redis services)
# ====================
test_start "Services created (3 Redis services)"
OUTPUT=$(helm template alerthistory ./helm/alert-history --set profile=standard 2>&1)

# Count Redis-specific services (by name pattern)
REDIS_SERVICES=$(echo "$OUTPUT" | grep "name: alerthistory-redis" | grep -c "kind: Service" || echo 0)

if [ "$REDIS_SERVICES" -ge 3 ]; then
    test_pass "Found $REDIS_SERVICES Redis services (headless, ClusterIP, metrics)"
else
    test_fail "Expected at least 3 Redis services, found $REDIS_SERVICES"
fi

# ====================
# Test 5: ServiceMonitor conditional (with monitoring)
# ====================
test_start "ServiceMonitor created with monitoring enabled"
OUTPUT=$(helm template alerthistory ./helm/alert-history \
    --set profile=standard \
    --set monitoring.prometheusEnabled=true 2>&1)

if echo "$OUTPUT" | grep -q "kind: ServiceMonitor"; then
    test_pass "ServiceMonitor created when monitoring enabled"
else
    test_fail "ServiceMonitor not found when monitoring enabled"
fi

# ====================
# Test 6: ServiceMonitor conditional (without monitoring)
# ====================
test_start "ServiceMonitor NOT created with monitoring disabled"
OUTPUT=$(helm template alerthistory ./helm/alert-history \
    --set profile=standard \
    --set monitoring.prometheusEnabled=false 2>&1)

if echo "$OUTPUT" | grep -q "kind: ServiceMonitor"; then
    test_fail "ServiceMonitor found when monitoring disabled (should be absent)"
else
    test_pass "ServiceMonitor correctly absent when monitoring disabled"
fi

# ====================
# Test 7: PrometheusRule conditional (with monitoring)
# ====================
test_start "PrometheusRule created with monitoring enabled"
OUTPUT=$(helm template alerthistory ./helm/alert-history \
    --set profile=standard \
    --set monitoring.prometheusEnabled=true 2>&1)

if echo "$OUTPUT" | grep -q "kind: PrometheusRule"; then
    test_pass "PrometheusRule created when monitoring enabled"
else
    test_fail "PrometheusRule not found when monitoring enabled"
fi

# ====================
# Test 8: NetworkPolicy conditional (enabled)
# ====================
test_start "NetworkPolicy created when enabled"
OUTPUT=$(helm template alerthistory ./helm/alert-history \
    --set profile=standard \
    --set valkey.networkPolicy.enabled=true 2>&1)

if echo "$OUTPUT" | grep -q "kind: NetworkPolicy"; then
    test_pass "NetworkPolicy created when enabled"
else
    test_fail "NetworkPolicy not found when enabled"
fi

# ====================
# Test 9: NetworkPolicy conditional (disabled)
# ====================
test_start "NetworkPolicy NOT created when disabled"
OUTPUT=$(helm template alerthistory ./helm/alert-history \
    --set profile=standard \
    --set valkey.networkPolicy.enabled=false 2>&1)

# Count NetworkPolicies (should not contain Redis NetworkPolicy)
COUNT=$(echo "$OUTPUT" | grep "alerthistory-redis" | grep -c "kind: NetworkPolicy" || echo 0)
if [ "$COUNT" -eq 0 ]; then
    test_pass "Redis NetworkPolicy correctly absent when disabled"
else
    test_fail "Redis NetworkPolicy found when disabled (should be absent)"
fi

# ====================
# Summary
# ====================
echo ""
echo "================================================================"
log_info "Test Summary"
echo "================================================================"
echo "Total tests run: $TESTS_RUN"
echo -e "Tests passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests failed: ${RED}$TESTS_FAILED${NC}"

if [ "$TESTS_FAILED" -eq 0 ]; then
    echo ""
    log_success "All tests passed! ✅"
    exit 0
else
    echo ""
    log_error "$TESTS_FAILED test(s) failed ❌"
    exit 1
fi
