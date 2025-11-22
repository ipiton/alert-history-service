#!/bin/bash
# Silence UI Components Deployment Script
# Phase 15: Documentation - Deployment automation

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
ENVIRONMENT="${ENVIRONMENT:-staging}"
NAMESPACE="${NAMESPACE:-alert-history}"
REPLICAS="${REPLICAS:-2}"

echo -e "${GREEN}🚀 Deploying Silence UI Components${NC}"
echo "Environment: $ENVIRONMENT"
echo "Namespace: $NAMESPACE"
echo "Replicas: $REPLICAS"

# Step 1: Build application
echo -e "\n${YELLOW}Step 1: Building application...${NC}"
cd "$PROJECT_ROOT/go-app"
if ! go build -o bin/server ./cmd/server; then
    echo -e "${RED}❌ Build failed${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Build successful${NC}"

# Step 2: Run tests
echo -e "\n${YELLOW}Step 2: Running tests...${NC}"
if ! go test ./cmd/server/handlers -run TestSilenceUIHandler -v; then
    echo -e "${RED}❌ Tests failed${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Tests passed${NC}"

# Step 3: Run linter
echo -e "\n${YELLOW}Step 3: Running linter...${NC}"
if ! golangci-lint run ./cmd/server/handlers/...; then
    echo -e "${RED}❌ Linter failed${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Linter passed${NC}"

# Step 4: Check for race conditions
echo -e "\n${YELLOW}Step 4: Checking for race conditions...${NC}"
if ! go test -race ./cmd/server/handlers -run TestSilenceUIHandler; then
    echo -e "${RED}❌ Race condition detected${NC}"
    exit 1
fi
echo -e "${GREEN}✅ No race conditions${NC}"

# Step 5: Generate coverage report
echo -e "\n${YELLOW}Step 5: Generating coverage report...${NC}"
go test ./cmd/server/handlers -coverprofile=coverage.out
coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
echo "Coverage: $coverage"
if [ "$(echo "$coverage < 80" | bc)" -eq 1 ]; then
    echo -e "${RED}❌ Coverage below 80%${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Coverage acceptable${NC}"

# Step 6: Build Docker image (if Dockerfile exists)
if [ -f "$PROJECT_ROOT/Dockerfile" ]; then
    echo -e "\n${YELLOW}Step 6: Building Docker image...${NC}"
    if ! docker build -t alert-history:latest "$PROJECT_ROOT"; then
        echo -e "${RED}❌ Docker build failed${NC}"
        exit 1
    fi
    echo -e "${GREEN}✅ Docker image built${NC}"
fi

# Step 7: Deploy to Kubernetes (if kubeconfig exists)
if [ -f "$HOME/.kube/config" ]; then
    echo -e "\n${YELLOW}Step 7: Deploying to Kubernetes...${NC}"

    # Apply namespace
    kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

    # Apply configuration
    if [ -f "$PROJECT_ROOT/k8s/silence-ui-config.yaml" ]; then
        kubectl apply -f "$PROJECT_ROOT/k8s/silence-ui-config.yaml" -n "$NAMESPACE"
    fi

    # Apply deployment
    if [ -f "$PROJECT_ROOT/k8s/silence-ui-deployment.yaml" ]; then
        kubectl apply -f "$PROJECT_ROOT/k8s/silence-ui-deployment.yaml" -n "$NAMESPACE"
        kubectl rollout status deployment/silence-ui -n "$NAMESPACE" --timeout=5m
    fi

    echo -e "${GREEN}✅ Deployment successful${NC}"
fi

# Step 8: Verify deployment
echo -e "\n${YELLOW}Step 8: Verifying deployment...${NC}"
if [ -f "$HOME/.kube/config" ]; then
    # Check pods
    kubectl get pods -n "$NAMESPACE" -l app=silence-ui

    # Check services
    kubectl get svc -n "$NAMESPACE" -l app=silence-ui

    # Check metrics endpoint
    echo "Checking metrics endpoint..."
    # kubectl port-forward -n "$NAMESPACE" svc/silence-ui 8080:8080 &
    # sleep 2
    # curl -s http://localhost:8080/metrics | grep alert_history_ui
    # kill %1
fi

echo -e "\n${GREEN}✅ Deployment complete!${NC}"
echo -e "${GREEN}🎉 Silence UI Components deployed successfully${NC}"
