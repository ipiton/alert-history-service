#!/bin/bash

# Disable Health Monitoring Integration (TN-049)
# This script comments out the integration code in main.go

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

echo -e "${BLUE}🔧 Disabling TN-049 Health Monitoring Integration${NC}"
echo ""

# Check if main.go exists
if [ ! -f "$MAIN_FILE" ]; then
    echo -e "${RED}❌ Error: $MAIN_FILE not found${NC}"
    exit 1
fi

# Check if already disabled
if grep -q "^[[:space:]]*// k8sClient, err := k8s.NewK8sClient" "$MAIN_FILE"; then
    echo -e "${YELLOW}⚠️  Integration already disabled!${NC}"
    echo ""
    echo "Current status:"
    grep -A 5 "// k8sClient, err := k8s.NewK8sClient" "$MAIN_FILE" | head -6
    echo ""
    echo -e "${BLUE}ℹ️  If you want to re-disable, run:${NC}"
    echo "  ./scripts/enable-health-monitoring.sh"
    echo "  ./scripts/disable-health-monitoring.sh"
    exit 0
fi

# Create backup
BACKUP_FILE="${MAIN_FILE}.backup.$(date +%Y%m%d_%H%M%S)"
cp "$MAIN_FILE" "$BACKUP_FILE"
echo -e "${GREEN}✅ Backup created: $BACKUP_FILE${NC}"

# Comment lines 809-947 (Publishing System block)
# Strategy: Add "// " prefix to non-empty lines
sed -i.tmp '809,947s|^\([[:space:]]*\)\([^/[:space:]].*\)|\1// \2|' "$MAIN_FILE"
rm "${MAIN_FILE}.tmp"

# Verify changes
if grep -q "^[[:space:]]*// k8sClient, err := k8s.NewK8sClient" "$MAIN_FILE"; then
    echo -e "${GREEN}✅ Integration disabled successfully!${NC}"
    echo ""
    echo -e "${BLUE}📋 What was disabled:${NC}"
    echo "  - TN-046: K8s Client initialization"
    echo "  - TN-047: Target Discovery Manager"
    echo "  - TN-048: Target Refresh Manager"
    echo "  - TN-049: Health Monitor"
    echo ""
    echo -e "${BLUE}📝 Next steps:${NC}"
    echo "  1. Review changes: git diff $MAIN_FILE"
    echo "  2. Test compilation: cd go-app && go build ./cmd/server"
    echo "  3. Deploy: docker build && kubectl apply"
    echo ""
    echo -e "${BLUE}ℹ️  To re-enable integration:${NC}"
    echo "  ./scripts/enable-health-monitoring.sh"
    echo ""
    echo -e "${GREEN}✅ Done!${NC}"
else
    echo -e "${RED}❌ Error: Failed to disable integration${NC}"
    echo "Restoring backup..."
    cp "$BACKUP_FILE" "$MAIN_FILE"
    exit 1
fi
