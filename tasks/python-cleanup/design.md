# Python Code Cleanup - Design Document

## Стратегия очистки

### Общий подход: "Progressive Deprecation"

```
┌─────────────────────────────────────────────────────────────┐
│              Current State (Both Active)                    │
│  ┌─────────────┐              ┌─────────────┐              │
│  │   Python    │◄────────────►│   Go App    │              │
│  │  (FastAPI)  │   Share DB   │  (Gin/Fiber)│              │
│  │  Production │   & Redis    │  Production │              │
│  └─────────────┘              └─────────────┘              │
└─────────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────────┐
│            Phase 1: Analysis & Documentation                │
│  • Map Python → Go components                               │
│  • Identify gaps                                            │
│  • Create migration matrix                                  │
└─────────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────────┐
│          Phase 2: Code Reorganization                       │
│  ┌─────────────┐              ┌─────────────┐              │
│  │   legacy/   │              │   Go App    │              │
│  │  (Python)   │              │  PRIMARY ✅  │              │
│  │  Deprecated │              │             │              │
│  └─────────────┘              └─────────────┘              │
└─────────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────────┐
│           Phase 3: Gradual Transition                       │
│  Traffic: 90% Go, 10% Python (canary)                      │
│  Monitor: Errors, Performance, Compatibility               │
└─────────────────────────────────────────────────────────────┘
                         ↓
┌─────────────────────────────────────────────────────────────┐
│            Phase 4: Python Sunset                           │
│  ┌─────────────┐              ┌─────────────┐              │
│  │  archive/   │              │   Go App    │              │
│  │  (Reference)│              │   ONLY ✅    │              │
│  │  Read-only  │              │  Production │              │
│  └─────────────┘              └─────────────┘              │
└─────────────────────────────────────────────────────────────┘
```

## Матрица соответствия: Python → Go

### ✅ Полностью мигрировано

| Python Component | Go Component | Status | Notes |
|------------------|--------------|--------|-------|
| `config.py` | `internal/config/` | ✅ 100% | Viper config loader |
| `logging_config.py` | `pkg/logger/` | ✅ 100% | Slog structured logging |
| `core/interfaces.py` | `internal/core/interfaces.go` | ✅ 100% | Alert, Classification models |
| `database/sqlite_adapter.py` | `internal/database/sqlite.go` | ✅ 100% | SQLite support |
| `database/postgresql_adapter.py` | `internal/database/postgres.go` | ✅ 100% | PostgreSQL with pgx |
| `database/migration_manager.py` | `internal/infrastructure/migrations/` | ✅ 100% | Goose migrations |
| `services/redis_cache.py` | `internal/infrastructure/cache/` | ✅ 100% | go-redis v9 |
| `core/metrics.py` | `pkg/metrics/` | ✅ 100% | Prometheus metrics |
| `api/health_endpoints.py` | `cmd/server/handlers/health.go` | ✅ 100% | /healthz, /readyz |

### 🔄 Частично мигрировано

| Python Component | Go Component | Status | Missing in Go |
|------------------|--------------|--------|---------------|
| `services/alert_classifier.py` | `internal/infrastructure/llm/` | 🔄 80% | Retry logic advanced |
| `services/filter_engine.py` | `internal/core/filtering.go` | 🔄 95% | LLM-based filtering |
| `api/webhook_endpoints.py` | `cmd/server/handlers/webhook.go` | 🔄 70% | Complex routing |
| `services/alert_publisher.py` | TBD | 🔄 50% | Multi-target publishing |
| `services/target_discovery.py` | TBD | ⏸️ 0% | K8s secrets discovery |
| `api/dashboard_endpoints.py` | TBD | ⏸️ 0% | HTML5 dashboard |
| `api/enrichment_endpoints.py` | TBD | ⏸️ 30% | Mode switching API |

### ❌ Не мигрировано (нужно решение)

| Python Component | Действие | Причина |
|------------------|----------|---------|
| `api/proxy_endpoints.py` | 🔄 Migrate | Intelligent proxy core logic |
| `services/webhook_processor.py` | 🔄 Migrate | Complex webhook processing |
| `api/publishing_endpoints.py` | 🔄 Migrate | Publishing API |
| `api/classification_endpoints.py` | ⏸️ Evaluate | Может быть deprecated? |
| `services/graceful_shutdown.py` | ✅ Already in Go | Можно удалить |
| `services/health_checker.py` | ✅ Already in Go | Можно удалить |
| `core/app_state.py` | ❓ Evaluate | Stateful management - нужен ли? |
| `core/stateless_manager.py` | ✅ Go stateless | Можно удалить |
| `utils/*` | 🔄 Case-by-case | Некоторые нужны, некоторые нет |

## Стратегия по категориям

### Категория 1: УДАЛИТЬ (Полные дубликаты)

**Компоненты:**
```python
src/alert_history/
├── logging_config.py           # ✅ Go: pkg/logger/
├── core/metrics.py             # ✅ Go: pkg/metrics/
├── services/health_checker.py  # ✅ Go: handlers/health.go
├── services/graceful_shutdown.py # ✅ Go: cmd/server/main.go
├── core/stateless_manager.py   # ✅ Go: stateless by design
└── utils/stateless_decorators.py # ✅ Go: not needed
```

**Действие:**
1. Git move → `legacy/deprecated/` с README
2. Deprecation warning в imports
3. Убрать из активного CI/CD
4. Scheduled deletion: 3 месяца

### Категория 2: АРХИВИРОВАТЬ (Reference)

**Компоненты:**
```python
src/alert_history/
├── services/alert_classifier.py   # Complex LLM logic
├── services/filter_engine.py      # Advanced filtering algorithms
├── api/proxy_endpoints.py         # Intelligent proxy patterns
└── services/webhook_processor.py  # Webhook processing logic
```

**Действие:**
1. Git move → `legacy/reference/` с документацией
2. Оставить read-only
3. Ссылки в Go код (comments): "See Python reference: legacy/reference/..."
4. Периодический review (раз в квартал)

### Категория 3: ПОДДЕРЖИВАТЬ (Active Legacy)

**Компоненты:**
```python
src/alert_history/
├── main.py                      # Entry point (пока нужен)
├── api/legacy_adapter.py        # Legacy API compatibility
├── api/dashboard_endpoints.py   # HTML dashboard (пока не в Go)
└── api/publishing_endpoints.py  # Publishing API (миграция в процессе)
```

**Действие:**
1. Оставить в `src/alert_history/`
2. Минимальная поддержка (security fixes only)
3. Чёткий deprecation timeline
4. Migration guide для users

### Категория 4: МИГРИРОВАТЬ СРОЧНО

**Приоритет на миграцию:**

1. **publishing_endpoints.py** → Go (Part of TN-46 to TN-60)
   - Критично для production
   - Активно используется
   - Миграция: 2-3 недели

2. **proxy_endpoints.py** → Go (Part of Core)
   - Intelligent proxy - core feature
   - LLM integration critical
   - Миграция: 1-2 недели

3. **webhook_processor.py** → Go (Part of TN-41 to TN-45)
   - Webhook processing logic
   - Can reuse from Python
   - Миграция: 1 неделя

## Структура после cleanup

```
AlertHistory/
├── go-app/                    # 🎯 PRIMARY (Go)
│   ├── cmd/
│   ├── internal/
│   ├── pkg/
│   └── README.md             # "This is the primary codebase"
│
├── legacy/                    # 📦 LEGACY CODE
│   ├── reference/            # Reference implementations
│   │   ├── alert_classifier.py
│   │   ├── filter_engine.py
│   │   └── README.md         # "For reference only, see Go impl"
│   │
│   ├── deprecated/           # Scheduled for deletion
│   │   ├── metrics.py
│   │   ├── health_checker.py
│   │   └── DEPRECATION.md    # "Will be deleted on YYYY-MM-DD"
│   │
│   └── active/               # Still in use (temporary)
│       ├── main.py
│       ├── api/
│       │   ├── legacy_adapter.py
│       │   └── dashboard_endpoints.py
│       └── README.md         # "Active legacy, migration in progress"
│
├── src/                       # ❌ УДАЛИТЬ (после миграции)
│   └── alert_history/        # Currently active Python code
│       └── ...               # Will be moved to legacy/
│
├── MIGRATION.md              # 📖 Migration guide (Python → Go)
├── DEPRECATION.md            # 📅 Deprecation timeline
└── README.md                 # 🎯 "Go is primary, Python is legacy"
```

## Dependency Cleanup Strategy

### Current Dependencies (requirements.txt)

```python
# Web Framework (Can remove после полной миграции на Go)
fastapi==0.104.1           # ❌ REMOVE after Go API complete
uvicorn==0.24.0            # ❌ REMOVE after Go server only
starlette==0.27.0          # ❌ REMOVE (FastAPI dependency)

# Database (Keep minimal)
sqlalchemy==2.0.23         # ⚠️  EVALUATE (Go uses pgx/sqlite directly)
psycopg2-binary==2.9.9     # ⚠️  EVALUATE (Go doesn't need)
alembic==1.12.1            # ❌ REMOVE (Go uses goose)

# Redis (Evaluate)
redis==5.0.1               # ⚠️  EVALUATE (Go uses go-redis)

# LLM/AI (Keep if Python endpoints active)
openai==1.3.7              # ✅ KEEP (пока нужен для legacy endpoints)

# Data validation (Can remove)
pydantic==2.5.2            # ❌ REMOVE (Go uses structs + validator)

# Monitoring (Keep minimal)
prometheus-client==0.19.0  # ⚠️  EVALUATE (Go has native Prometheus)

# Utils (Review case-by-case)
python-dotenv==1.0.0       # ✅ KEEP (for local dev)
pyyaml==6.0.1              # ✅ KEEP (config parsing)
```

### Target Dependencies (Minimal)

```python
# Only for active legacy endpoints
fastapi==0.104.1       # If dashboard still in Python
uvicorn==0.24.0        # If serving Python
openai==1.3.7          # If LLM calls from Python
python-dotenv==1.0.0   # For local development
pyyaml==6.0.1          # For config parsing
```

**Reduction**: 30 deps → ~5 deps (83% reduction)

## Deployment Strategy

### Phase 1: Dual-Stack (Current, 2 weeks)

```yaml
# docker-compose.yml
services:
  alert-history-go:
    image: alert-history:go-latest
    ports:
      - "8080:8080"
    environment:
      - PRIMARY=true
      - TRAFFIC_WEIGHT=90

  alert-history-python:
    image: alert-history:python-latest
    ports:
      - "8081:8080"
    environment:
      - LEGACY=true
      - TRAFFIC_WEIGHT=10
      - DEPRECATION_MODE=true

  load-balancer:
    image: nginx:alpine
    depends_on:
      - alert-history-go
      - alert-history-python
    ports:
      - "80:80"
    # Route 90% to Go, 10% to Python
```

### Phase 2: Go Primary with Python Fallback (2-4 weeks)

```yaml
services:
  alert-history-go:
    image: alert-history:go-latest
    environment:
      - PRIMARY=true
      - TRAFFIC_WEIGHT=99
      - FALLBACK_URL=http://alert-history-python:8080

  alert-history-python:
    image: alert-history:python-latest
    environment:
      - LEGACY=true
      - TRAFFIC_WEIGHT=1
      - READ_ONLY_MODE=true  # Only serves legacy endpoints
```

### Phase 3: Go Only (After successful transition)

```yaml
services:
  alert-history:
    image: alert-history:go-latest
    # Python removed entirely
```

## Testing Strategy

### Compatibility Tests

```python
# tests/compatibility/test_python_go_parity.py

import pytest
import requests

PYTHON_URL = "http://localhost:8081"
GO_URL = "http://localhost:8080"

def test_webhook_endpoint_parity():
    """Ensure Go and Python respond identically"""
    payload = {
        "alerts": [{
            "labels": {"alertname": "TestAlert"},
            "status": "firing"
        }]
    }

    python_resp = requests.post(f"{PYTHON_URL}/webhook", json=payload)
    go_resp = requests.post(f"{GO_URL}/webhook", json=payload)

    assert python_resp.status_code == go_resp.status_code
    # Compare response structure (not exact match)
    assert set(python_resp.json().keys()) == set(go_resp.json().keys())

def test_history_endpoint_parity():
    """Ensure history queries return same data"""
    python_resp = requests.get(f"{PYTHON_URL}/history?limit=10")
    go_resp = requests.get(f"{GO_URL}/history?limit=10")

    assert python_resp.status_code == go_resp.status_code
    # Data might differ slightly but structure should match
    assert len(python_resp.json()["alerts"]) == len(go_resp.json()["alerts"])
```

### Performance Comparison

```python
# tests/performance/compare_python_go.py

import time
import statistics

def benchmark_endpoint(url, iterations=100):
    times = []
    for _ in range(iterations):
        start = time.time()
        requests.get(url)
        times.append(time.time() - start)

    return {
        "mean": statistics.mean(times),
        "median": statistics.median(times),
        "p95": statistics.quantiles(times, n=20)[18],  # 95th percentile
        "p99": statistics.quantiles(times, n=100)[98]  # 99th percentile
    }

python_stats = benchmark_endpoint(f"{PYTHON_URL}/health")
go_stats = benchmark_endpoint(f"{GO_URL}/healthz")

print(f"Python p95: {python_stats['p95']*1000:.2f}ms")
print(f"Go p95: {go_stats['p95']*1000:.2f}ms")
print(f"Improvement: {(1 - go_stats['p95']/python_stats['p95'])*100:.1f}%")

# Expected: Go should be 2-5x faster
assert go_stats['p95'] < python_stats['p95'] * 0.5, "Go should be at least 2x faster"
```

## Rollback Plan

### If Go version has critical issues:

1. **Immediate** (< 5 minutes):
   ```bash
   # Switch load balancer back to Python
   kubectl patch service alert-history --patch '{"spec":{"selector":{"app":"alert-history-python"}}}'
   ```

2. **Short-term** (< 1 hour):
   ```bash
   # Revert deployment
   helm rollback alert-history
   ```

3. **Investigation** (parallel):
   - Analyze Go version logs/metrics
   - Identify root cause
   - Fix in development
   - Re-test before next attempt

### Rollback Criteria:

- Error rate > 1% (compared to Python baseline <0.1%)
- P95 latency > 2x Python baseline
- Data loss or corruption
- Critical feature regression
- Customer complaints

## Documentation Updates

### README.md (Root)

```markdown
# Alert History Service

🚀 **Go version is now PRIMARY**. Python version is in maintenance mode.

## Quick Start

### Recommended: Go Version
\`\`\`bash
cd go-app
make docker-build && make docker-run
\`\`\`

### Legacy: Python Version (Deprecated)
See [DEPRECATION.md](DEPRECATION.md) for timeline.
\`\`\`bash
# For legacy endpoints only
python -m uvicorn src.alert_history.main:app
\`\`\`

## Migration Guide
See [MIGRATION.md](MIGRATION.md) for Python → Go migration instructions.
```

### MIGRATION.md (New)

```markdown
# Migration Guide: Python → Go

This guide helps users migrate from Python to Go version of Alert History Service.

## Timeline
- **2025-01-09**: Go version becomes primary
- **2025-02-01**: Python deprecation announced
- **2025-03-01**: Python receives security fixes only
- **2025-04-01**: Python version sunset (removed)

## API Changes
### Endpoints that changed:
- `/health` → `/healthz` (Go standard)
- `/metrics` → `/metrics` (compatible)
- `/webhook` → `/webhook` (compatible)

### Endpoints removed:
- `/legacy/...` endpoints are removed
- Use new Go endpoints instead

## Breaking Changes
(List specific breaking changes)
```

---

**Архитектор**: DevOps Team
**Дата создания**: 2025-01-09
**Версия**: 1.0
**Статус**: Ready for Implementation
