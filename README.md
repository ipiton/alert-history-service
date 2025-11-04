# Alert History Service - Intelligent Alert Proxy

![GitHub](https://img.shields.io/badge/GitHub-ipiton%2Falert--history--service-blue?logo=github)
![Go CI](https://github.com/ipiton/alert-history-service/actions/workflows/go.yml/badge.svg)
![Docker](https://img.shields.io/badge/Docker-Supported-blue?logo=docker)
![Kubernetes](https://img.shields.io/badge/Kubernetes-Helm%20Chart-blue?logo=kubernetes)
![LLM](https://img.shields.io/badge/LLM-Intelligent%20Classification-green?logo=openai)
![Prometheus](https://img.shields.io/badge/Prometheus-Metrics-orange?logo=prometheus)

🚀 **Production-ready Intelligent Alert Proxy** для Alertmanager с LLM классификацией, автоматической публикацией в внешние системы, horizontal scaling и переключаемыми режимами обработки.

**GitHub Repository:** [https://github.com/ipiton/alert-history-service.git](https://github.com/ipiton/alert-history-service.git)

---

## 🚨 IMPORTANT: Go Version is Now PRIMARY

> **📢 Announcement** (2025-01-09): **Go version is now the PRIMARY codebase**
> **🔴 Python version is DEPRECATED** and will be sunset on **April 1, 2025**

### Migration Required

- ✅ **Use Go version** for all new deployments
- ⚠️ **Migrate from Python** before April 1, 2025
- 📖 **Read the migration guide**: [MIGRATION.md](MIGRATION.md)
- 📅 **Deprecation timeline**: [DEPRECATION.md](DEPRECATION.md)

### Why Migrate?

| Feature | Python | Go | Improvement |
|---------|--------|----|----|
| Performance | Baseline | **2-5x faster** | 🚀 |
| Memory | 300 MB | **50 MB** | 83% ⬇️ |
| Docker Image | 500 MB | **20 MB** | 96% ⬇️ |
| Startup Time | 5s | **<1s** | 80% ⬇️ |
| Type Safety | Runtime | **Compile-time** | ✅ |
| Concurrency | asyncio | **Goroutines** | ✅ |

---

## 🚀 Quick Start (Go Version - Recommended)

### Быстрый старт с Go

```bash
# Перейти в Go директорию
cd go-app

# Установить зависимости и собрать
make deps && make build

# Запустить приложение
make run

# Health check
curl http://localhost:8080/healthz
```

### Docker (рекомендуемый способ)

```bash
cd go-app

# Собрать образ
make docker-build

# Запустить контейнер
make docker-run

# Проверить health
curl http://localhost:8080/healthz
```

### Особенности Go версии
- ✅ **Multi-stage Docker build** (< 10MB образ)
- ✅ **Structured logging** в JSON формате
- ✅ **Graceful shutdown** с таймаутами
- ✅ **Health checks** для Kubernetes
- ✅ **Static binary** без зависимостей
- ✅ **Production-ready** containerization

📖 **[Подробная документация Go версии](go-app/README.md)**

---

## ✨ Основные возможности

### 🎯 Alert Grouping System (NEW - 2025-11-03) ⭐
**Status**: 80% Complete (4/5 tasks) | **Quality**: 171% average

#### TN-124: Group Wait/Interval Timers ✅ (2025-11-03)
- **Redis-persisted timer management** для group_wait, group_interval, repeat_interval
- **High Availability**: RestoreTimers recovery после рестарта
- **2.4x faster** than target (0.42ms StartTimer!)
- **7 Prometheus metrics** для мониторинга таймеров
- **82.7% test coverage** (177 tests, 7 benchmarks)
- **Graceful degradation**: Redis → in-memory fallback
- **152.6% quality** achievement (Grade A+)

#### TN-123: Alert Group Manager ✅ (2025-11-03)
- **Высокопроизводительное** управление группами алертов
- **1300x faster** than target (0.38µs operations!)
- **Thread-safe** concurrent access
- **Advanced filtering** (state, labels, receiver, pagination)
- **4 Prometheus metrics** для мониторинга
- **95%+ test coverage** (27 tests, 8 benchmarks)
- **183.6% quality** achievement (Grade A+)

#### TN-122: Group Key Generator ✅ (2025-11-03)
- **FNV-1a hash-based** grouping (404x faster than target!)
- **Deterministic** key generation
- **200% quality** achievement (Grade A+)

#### TN-121: Grouping Config Parser ✅ (2025-11-03)
- **Alertmanager-compatible** YAML routing configuration
- **150% quality** achievement (Grade A+)

**Next**: TN-125 Group Storage (Redis Backend)

### 🧠 Intelligent Alert Processing
- **LLM-powered alert classification** с GPT-4 через LLM-proxy
- **Переключаемые режимы**: `transparent` (без LLM) и `enriched` (с LLM)
- **Dynamic Target Discovery** из Kubernetes Secrets
- **Advanced Alert Filtering** по severity, confidence, namespace, labels

### 🎯 Publishing & Integration
- **Intelligent Alert Proxy** для автоматической публикации
- **Multi-target publishing**: Rootly, PagerDuty, Slack, custom webhooks
- **Retry logic** с exponential backoff и circuit breaker
- **Metrics-only mode** при отсутствии targets

### 🏗️ Architecture & Scaling
- **12-Factor App compliance** с конфигурацией через ENV
- **Horizontal autoscaling** (2-10 replicas) с Kubernetes HPA
- **Stateless design** с координацией через Redis/PostgreSQL
- **Graceful shutdown** и health probes

### 📊 Monitoring & Observability
- **Real-time HTML5 dashboards** с CSS Grid/Flexbox
- **Prometheus metrics** с recording rules для aggregation
- **Grafana dashboards** для enrichment mode monitoring
- **Comprehensive logging** в JSON формате

### 🗄️ Data & Storage
- **PostgreSQL** для production persistence
- **Redis** для distributed caching и coordination
- **SQLite** для development и testing
- **Database migration CLI** с version-based scripts

---

## 🚀 Quick Start

### Development Environment

```bash
# Clone repository
git clone https://github.com/ipiton/alert-history-service.git
cd alert-history-service

# Setup virtual environment
python3 -m venv .venv
source .venv/bin/activate

# Install dependencies
pip install -r requirements.txt

# Start development server
uvicorn src.alert_history.main:app --host 0.0.0.0 --port 8080 --reload
```

### Health & Status Checks

```bash
# Health check
curl http://localhost:8080/healthz

# Readiness check
curl http://localhost:8080/readyz

# Metrics
curl http://localhost:8080/metrics

# Dashboard
open http://localhost:8080/dashboard

# API Documentation
open http://localhost:8080/docs
```

---

## 🎛️ API Endpoints

### Core Endpoints
- **POST /webhook** — universal webhook (auto-switches between legacy and intelligent modes)
- **POST /webhook/proxy** — explicit intelligent proxy с classification & publishing
- **GET /history** — alert history с advanced filtering
- **GET /report** — analytics (top alerts, flapping, summary)

### Publishing & Targets
- **GET /publishing/targets** — discovered publishing targets
- **POST /publishing/targets/refresh** — refresh target discovery
- **GET /publishing/mode** — current publishing mode
- **GET /publishing/stats** — publishing statistics

### Classification & LLM
- **GET /classification/stats** — classification statistics
- **POST /classification/classify** — manual alert classification
- **GET /classification/models** — available LLM models

### Enrichment Mode
- **GET /enrichment/mode** — current enrichment mode
- **POST /enrichment/mode** — switch enrichment mode

### Dashboard & API
- **GET /dashboard/modern** — HTML5 dashboard
- **GET /api/dashboard/overview** — dashboard overview data
- **GET /api/dashboard/charts** — time series chart data
- **GET /api/dashboard/health** — system health data

---

## 🏗️ Production Deployment

### 1. Kubernetes with Helm

```bash
# Install from Git repository
helm install alert-history \
  oci://ghcr.io/ipiton/alert-history-service/helm \
  --version latest

# Or install from local chart
helm install alert-history ./helm/alert-history \
  --set image.repository=ghcr.io/ipiton/alert-history-service \
  --set image.tag=latest \
  --set postgresql.enabled=true \
  --set redis.enabled=true
```

### 2. Environment Configuration

```bash
# Core configuration
ENVIRONMENT=production
LOG_LEVEL=info
ENRICHMENT_MODE=enriched  # or transparent

# Database
DATABASE_URL=postgresql://user:pass@host:5432/alerthistory
REDIS_URL=redis://redis:6379/0

# LLM Integration
LLM_PROXY_URL=http://llm-proxy:8080
LLM_MODEL=gpt-4

# Publishing
PUBLISHING_ENABLED=true
TARGET_DISCOVERY_ENABLED=true
TARGET_DISCOVERY_NAMESPACE=alert-targets
```

### 3. Kubernetes Secrets

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: rootly-config
  namespace: alert-targets
  labels:
    alert-history.io/target: "true"
    alert-history.io/format: "rootly"
type: Opaque
data:
  url: <base64-encoded-rootly-url>
  api_key: <base64-encoded-api-key>
---
apiVersion: v1
kind: Secret
metadata:
  name: slack-webhook
  namespace: alert-targets
  labels:
    alert-history.io/target: "true"
    alert-history.io/format: "slack"
type: Opaque
data:
  webhook_url: <base64-encoded-slack-webhook>
```

---

## 📊 Monitoring & Dashboards

### Grafana Dashboards

1. **Import dashboard**: `alert_history_grafana_dashboard_v3_enrichment.json`
2. **Configure Prometheus**: recording rules в `src/alert_history/api/metrics.py`
3. **Key metrics**:
   - Enrichment mode status и switches
   - Alert processing rates по режимам
   - Classification success rate
   - Publishing success rate по targets

### HTML5 Dashboard

- **URL**: `http://your-service/dashboard/modern`
- **Features**: Overview, Charts, Recent Alerts, Recommendations, Publishing
- **Real-time updates**: Auto-refresh с API polling

---

## 🔧 Configuration Reference

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ENVIRONMENT` | `development` | Environment name |
| `LOG_LEVEL` | `info` | Logging level |
| `ENRICHMENT_MODE` | `enriched` | Default enrichment mode |
| `DATABASE_URL` | `sqlite:///alerts.db` | Database connection |
| `REDIS_URL` | `redis://localhost:6379/0` | Redis connection |
| `LLM_PROXY_URL` | `""` | LLM proxy service URL |
| `PUBLISHING_ENABLED` | `true` | Enable alert publishing |
| `TARGET_DISCOVERY_ENABLED` | `true` | Enable target discovery |

### Helm Chart Values

```yaml
# Image configuration
image:
  repository: ghcr.io/ipiton/alert-history-service
  tag: latest
  pullPolicy: IfNotPresent

# Scaling
replicaCount: 2
autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70

# Dependencies
postgresql:
  enabled: true
redis:
  enabled: true
```

---

## 🧪 Testing

### Unit Tests
```bash
# Run all tests (pytest)
pytest -q

# Or verbose with coverage
pytest -v --cov=src/alert_history --cov-report=term-missing
```

### Integration Tests
```bash
# Run specific test suites
python tests/test_t1_2_database_migration.py
python tests/test_t1_3_redis_integration.py
python tests/test_t6_dashboard.py
```

### Load Testing
```bash
# Comprehensive test suite
python run_all_tests.py
```

---

## 🔍 Troubleshooting

### Common Issues

1. **Enrichment mode не переключается**
   - Проверьте Redis connectivity: `redis-cli ping`
   - Посмотрите logs: `kubectl logs deployment/alert-history`

2. **Publishing не работает**
   - Проверьте target discovery: `GET /publishing/targets`
   - Verify Kubernetes RBAC permissions
   - Check secret labels: `alert-history.io/target: "true"`

3. **LLM classification fails**
   - Verify LLM proxy connectivity
   - Check API keys в secrets
   - Switch to `transparent` mode temporarily

### Debug Commands

```bash
# Check service health
kubectl get pods -l app=alert-history
kubectl logs deployment/alert-history --tail=100

# Check target discovery
kubectl get secrets -l alert-history.io/target=true

# Test enrichment mode API
curl -X GET http://your-service/enrichment/mode
curl -X POST http://your-service/enrichment/mode -d '{"mode":"transparent"}'
```

---

## 🤝 Contributing

1. Fork the repository
2. Create feature branch: `git checkout -b feature/amazing-feature`
3. Commit changes: `git commit -m 'Add amazing feature'`
4. Push branch: `git push origin feature/amazing-feature`
5. Open Pull Request

### Development Setup
```bash
# Install development dependencies
pip install -r requirements-dev.txt

# Setup pre-commit hooks
pre-commit install

# Run code quality checks
black src/
flake8 src/
mypy src/
```

---

## 🔄 Go Migration Progress

### ФАЗА 1: Infrastructure Foundation ✅ **87.5% Complete**

| Task | Status | Description |
|------|--------|-------------|
| TN-01 | ✅ | Go module initialization |
| TN-02 | ✅ | Directory structure setup |
| TN-03 | ✅ | Makefile with development tools |
| TN-04 | ✅ | golangci-lint configuration |
| TN-05 | ✅ | GitHub Actions CI/CD pipeline |
| TN-06 | ✅ | HTTP server with /healthz endpoint |
| TN-07 | ✅ | Multi-stage Dockerfile (< 10MB) |
| TN-08 | 🔄 | **README documentation** (Current) |

### ФАЗА 2: Data Layer (Documented)

- Database connections (PostgreSQL, Redis, SQLite)
- Migration system
- ORM/Driver evaluation (pgx vs GORM)
- Connection pooling
- Health checks for databases

### ФАЗА 3: Core Services

- Configuration management (Viper)
- Structured logging (slog)
- HTTP framework evaluation (Fiber vs Gin)
- Middleware stack (CORS, logging, metrics)
- Error handling patterns

### ФАЗА 4: Business Logic

- Alert processing pipeline
- LLM integration (HTTP client)
- Publishing system (Rootly, PagerDuty, Slack)
- Target discovery (Kubernetes)
- Alert filtering engine

### Особенности Go версии

#### Преимущества миграции:
- 🚀 **Performance**: 2-5x faster than Python
- 📦 **Deployment**: Single static binary
- 🔒 **Security**: Minimal attack surface
- 🎯 **Resource usage**: Lower memory footprint
- ⚡ **Startup time**: Near-instant cold starts
- 🏗️ **Maintainability**: Strong typing, better tooling

#### Архитектура:
- **Hexagonal Architecture** (Ports & Adapters)
- **Dependency Injection** (Google Wire)
- **Clean Architecture** principles
- **12-Factor App** compliance

📁 **[Детальная документация по задачам](tasks/go-migration-analysis/)**

---

---

## 🔴 Python Version (DEPRECATED)

> **⚠️ WARNING**: Python version is deprecated and will be sunset on **April 1, 2025**

### Deprecation Status

| Phase | Date | Status |
|-------|------|--------|
| Deprecation Announced | 2025-02-01 | 📢 Upcoming |
| Security Fixes Only | 2025-03-01 | ⏳ 51 days |
| **Python Sunset** | 2025-04-01 | 🔴 **82 days** |

### For Existing Python Users

**You MUST migrate to Go before April 1, 2025**

1. 📖 Read [MIGRATION.md](MIGRATION.md) - Complete migration guide
2. 📅 Review [DEPRECATION.md](DEPRECATION.md) - Timeline and support policy
3. 🧪 Test Go version in staging environment
4. 🚀 Plan your migration (recommended: 1-2 weeks)
5. 📧 Get help: #alert-history-migration on Slack

### Python Quick Start (For Legacy Deployments Only)

> **Not recommended** for new deployments. Use Go version instead.

```bash
# Install dependencies
pip install -r requirements.txt

# Run locally
uvicorn src.alert_history.main:app --reload

# Health check
curl http://localhost:8000/health
```

**Docker** (Python):
```bash
docker build -t alert-history:python .
docker run -p 8000:8000 alert-history:python
```

**Important**: Python version will stop receiving updates after March 1, 2025.

---

## 📋 Roadmap

- [x] **Go Migration** - Core features complete ✅
- [ ] **Publishing System** (TN-46 to TN-60) - In progress
- [ ] **Alertmanager++** (TN-121 to TN-180) - Planned
- [ ] **ML локальная классификация** (Phase 9)
- [ ] **Advanced analytics** с predictive capabilities
- [ ] **Multi-cluster coordination** для enterprise
- [ ] **Custom LLM model fine-tuning**
- [ ] **GraphQL API** для complex queries

---

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details.

---

## 👨‍💻 Author

**Vitaly Semenov** - [@VitalySemenov](https://github.com/VitalySemenov)

**Organization**: [ipiton](https://github.com/ipiton)

---

![MIT License](https://img.shields.io/badge/license-MIT-green)
![Production Ready](https://img.shields.io/badge/production-ready-brightgreen)
![Kubernetes](https://img.shields.io/badge/kubernetes-native-blue)
![12Factor](https://img.shields.io/badge/12--factor-compliant-orange)
