#!/bin/bash

# Enable Health Monitoring Integration (TN-049)
# This script uncomments the integration code in main.go

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

echo -e "${BLUE}🔧 Enabling TN-049 Health Monitoring Integration${NC}"
echo ""

# Check if main.go exists
if [ ! -f "$MAIN_FILE" ]; then
    echo -e "${RED}❌ Error: $MAIN_FILE not found${NC}"
    exit 1
fi

# Check if already enabled
if grep -q "^[[:space:]]*k8sClient, err := k8s.NewK8sClient" "$MAIN_FILE"; then
    echo -e "${YELLOW}⚠️  Integration already enabled!${NC}"
    echo ""
    echo "Current status:"
    grep -A 5 "k8sClient, err := k8s.NewK8sClient" "$MAIN_FILE" | head -6
    echo ""
    echo -e "${BLUE}ℹ️  If you want to re-enable, run:${NC}"
    echo "  ./scripts/disable-health-monitoring.sh"
    echo "  ./scripts/enable-health-monitoring.sh"
    exit 0
fi

# Create backup
BACKUP_FILE="${MAIN_FILE}.backup.$(date +%Y%m%d_%H%M%S)"
cp "$MAIN_FILE" "$BACKUP_FILE"
echo -e "${GREEN}✅ Backup created: $BACKUP_FILE${NC}"

# Uncomment lines 809-947 (Publishing System block)
# Strategy: Remove leading "// " from lines starting with whitespace + //
sed -i.tmp '809,947s|^\([[:space:]]*\)// \(.*\)|\1\2|' "$MAIN_FILE"
rm "${MAIN_FILE}.tmp"

# Verify changes
if grep -q "^[[:space:]]*k8sClient, err := k8s.NewK8sClient" "$MAIN_FILE"; then
    echo -e "${GREEN}✅ Integration enabled successfully!${NC}"
    echo ""
    echo -e "${BLUE}📋 What was enabled:${NC}"
    echo "  - TN-046: K8s Client initialization"
    echo "  - TN-047: Target Discovery Manager"
    echo "  - TN-048: Target Refresh Manager"
    echo "  - TN-049: Health Monitor"
    echo ""
    echo -e "${BLUE}📝 Next steps:${NC}"
    echo "  1. Review changes: git diff $MAIN_FILE"
    echo "  2. Test compilation: cd go-app && go build ./cmd/server"
    echo "  3. Create K8s RBAC: kubectl apply -f k8s/publishing/"
    echo "  4. Deploy to K8s: helm upgrade alert-history ./helm/alert-history"
    echo ""
    echo -e "${BLUE}ℹ️  To disable integration:${NC}"
    echo "  ./scripts/disable-health-monitoring.sh"
    echo ""
    echo -e "${GREEN}🎉 Ready for deployment!${NC}"
else
    echo -e "${RED}❌ Error: Failed to enable integration${NC}"
    echo "Restoring backup..."
    cp "$BACKUP_FILE" "$MAIN_FILE"
    exit 1
fi
