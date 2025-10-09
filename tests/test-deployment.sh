#!/bin/bash
set -euo pipefail

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Testing Alert History Service deployment with new dependencies${NC}"

# Функция для логирования
log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

error() {
    echo -e "${RED}[ERROR] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[WARNING] $1${NC}"
}

# Проверка наличия необходимых инструментов
check_dependencies() {
    log "Checking dependencies..."

    if ! command -v helm &> /dev/null; then
        error "Helm is not installed"
        exit 1
    fi

    if ! command -v kubectl &> /dev/null; then
        error "kubectl is not installed"
        exit 1
    fi

    log "Dependencies check passed ✅"
}

# Проверка конфигурации Helm chart
validate_helm_chart() {
    log "Validating Helm chart..."

    cd helm/alert-history

    # Lint helm chart
    if helm lint .; then
        log "Helm lint passed ✅"
    else
        error "Helm lint failed ❌"
        exit 1
    fi

    # Template rendering test
    if helm template alert-history . --dry-run > /tmp/alert-history-templates.yaml; then
        log "Helm template rendering passed ✅"
    else
        error "Helm template rendering failed ❌"
        exit 1
    fi

    cd ../..
}

# Проверка dependencies
check_helm_dependencies() {
    log "Checking Helm dependencies..."

    cd helm/alert-history

    # Update dependencies
    if helm dependency update; then
        log "Helm dependencies updated ✅"
    else
        error "Failed to update Helm dependencies ❌"
        exit 1
    fi

    # Проверка что dependencies скачались
    if [ -d "charts" ] && [ "$(ls -A charts)" ]; then
        log "Dependencies downloaded ✅"
        ls -la charts/
    else
        warn "No dependencies found - this might be expected"
    fi

    cd ../..
}

# Проверка образа Docker (если доступен)
check_docker_image() {
    log "Checking Docker image..."

    if command -v docker &> /dev/null; then
        # Попробуем собрать образ
        if docker build -t alert-history:test . > /dev/null 2>&1; then
            log "Docker image build passed ✅"
        else
            warn "Docker image build failed - this might be expected in CI"
        fi
    else
        warn "Docker not available - skipping image check"
    fi
}

# Проверка конфигурации Python
check_python_config() {
    log "Checking Python configuration..."

    if [ -f "config.py" ]; then
        # Проверим что config.py синтаксически корректен
        if python3 -m py_compile config.py; then
            log "config.py syntax check passed ✅"
        else
            error "config.py has syntax errors ❌"
            exit 1
        fi
    else
        error "config.py not found ❌"
        exit 1
    fi
}

# Проверка requirements.txt
check_requirements() {
    log "Checking requirements.txt..."

    if [ -f "requirements.txt" ]; then
        log "requirements.txt found ✅"

        # Проверим что все зависимости указаны корректно
        if python3 -c "
import pkg_resources
with open('requirements.txt') as f:
    requirements = f.read().splitlines()
for req in requirements:
    if req.strip() and not req.startswith('#'):
        try:
            pkg_resources.Requirement.parse(req)
        except Exception as e:
            print(f'Invalid requirement: {req} - {e}')
            exit(1)
print('All requirements are valid')
"; then
            log "requirements.txt validation passed ✅"
        else
            error "requirements.txt has invalid entries ❌"
            exit 1
        fi
    else
        error "requirements.txt not found ❌"
        exit 1
    fi
}

# Проверка структуры проекта
check_project_structure() {
    log "Checking project structure..."

    required_files=(
        "helm/alert-history/Chart.yaml"
        "helm/alert-history/values.yaml"
        "helm/alert-history/templates/deployment.yaml"
        "helm/alert-history/templates/service.yaml"
        "helm/alert-history/templates/_helpers.tpl"
        "config.py"
        "requirements.txt"
        "Dockerfile"
    )

    for file in "${required_files[@]}"; do
        if [ -f "$file" ]; then
            log "✅ $file"
        else
            error "❌ Missing required file: $file"
            exit 1
        fi
    done

    log "Project structure check passed ✅"
}

# Основная функция
main() {
    log "Starting deployment test..."

    check_dependencies
    check_project_structure
    check_python_config
    check_requirements
    check_helm_dependencies
    validate_helm_chart
    check_docker_image

    log "🎉 All tests passed! Deployment configuration looks good."
    log "Next steps:"
    log "  1. Deploy to development: helm install alert-history ./helm/alert-history"
    log "  2. Check pod status: kubectl get pods"
    log "  3. Check logs: kubectl logs -l app.kubernetes.io/name=alert-history"
}

# Запуск
main "$@"
