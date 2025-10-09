# Legacy Python Code

> **⚠️ WARNING**: This directory contains deprecated Python code

## Purpose

This directory contains Python code that is being phased out as part of the migration to Go. The code is organized into three categories:

```
legacy/
├── deprecated/     # Full duplicates of Go functionality - scheduled for deletion
├── reference/      # Complex implementations kept as reference material
├── active/         # Still-active legacy endpoints during transition
└── docs/          # Legacy documentation and migration artifacts
```

## Directory Descriptions

### `deprecated/`
**Status**: 🔴 **Scheduled for Deletion**
**Deletion Date**: April 1, 2025

Code that has been fully replaced by Go implementation. These files are kept for 3 months for emergency rollback purposes only.

**Do NOT**:
- ❌ Use this code in new features
- ❌ Fix bugs (use Go version)
- ❌ Add features (Go only)
- ❌ Update dependencies

**You MAY**:
- ✅ Reference for migration purposes
- ✅ Emergency rollback (last resort)

---

### `reference/`
**Status**: 🟡 **Reference Only**
**Purpose**: Documentation and algorithm reference

Complex Python implementations that may be useful as reference material during Go development. These are NOT actively maintained but preserved for their algorithmic or architectural value.

**Use Cases**:
- 📖 Understanding complex algorithms
- 🔍 Clarifying business logic
- 🧪 Comparing implementations
- 📝 Documentation reference

**Do NOT**:
- ❌ Run in production
- ❌ Import in active code
- ❌ Expect bug fixes

---

### `active/`
**Status**: 🟢 **Temporarily Active**
**Maintenance**: Security fixes only

Python code that is still serving production traffic during the migration transition. This code receives minimal maintenance (security patches only) until Go equivalents are complete.

**Timeline**:
- Until Feb 2025: Limited support
- Until Mar 2025: Security fixes only
- Apr 1, 2025: DELETED

**Migration Status**: See `active/MIGRATION_STATUS.md`

---

## Migration Timeline

| Date | Event |
|------|-------|
| 2025-01-09 | Python code moved to `legacy/` |
| 2025-02-01 | Deprecation officially announced |
| 2025-03-01 | Security fixes only |
| 2025-04-01 | **All Python code DELETED** |

## Finding Go Equivalents

For each legacy file, see the mapping:

```
legacy/deprecated/logging_config.py     → go-app/pkg/logger/
legacy/deprecated/core/metrics.py       → go-app/pkg/metrics/
legacy/reference/alert_classifier.py    → go-app/internal/infrastructure/llm/
legacy/reference/filter_engine.py       → go-app/internal/core/filtering.go
legacy/active/main.py                   → go-app/cmd/server/main.go
```

**Full Mapping**: See `tasks/python-cleanup/analysis/component-matrix.csv`

## Documentation

- 📖 [MIGRATION.md](../MIGRATION.md) - Migration guide
- 📅 [DEPRECATION.md](../DEPRECATION.md) - Deprecation timeline
- 📊 [Component Matrix](../tasks/python-cleanup/analysis/component-matrix.csv) - Python → Go mapping
- 🔍 [Migration Gaps](../tasks/python-cleanup/analysis/migration-gaps.md) - What's missing in Go

## Support

**Questions about legacy code?**
- 💬 Slack: #python-sunset
- 📧 Email: legacy-support@example.com
- 🐛 Issues: Tag with `legacy` label

**Need help migrating?**
- 📖 See [MIGRATION.md](../MIGRATION.md)
- 💬 Slack: #alert-history-migration

---

**Last Updated**: 2025-01-09
**Python Sunset**: April 1, 2025 (82 days)
