#!/bin/bash

# Check Health Monitoring Integration Status
# This script checks if TN-049 integration is enabled or disabled

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
MAIN_FILE="$PROJECT_ROOT/go-app/cmd/server/main.go"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔍 Checking TN-049 Health Monitoring Integration Status${NC}"
echo ""

# Check if main.go exists
if [ ! -f "$MAIN_FILE" ]; then
    echo -e "${RED}❌ Error: $MAIN_FILE not found${NC}"
    exit 1
fi

# Check integration status
if grep -q "^[[:space:]]*k8sClient, err := k8s.NewK8sClient" "$MAIN_FILE"; then
    echo -e "${GREEN}✅ Integration: ENABLED${NC}"
    echo ""
    echo -e "${BLUE}📋 Enabled components:${NC}"

    # Check each component
    if grep -q "^[[:space:]]*k8sClient, err := k8s.NewK8sClient" "$MAIN_FILE"; then
        echo "  ✅ TN-046: K8s Client"
    fi

    if grep -q "^[[:space:]]*discoveryMgr, err := publishing.NewTargetDiscoveryManager" "$MAIN_FILE"; then
        echo "  ✅ TN-047: Target Discovery Manager"
    fi

    if grep -q "^[[:space:]]*refreshManager, err = publishing.NewRefreshManager" "$MAIN_FILE"; then
        echo "  ✅ TN-048: Target Refresh Manager"
    fi

    if grep -q "^[[:space:]]*healthMonitor, err := publishing.NewHealthMonitor" "$MAIN_FILE"; then
        echo "  ✅ TN-049: Health Monitor"
    fi

    echo ""
    echo -e "${BLUE}📝 Code snippet (first 10 lines):${NC}"
    grep -A 10 "k8sClient, err := k8s.NewK8sClient" "$MAIN_FILE" | head -10 | sed 's/^/  /'

    echo ""
    echo -e "${BLUE}ℹ️  To disable:${NC}"
    echo "  ./scripts/disable-health-monitoring.sh"

elif grep -q "^[[:space:]]*// k8sClient, err := k8s.NewK8sClient" "$MAIN_FILE"; then
    echo -e "${YELLOW}⚠️  Integration: DISABLED${NC}"
    echo ""
    echo -e "${BLUE}📋 Disabled components:${NC}"
    echo "  ⏸️  TN-046: K8s Client"
    echo "  ⏸️  TN-047: Target Discovery Manager"
    echo "  ⏸️  TN-048: Target Refresh Manager"
    echo "  ⏸️  TN-049: Health Monitor"

    echo ""
    echo -e "${BLUE}📝 Code snippet (commented):${NC}"
    grep -A 10 "// k8sClient, err := k8s.NewK8sClient" "$MAIN_FILE" | head -10 | sed 's/^/  /'

    echo ""
    echo -e "${BLUE}ℹ️  To enable:${NC}"
    echo "  ./scripts/enable-health-monitoring.sh"
else
    echo -e "${RED}❌ Integration: UNKNOWN STATE${NC}"
    echo ""
    echo "Cannot determine integration status. The code may have been manually modified."
    echo ""
    echo "Expected pattern not found:"
    echo "  k8sClient, err := k8s.NewK8sClient"
    echo "  or"
    echo "  // k8sClient, err := k8s.NewK8sClient"
    exit 1
fi

echo ""
echo -e "${BLUE}📊 Additional checks:${NC}"

# Check if K8s manifests exist
if [ -f "$PROJECT_ROOT/k8s/publishing/serviceaccount.yaml" ]; then
    echo "  ✅ K8s manifests exist (k8s/publishing/)"
else
    echo "  ❌ K8s manifests not found"
fi

# Check if integration guide exists
if [ -f "$PROJECT_ROOT/tasks/go-migration-analysis/TN-049-target-health-monitoring/INTEGRATION_GUIDE.md" ]; then
    echo "  ✅ Integration guide available"
else
    echo "  ❌ Integration guide not found"
fi

# Check if README exists
if [ -f "$PROJECT_ROOT/go-app/internal/business/publishing/HEALTH_MONITORING_README.md" ]; then
    echo "  ✅ Health Monitoring README available"
else
    echo "  ❌ Health Monitoring README not found"
fi

echo ""
echo -e "${GREEN}✅ Status check complete!${NC}"
