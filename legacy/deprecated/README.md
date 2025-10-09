# Deprecated Python Code

> **🔴 DELETION SCHEDULED: April 1, 2025**

## Status

This directory contains Python code that has been **fully replaced** by Go implementations and is scheduled for permanent deletion.

## Why These Files Are Here

All files in this directory are **duplicates** of functionality that now exists in the Go codebase:

- ✅ Go implementation is complete and tested
- ✅ Feature parity achieved
- ✅ Go version is production-ready
- ✅ No longer needed for any purpose

## Deletion Timeline

```
2025-01-09  │ 2025-02-01  │ 2025-03-01  │ 2025-04-01
    ▼       │      ▼      │      ▼      │      ▼
  Moved     │  Warning    │  Final      │  DELETED
  to here   │  Period     │  Warning    │  🔴🔴🔴
```

**Days until deletion**: 82 days

## ⚠️ DO NOT USE

**Do NOT**:
- ❌ Import this code in any project
- ❌ Fix bugs (fix in Go version)
- ❌ Add features (Go only)
- ❌ Update dependencies
- ❌ Reference in documentation
- ❌ Use in production

**Only Valid Use**: Emergency rollback (strongly discouraged)

## Files in This Directory

### Infrastructure Components

| File | Go Replacement | Reason for Deprecation |
|------|----------------|------------------------|
| `logging_config.py` | `go-app/pkg/logger/` | Go slog implementation |
| `config.py` | `go-app/internal/config/` | Viper config loader |

### Core Components

| File | Go Replacement | Reason for Deprecation |
|------|----------------|------------------------|
| `core/metrics.py` | `go-app/pkg/metrics/` | Native Prometheus metrics |
| `core/shutdown.py` | `go-app/cmd/server/main.go` | Built-in graceful shutdown |
| `core/stateless_manager.py` | N/A | Go is stateless by design |
| `core/base_classes.py` | N/A | Not needed in Go |

### Services

| File | Go Replacement | Reason for Deprecation |
|------|----------------|------------------------|
| `services/graceful_shutdown.py` | `cmd/server/main.go` | Native Go implementation |
| `services/health_checker.py` | `handlers/health.go` | Health endpoint in Go |
| `services/redis_cache.py` | `infrastructure/cache/` | go-redis v9 |

### API

| File | Go Replacement | Reason for Deprecation |
|------|----------------|------------------------|
| `api/health_endpoints.py` | `handlers/health.go` | /healthz, /readyz |
| `api/metrics.py` | `pkg/metrics/` | Prometheus /metrics |

### Database

| File | Go Replacement | Reason for Deprecation |
|------|----------------|------------------------|
| `database/migration_manager.py` | `infrastructure/migrations/` | Goose migrations |
| `cli/database_migrate.py` | `cmd/migrate/main.go` | Go migration CLI |

### Utils

| File | Go Replacement | Reason for Deprecation |
|------|----------------|------------------------|
| `utils/stateless_decorators.py` | N/A | Python-specific decorators |
| `utils/decorators.py` | N/A | Not needed in Go |

### Package Files

| File | Go Replacement | Reason for Deprecation |
|------|----------------|------------------------|
| `__init__.py` | N/A | Python package init |

**Total**: 16 files scheduled for deletion

## Migration Verification

Each file in this directory has been verified:

1. ✅ Go implementation exists
2. ✅ Go tests pass
3. ✅ Feature parity confirmed
4. ✅ No production dependencies
5. ✅ Safe to delete

See verification report: `tasks/python-cleanup/analysis/component-matrix.csv`

## If You Need This Code

### Option 1: Use Go Version (Recommended)

```bash
# Example: Instead of logging_config.py
# Use: go-app/pkg/logger/

import "alert-history/pkg/logger"

log := logger.New(logger.Config{
    Level: "info",
    Format: "json",
})
```

### Option 2: Retrieve from Git History

If you absolutely need to reference old Python code:

```bash
# Find when file was moved to legacy/
git log --follow -- legacy/deprecated/logging_config.py

# Retrieve old version
git show <commit>:src/alert_history/logging_config.py
```

## Rollback Procedure (Emergency Only)

**Only use if Go version has critical production issue**

```bash
# 1. Restore from this directory
cp legacy/deprecated/logging_config.py src/alert_history/

# 2. Reinstall Python dependencies
pip install -r requirements-legacy.txt

# 3. Restart Python service
kubectl scale deployment alert-history-python --replicas=3

# 4. IMMEDIATELY file critical bug report for Go version
```

**Note**: Rollback support ends March 1, 2025

## Deletion Process

### Phase 1 (Now - Feb 1, 2025)
- Files remain in `legacy/deprecated/`
- No modifications allowed
- Read-only access

### Phase 2 (Feb 1 - Mar 1, 2025)
- Final warning period
- Last chance to migrate dependencies
- Deprecation warnings in logs

### Phase 3 (Mar 1 - Apr 1, 2025)
- Imminent deletion warnings
- No rollback support
- Prepare for removal

### Phase 4 (Apr 1, 2025)
- **PERMANENT DELETION**
- All files removed from repository
- No recovery possible

## Questions?

**Why keep deprecated code for 3 months?**
- Safety buffer for rollbacks
- Time to identify hidden dependencies
- Gradual migration support

**Can I extend the deadline?**
- No, unless critical blockers
- See [DEPRECATION.md](../../DEPRECATION.md)

**What if I find a bug?**
- Fix in Go version, not here
- Report to #alert-history on Slack

---

**⚠️ REMINDER**: These files will be DELETED in 82 days

**Last Updated**: 2025-01-09
**Deletion Date**: 2025-04-01
