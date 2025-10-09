# Active Legacy Python Code

> **🟢 Temporarily Active - Migration in Progress**

## Status

This directory contains Python code that is **still serving production traffic** during the migration transition period. This code receives **minimal maintenance** (security fixes only) until Go equivalents are complete.

## ⚠️ Temporary Status

**Current**: Active in production
**Support**: Security fixes only (after Mar 1, 2025)
**Sunset**: April 1, 2025 (82 days)

## Files in This Directory

### Production Endpoints

#### `main.py`
**Purpose**: Python FastAPI application entry point
**Status**: 🟢 ACTIVE
**Go Equivalent**: `go-app/cmd/server/main.go` (✅ Complete)
**Sunset Date**: When Go reaches 100% traffic

**Why Still Active**:
- Serving legacy API endpoints
- Dashboard HTML rendering
- Publishing system (until TN-46 to TN-60 complete)

**Migration Plan**: Gradual traffic shift to Go (90% complete)

---

#### `api/legacy_adapter.py`
**Purpose**: Backward compatibility layer for old API clients
**Status**: 🟢 ACTIVE
**Go Equivalent**: Not needed (clients should update)
**Sunset Date**: April 1, 2025

**Why Still Active**:
- Some clients haven't migrated to new endpoints
- Provides `/health` → `/healthz` redirect
- Old pagination format support

**Migration Plan**:
- Feb 2025: Notify all clients to upgrade
- Mar 2025: Remove support

---

#### `api/dashboard_endpoints.py`
**Purpose**: HTML dashboard rendering
**Status**: 🟢 ACTIVE
**Go Equivalent**: TBD (TN-76 to TN-85, ETA March 2025)
**Sunset Date**: When Go dashboard complete

**Why Still Active**:
- Go dashboard not implemented yet
- Users rely on visual interface
- Cannot remove until replacement ready

**Migration Plan**:
- Feb 2025: Start TN-76 (Dashboard in Go)
- Mar 2025: Deploy Go dashboard
- Apr 2025: Sunset Python dashboard

---

#### `api/publishing_endpoints.py`
**Purpose**: Publishing configuration and stats API
**Status**: 🟢 ACTIVE
**Go Equivalent**: TBD (TN-59, ETA February 2025)
**Sunset Date**: When Go publishing API complete

**Why Still Active**:
- Publishing system core functionality
- High traffic endpoint
- Critical for production

**Migration Plan**:
- Feb 2025: Complete TN-46 to TN-60 (Publishing in Go)
- Feb 2025: Deploy Go publishing API
- Mar 2025: Sunset Python publishing

---

#### `api/enrichment_endpoints.py`
**Purpose**: Enrichment mode switching API
**Status**: 🟢 ACTIVE
**Go Equivalent**: Partial (`internal/core/enrichment.go` has core, needs API)
**Sunset Date**: March 2025

**Why Still Active**:
- Mode switching frequently used
- Go API layer not complete
- Low complexity, easy to port

**Migration Plan**:
- Feb 2025: Add Go enrichment API
- Mar 2025: Deprecate Python

---

## Maintenance Policy

### What We DO Maintain

**Until February 1, 2025**:
- ✅ Critical bug fixes
- ✅ Security patches
- ✅ Performance issues (if severe)
- ✅ Data corruption bugs

**February 1 - March 1, 2025**:
- ✅ Security patches only
- ⚠️ Critical bugs (case-by-case)
- ❌ No feature updates
- ❌ No dependency updates (unless security)

**March 1 - April 1, 2025**:
- ✅ Critical security only
- ❌ Nothing else

### What We DON'T Maintain

**Never**:
- ❌ New features (Go only)
- ❌ Refactoring
- ❌ Performance optimizations
- ❌ Code quality improvements
- ❌ Documentation updates
- ❌ Test coverage improvements

---

## Traffic Allocation

### Current Status

```
Go Version:    ████████████████░░ 80%
Python Active: ████░░░░░░░░░░░░░░ 20%
```

### Migration Timeline

| Date | Go Traffic | Python Traffic |
|------|-----------|----------------|
| 2025-01-09 | 80% | 20% |
| 2025-01-20 | 90% | 10% |
| 2025-02-01 | 95% | 5% |
| 2025-02-15 | 98% | 2% (dashboard only) |
| 2025-03-15 | 99.5% | 0.5% (legacy adapter only) |
| 2025-04-01 | 100% | 0% (SUNSET) |

---

## Deployment Status

### Production Environments

#### Production
**Status**: 🟢 Running
**Replicas**: 2 (reduced from 5)
**Traffic**: 20%
**Purpose**: Legacy endpoints, dashboard

#### Staging
**Status**: 🟢 Running
**Replicas**: 1
**Purpose**: Testing rollback procedures

#### Development
**Status**: ⚠️ Deprecated
**Replicas**: 0
**Purpose**: Use Go version

---

## Deprecation Warnings

### Application Logs

All active endpoints now log deprecation warnings:

```python
@app.post("/webhook")
async def webhook_endpoint():
    logger.warning(
        "DEPRECATION: Python /webhook endpoint called. "
        "Migrate to Go version: POST /webhook on port 8080. "
        "Python sunset: April 1, 2025"
    )
    # ... existing logic
```

### HTTP Headers

All responses include deprecation headers:

```http
HTTP/1.1 200 OK
Deprecation: true
Sunset: Sat, 01 Apr 2025 00:00:00 GMT
Link: </go-api>; rel="alternate"; type="application/json"
```

### API Response

All JSON responses include deprecation notice:

```json
{
  "data": {...},
  "_deprecation": {
    "deprecated": true,
    "sunset_date": "2025-04-01",
    "replacement": "http://alert-history:8080",
    "migration_guide": "https://github.com/.../MIGRATION.md"
  }
}
```

---

## For Developers

### Making Changes

**If you MUST change active Python code**:

1. **Check if urgent**:
   - 🔴 Security vulnerability? → Yes, fix
   - 🟡 Critical production bug? → Maybe, case-by-case
   - 🟢 Feature request? → No, do in Go

2. **Open issue first**:
   - Label: `legacy-python`, `security` or `critical`
   - Justify why can't wait for Go version
   - Get approval from tech lead

3. **Minimal changes only**:
   - Smallest possible fix
   - No refactoring
   - No "while I'm here" changes

4. **Update Go version**:
   - Same fix should go to Go
   - Go fix takes priority
   - Ensure behavior parity

---

### Testing Changes

**Required Tests**:
- ✅ Unit tests pass (existing tests only)
- ✅ Integration tests pass
- ✅ No new dependencies
- ✅ Go version has equivalent fix

**NOT Required**:
- ❌ New test coverage
- ❌ Performance benchmarks
- ❌ Code quality improvements

---

### Deployment Process

**For Security Patches**:
```bash
# 1. Fix + test locally
pytest tests/

# 2. Deploy to staging
kubectl set image deployment/alert-history-python \
  alert-history=alert-history:python-patch-X

# 3. Monitor for 24 hours

# 4. Deploy to production (if stable)
kubectl set image deployment/alert-history-python \
  alert-history=alert-history:python-patch-X \
  --namespace=production
```

**For Critical Bugs**:
- Same process as security patches
- Requires tech lead approval
- Must have corresponding Go fix

---

## Monitoring

### Key Metrics

**Health**:
- `alert_history_python_up` - Service availability
- `alert_history_python_requests_total` - Request count
- `alert_history_python_errors_total` - Error rate

**Traffic**:
- `alert_history_python_traffic_percent` - % of total traffic
- `alert_history_python_active_connections` - Open connections

**Deprecation**:
- `alert_history_python_deprecation_warnings` - Deprecation notices served

### Alerts

**Critical Alerts**:
- Python service down (PagerDuty)
- Error rate > 5% (PagerDuty)
- Security vulnerability detected (Immediate)

**Warning Alerts**:
- Traffic > 30% (should be decreasing)
- Memory usage > 80%
- Response time > 2s

---

## Rollback from Go

**If Go version has critical issues**:

### Immediate Rollback (<5 min)
```bash
# Increase Python replicas
kubectl scale deployment alert-history-python --replicas=5

# Route traffic to Python
kubectl patch service alert-history \
  --patch '{"spec":{"selector":{"app":"alert-history-python"}}}'
```

### Verify Rollback
```bash
# Check Python health
curl http://alert-history/health

# Monitor metrics
open https://grafana.example.com/d/alert-history-python

# Check error rate
kubectl logs -l app=alert-history-python | grep ERROR
```

**Note**: Rollback support ends March 1, 2025

---

## Sunset Procedure

### Pre-Sunset Checklist

**30 Days Before (Mar 1)**:
- [ ] Verify all Go endpoints working
- [ ] Check no Python-specific traffic remains
- [ ] Notify all API consumers
- [ ] Update documentation

**7 Days Before (Mar 25)**:
- [ ] Final migration reminder
- [ ] Check for hard dependencies
- [ ] Prepare deletion scripts

**1 Day Before (Mar 31)**:
- [ ] Scale Python to 1 replica (safety)
- [ ] Monitor for unexpected traffic
- [ ] Confirm deletion approval

### Sunset Day (Apr 1)

```bash
# 1. Scale to zero
kubectl scale deployment alert-history-python --replicas=0

# 2. Remove from service
kubectl delete service alert-history-python

# 3. Delete deployment
kubectl delete deployment alert-history-python

# 4. Remove from Git
git rm -r legacy/active/
git commit -m "feat: Remove Python version (sunset)"
git push origin main

# 5. Celebrate! 🎉
```

---

## Questions?

**Can I add a feature to Python code?**
- No. Implement in Go version only.

**What if Go version is missing critical feature?**
- Prioritize Go implementation
- Keep Python active until Go ready
- May extend timeline if needed

**Who approves Python changes?**
- Security patches: Security team
- Critical bugs: Tech lead
- Everything else: No approval (blocked)

**How do I report Python issues?**
- Security: security@example.com (urgent)
- Bugs: #alert-history Slack
- Questions: #python-sunset Slack

---

**Status**: 🟢 Active (20% traffic)
**Sunset**: April 1, 2025 (82 days)
**Last Updated**: 2025-01-09
