# Python Dependency Analysis Report

**Date**: 2025-01-09
**Analyzed**: requirements.txt + requirements-dev.txt
**Goal**: Identify which dependencies can be removed after Go migration

---

## Production Dependencies (requirements.txt)

### ✅ KEEP - Essential (Active Python endpoints)

| Dependency | Version | Reason | Used By |
|------------|---------|--------|---------|
| PyYAML | >=6.0 | Config parsing | Config loading for active endpoints |
| fastapi | >=0.104.0 | Web framework | main.py, dashboard, publishing, legacy APIs |
| uvicorn[standard] | >=0.24.0 | ASGI server | Running Python service |
| jinja2 | >=3.1.0 | Templates | Dashboard HTML rendering |
| python-dotenv | >=1.0.0 | Env vars | Local development |

**Total**: 5 dependencies ✅

---

### ❌ REMOVE - Replaced by Go

| Dependency | Version | Reason | Go Equivalent |
|------------|---------|--------|---------------|
| prometheus_client | >=0.17.0 | Metrics | pkg/metrics/ (native Prometheus) |
| aiohttp | >=3.9.0 | HTTP client | net/http |
| asyncpg | >=0.29.0 | PostgreSQL | github.com/jackc/pgx |
| aioredis | >=2.0.0 | Redis | github.com/redis/go-redis/v9 |
| pydantic | >=2.5.0 | Data validation | Go structs + validator/v10 |
| asyncio-throttle | >=1.0.0 | Rate limiting | Can implement in Go if needed |
| structlog | >=23.2.0 | Logging | pkg/logger/ (slog) |

**Total**: 7 dependencies ❌

---

### ⚠️ EVALUATE - Depends on migration progress

| Dependency | Version | Status | Decision |
|------------|---------|--------|----------|
| kubernetes | >=28.1.0 | Used by target_discovery.py | ❌ REMOVE after TN-46 (K8s discovery in Go) |

**Total**: 1 dependency ⚠️

---

## Development Dependencies (requirements-dev.txt)

### Analysis by Category

#### Code Quality Tools (Can Keep Minimal)

| Tool | Current Version | Keep? | Reason |
|------|----------------|-------|--------|
| black | >=23.12.1 | ✅ YES | Python code formatting |
| isort | >=5.13.2 | ✅ YES | Import sorting |
| flake8 | >=6.1.0 | ✅ YES | Style checking |
| flake8-* | Multiple | ❌ NO | Excessive for legacy code |
| mypy | >=1.8.0 | ⚠️ MAYBE | Type checking valuable |
| pylint | >=3.0.3 | ❌ NO | Too heavy for legacy |

**Keep**: 3-4 tools (black, isort, flake8, maybe mypy)

---

#### Security Tools

| Tool | Current Version | Keep? | Reason |
|------|----------------|-------|--------|
| bandit | >=1.7.5 | ✅ YES | Security scanner (важно!) |
| safety | >=2.3.5 | ✅ YES | Vulnerability checking |
| semgrep | >=1.50.0 | ❌ NO | Overkill for legacy |

**Keep**: 2 tools (bandit, safety) - critical for security

---

#### Testing Framework

| Tool | Current Version | Keep? | Reason |
|------|----------------|-------|--------|
| pytest | >=7.4.3 | ✅ YES | Core testing framework |
| pytest-asyncio | >=0.21.1 | ✅ YES | Async testing |
| pytest-cov | >=4.1.0 | ✅ YES | Coverage reporting |
| pytest-mock | >=3.12.0 | ⚠️ MAYBE | Mocking utilities |
| pytest-xdist | >=3.5.0 | ❌ NO | Parallel testing (overkill) |
| pytest-benchmark | >=4.0.0 | ❌ NO | Use Go benchmarks |
| pytest-timeout | >=2.2.0 | ❌ NO | Not critical |

**Keep**: 3-4 tools (pytest, pytest-asyncio, pytest-cov, pytest-mock)

---

#### Test Utilities

| Tool | Current Version | Keep? | Reason |
|------|----------------|-------|--------|
| factory-boy | >=3.3.0 | ❌ NO | Test data generation (not needed) |
| faker | >=21.0.0 | ❌ NO | Fake data (not needed) |
| responses | >=0.24.1 | ❌ NO | HTTP mocking (Go tests handle this) |
| aioresponses | >=0.7.6 | ❌ NO | Async HTTP mocking |

**Keep**: 0 tools

---

#### Development Tools

| Tool | Current Version | Keep? | Reason |
|------|----------------|-------|--------|
| pre-commit | >=3.6.0 | ✅ YES | Git hooks (useful) |
| commitizen | >=3.13.0 | ❌ NO | Not essential |
| tox | >=4.11.4 | ❌ NO | Multi-version testing (unnecessary) |
| coverage | >=7.3.4 | ⚠️ MAYBE | If using pytest-cov |
| watchdog | >=3.0.0 | ❌ NO | File monitoring (unnecessary) |

**Keep**: 1 tool (pre-commit)

---

#### Documentation

| Tool | Current Version | Keep? | Reason |
|------|----------------|-------|--------|
| sphinx | >=7.2.6 | ❌ NO | Go docs are primary |
| sphinx-* | Multiple | ❌ NO | Not needed |
| myst-parser | >=2.0.0 | ❌ NO | Markdown support (unnecessary) |

**Keep**: 0 tools (legacy docs are frozen)

---

#### Debugging & Profiling

| Tool | Current Version | Keep? | Reason |
|------|----------------|-------|--------|
| ipdb | >=0.13.13 | ⚠️ MAYBE | Debugging useful |
| line-profiler | >=4.1.1 | ❌ NO | Use Go profiling |
| memory-profiler | >=0.61.0 | ❌ NO | Use Go profiling |
| py-spy | >=0.3.14 | ❌ NO | Use pprof |

**Keep**: 1 tool (ipdb) for debugging

---

#### Database Tools

| Tool | Current Version | Keep? | Reason |
|------|----------------|-------|--------|
| alembic | >=1.13.1 | ❌ NO | Go uses goose |
| sqlalchemy-stubs | >=0.4 | ❌ NO | Not using SQLAlchemy |

**Keep**: 0 tools

---

#### Kubernetes Tools

| Tool | Current Version | Keep? | Reason |
|------|----------------|-------|--------|
| kubernetes | >=28.1.0 | ❌ NO | After TN-46 complete |
| kopf | >=1.37.1 | ❌ NO | Not using operators |

**Keep**: 0 tools (after K8s migration)

---

#### API Tools

| Tool | Current Version | Keep? | Reason |
|------|----------------|-------|--------|
| httpx | >=0.25.2 | ⚠️ MAYBE | HTTP testing |
| openapi-spec-validator | >=0.7.1 | ❌ NO | Go OpenAPI tools |

**Keep**: 1 tool (httpx) if testing Python APIs

---

#### Jupyter Notebooks

| Tool | Current Version | Keep? | Reason |
|------|----------------|-------|--------|
| jupyter | >=1.0.0 | ❌ NO | Not used in production |
| notebook | >=7.0.6 | ❌ NO | Development tool |
| ipykernel | >=6.27.1 | ❌ NO | Not needed |

**Keep**: 0 tools

---

#### Performance Monitoring

| Tool | Current Version | Keep? | Reason |
|------|----------------|-------|--------|
| psutil | >=5.9.6 | ❌ NO | Go runtime metrics |
| py-cpuinfo | >=9.0.0 | ❌ NO | Not needed |

**Keep**: 0 tools

---

## Summary

### Production Dependencies

| Category | Current | Keep | Remove | Evaluate |
|----------|---------|------|--------|----------|
| Production | 13 | 5 (38%) | 7 (54%) | 1 (8%) |

### Development Dependencies

| Category | Current | Keep | Remove |
|----------|---------|------|--------|
| Code Quality | 10 | 4 | 6 |
| Security | 3 | 2 | 1 |
| Testing | 7 | 4 | 3 |
| Test Utils | 4 | 0 | 4 |
| Dev Tools | 5 | 1 | 4 |
| Documentation | 4 | 0 | 4 |
| Debug/Profile | 4 | 1 | 3 |
| Database | 2 | 0 | 2 |
| Kubernetes | 2 | 0 | 2 |
| API | 2 | 1 | 1 |
| Jupyter | 3 | 0 | 3 |
| Performance | 2 | 0 | 2 |
| **TOTAL DEV** | **48** | **13 (27%)** | **35 (73%)** |

---

## Proposed Minimal Dependencies

### requirements-minimal.txt (5 deps)

```python
# Only for active legacy endpoints
PyYAML>=6.0                    # Config parsing
fastapi>=0.104.0               # Web framework (dashboard, publishing)
uvicorn[standard]>=0.24.0      # ASGI server
jinja2>=3.1.0                  # Templates (dashboard)
python-dotenv>=1.0.0           # Environment variables
```

### requirements-dev-minimal.txt (13 deps)

```python
# Code Quality (4)
black>=23.12.1
isort>=5.13.2
flake8>=6.1.0
mypy>=1.8.0

# Security (2)
bandit>=1.7.5
safety>=2.3.5

# Testing (4)
pytest>=7.4.3
pytest-asyncio>=0.21.1
pytest-cov>=4.1.0
pytest-mock>=3.12.0

# Dev Tools (2)
pre-commit>=3.6.0
ipdb>=0.13.13

# API Testing (1)
httpx>=0.25.2
```

---

## Dependency Reduction

**Production**: 13 → 5 (62% reduction)
**Development**: 48 → 13 (73% reduction)
**Total**: 61 → 18 (70% reduction)

---

## Security Scan Recommendations

### Run before cleanup:
```bash
# Check for vulnerabilities
pip-audit --requirement requirements.txt
safety check --file requirements.txt

# After cleanup
pip-audit --requirement requirements-minimal.txt
safety check --file requirements-minimal.txt
```

### Expected Results:
- Current requirements.txt: May have vulnerabilities in unused deps
- Minimal requirements.txt: Fewer deps = smaller attack surface
- Focus security updates on only 5 production deps

---

## Migration Timeline

### Week 1: Preparation
- [ ] Create requirements-minimal.txt
- [ ] Create requirements-dev-minimal.txt
- [ ] Test that active Python endpoints work
- [ ] Update Dockerfile to use minimal requirements

### Week 2: Transition
- [ ] Switch CI/CD to use minimal requirements
- [ ] Update documentation
- [ ] Run security scans
- [ ] Fix any critical vulnerabilities

### Week 3: Cleanup
- [ ] Archive old requirements.txt as requirements-legacy.txt
- [ ] Rename requirements-minimal.txt → requirements.txt
- [ ] Update all documentation references
- [ ] Close dependency tickets

---

## Risk Assessment

### Low Risk (Can remove immediately)
- Documentation tools (sphinx, etc.)
- Jupyter notebooks
- Profiling tools (py-spy, memory-profiler)
- Database migration tools (alembic)
- Test data generators (factory-boy, faker)

**Action**: Remove in Phase 1

### Medium Risk (Test first)
- Testing frameworks (pytest-xdist, pytest-benchmark)
- Code quality tools (pylint, semgrep)
- Development tools (tox, commitizen)

**Action**: Remove in Phase 2, test thoroughly

### High Risk (Keep until migrated)
- kubernetes (until TN-46 complete)
- fastapi/uvicorn (until dashboard migrated)
- jinja2 (until templates migrated)

**Action**: Keep, mark as temporary

---

**Conclusion**: 70% dependency reduction possible with minimal risk. Focus on security and essential tools only.

**Next Steps**:
1. Create requirements-minimal.txt files
2. Test locally
3. Update Docker images
4. Run security scans
5. Update CI/CD
