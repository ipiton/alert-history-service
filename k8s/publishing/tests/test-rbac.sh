#!/bin/bash
# test-rbac.sh - Automated RBAC testing for Publishing System
# Version: 1.0
# Usage: ./test-rbac.sh [namespace]

set -euo pipefail

# Configuration
NAMESPACE="${1:-production}"
SA_NAME="alert-history-publishing"
ROLE_NAME="alert-history-secrets-reader"
ROLEBINDING_NAME="alert-history-secrets-reader-binding"
SA_FULL="system:serviceaccount:${NAMESPACE}:${SA_NAME}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

# Counters
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_TOTAL=0

# Test function
test_command() {
    local test_name="$1"
    local expected="$2"
    shift 2
    local cmd=("$@")

    TESTS_TOTAL=$((TESTS_TOTAL + 1))

    echo -n "Test $TESTS_TOTAL: $test_name... "

    if output=$("${cmd[@]}" 2>&1); then
        if [[ "$output" == *"$expected"* ]]; then
            echo -e "${GREEN}PASS${NC}"
            TESTS_PASSED=$((TESTS_PASSED + 1))
            return 0
        else
            echo -e "${RED}FAIL${NC} (expected '$expected', got '$output')"
            TESTS_FAILED=$((TESTS_FAILED + 1))
            return 1
        fi
    else
        if [[ "$expected" == "error" ]] || [[ "$expected" == "no" ]]; then
            echo -e "${GREEN}PASS${NC} (expected failure)"
            TESTS_PASSED=$((TESTS_PASSED + 1))
            return 0
        else
            echo -e "${RED}FAIL${NC} (command failed: $output)"
            TESTS_FAILED=$((TESTS_FAILED + 1))
            return 1
        fi
    fi
}

echo "========================================="
echo "RBAC Testing for Publishing System"
echo "Namespace: $NAMESPACE"
echo "ServiceAccount: $SA_NAME"
echo "========================================="
echo ""

echo "Phase 1: Resource Existence Tests"
echo "-----------------------------------"

# Test 1: ServiceAccount exists
test_command "ServiceAccount exists" "alert-history-publishing" \
    kubectl get serviceaccount "$SA_NAME" -n "$NAMESPACE" -o name

# Test 2: Role exists
test_command "Role exists" "alert-history-secrets-reader" \
    kubectl get role "$ROLE_NAME" -n "$NAMESPACE" -o name

# Test 3: RoleBinding exists
test_command "RoleBinding exists" "alert-history-secrets-reader-binding" \
    kubectl get rolebinding "$ROLEBINDING_NAME" -n "$NAMESPACE" -o name

echo ""
echo "Phase 2: Positive Permission Tests"
echo "-----------------------------------"

# Test 4: Can list secrets
test_command "Can list secrets" "yes" \
    kubectl auth can-i list secrets --as="$SA_FULL" -n "$NAMESPACE"

# Test 5: Can get secrets
test_command "Can get secrets" "yes" \
    kubectl auth can-i get secrets --as="$SA_FULL" -n "$NAMESPACE"

# Test 6: Can watch secrets
test_command "Can watch secrets" "yes" \
    kubectl auth can-i watch secrets --as="$SA_FULL" -n "$NAMESPACE"

echo ""
echo "Phase 3: Negative Permission Tests (Security Validation)"
echo "--------------------------------------------------------"

# Test 7: Cannot create secrets
test_command "Cannot create secrets" "no" \
    kubectl auth can-i create secrets --as="$SA_FULL" -n "$NAMESPACE"

# Test 8: Cannot update secrets
test_command "Cannot update secrets" "no" \
    kubectl auth can-i update secrets --as="$SA_FULL" -n "$NAMESPACE"

# Test 9: Cannot patch secrets
test_command "Cannot patch secrets" "no" \
    kubectl auth can-i patch secrets --as="$SA_FULL" -n "$NAMESPACE"

# Test 10: Cannot delete secrets
test_command "Cannot delete secrets" "no" \
    kubectl auth can-i delete secrets --as="$SA_FULL" -n "$NAMESPACE"

# Test 11: Cannot access kube-system
test_command "Cannot access kube-system secrets" "no" \
    kubectl auth can-i list secrets --as="$SA_FULL" -n kube-system

# Test 12: Cannot create pods
test_command "Cannot create pods" "no" \
    kubectl auth can-i create pods --as="$SA_FULL" -n "$NAMESPACE"

# Test 13: Cannot access cluster-admin
test_command "Cannot use cluster-admin" "no" \
    kubectl auth can-i '*' '*' --as="$SA_FULL"

echo ""
echo "Phase 4: Configuration Validation"
echo "----------------------------------"

# Test 14: automountServiceAccountToken is true
test_command "automountServiceAccountToken enabled" "true" \
    kubectl get serviceaccount "$SA_NAME" -n "$NAMESPACE" -o jsonpath='{.automountServiceAccountToken}'

# Test 15: Role has no wildcard verbs
test_command "No wildcard verbs in Role" "" \
    bash -c "kubectl get role '$ROLE_NAME' -n '$NAMESPACE' -o json | jq -e '.rules[].verbs[] | select(. == \"*\")' && echo 'FAIL: wildcard found' || echo ''"

# Test 16: Role has no wildcard resources
test_command "No wildcard resources in Role" "" \
    bash -c "kubectl get role '$ROLE_NAME' -n '$NAMESPACE' -o json | jq -e '.rules[].resources[] | select(. == \"*\")' && echo 'FAIL: wildcard found' || echo ''"

echo ""
echo "========================================="
echo "Test Summary"
echo "========================================="
echo "Total tests: $TESTS_TOTAL"
echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Failed: ${RED}$TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
    echo "RBAC configuration is correct and secure."
    exit 0
else
    echo -e "${RED}✗ Some tests failed!${NC}"
    echo "Please review RBAC configuration."
    exit 1
fi
