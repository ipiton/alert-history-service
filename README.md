# Alert History Service - Intelligent Alert Proxy

![GitHub](https://img.shields.io/badge/GitHub-ipiton%2Falert--history--service-blue?logo=github)
![Docker](https://img.shields.io/badge/Docker-Supported-blue?logo=docker)
![Kubernetes](https://img.shields.io/badge/Kubernetes-Helm%20Chart-blue?logo=kubernetes)
![LLM](https://img.shields.io/badge/LLM-Intelligent%20Classification-green?logo=openai)
![Prometheus](https://img.shields.io/badge/Prometheus-Metrics-orange?logo=prometheus)

🚀 **Production-ready Intelligent Alert Proxy** для Alertmanager с LLM классификацией, автоматической публикацией в внешние системы, horizontal scaling и переключаемыми режимами обработки.

**GitHub Repository:** [https://github.com/ipiton/alert-history-service.git](https://github.com/ipiton/alert-history-service.git)

---

## ✨ Основные возможности

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
open http://localhost:8080/dashboard/modern
```

---

## 🎛️ API Endpoints

### Core Endpoints
- **POST /webhook** — legacy Alertmanager webhook
- **POST /webhook/proxy** — intelligent proxy с classification & publishing
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

## 📋 Roadmap

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
