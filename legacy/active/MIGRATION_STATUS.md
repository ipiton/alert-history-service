# Active Legacy Code - Migration Status

**Last Updated**: 2025-01-09
**Status**: 🟢 Active in production (20% traffic)
**Sunset Date**: April 1, 2025

---

## Files and Migration Status

### API Endpoints

| File | Status | Go Status | Sunset Date |
|------|--------|-----------|-------------|
| `main.py` | 🟢 ACTIVE | ✅ Complete | When 100% traffic to Go |
| `api/legacy_adapter.py` | 🟢 ACTIVE | N/A (not needed) | Apr 1, 2025 |
| `api/dashboard_endpoints.py` | 🟢 ACTIVE | 🔄 TN-76 to TN-85 | Mar 2025 |
| `api/publishing_endpoints.py` | 🟢 ACTIVE | 🔄 TN-59 | Feb 2025 |
| `api/enrichment_endpoints.py` | 🟢 ACTIVE | ⚠️ Partial | Feb 2025 |
| `api/classification_endpoints.py` | 🟢 ACTIVE | 🔄 TN-71 to TN-73 | Mar 2025 |

### Services

| File | Status | Go Status | Sunset Date |
|------|--------|-----------|-------------|
| `services/target_discovery.py` | 🟢 ACTIVE | 🔄 TN-46 to TN-49 | Feb 2025 |
| `services/alert_publisher.py` | 🟢 ACTIVE | 🔄 TN-56 to TN-58 | Feb 2025 |

---

## Traffic Allocation

**Current**: 20% Python, 80% Go
**Target**: 0% Python by April 1, 2025

### Migration Timeline

```
Jan 2025  │ Feb 2025  │ Mar 2025  │ Apr 2025
  20%     │   10%     │    2%     │    0%
          │           │           │   SUNSET
```

---

## Blocking Issues

### Publishing System (TN-46 to TN-60)
**Status**: 🔴 CRITICAL
**ETA**: February 2025
**Impact**: Blocks full Python sunset

**Requires**:
- Target discovery (TN-46 to TN-49)
- Alert formatters (TN-51 to TN-55)
- Publishing core (TN-56 to TN-58)

### Dashboard (TN-76 to TN-85)
**Status**: 🟡 MEDIUM
**ETA**: March 2025
**Impact**: Can use Python dashboard temporarily

---

## Maintenance Policy

**Now - Feb 1, 2025**:
- ✅ Critical bugs
- ✅ Security patches
- ⚠️ Limited support

**Feb 1 - Mar 1, 2025**:
- ✅ Security only
- ❌ No bug fixes

**Mar 1 - Apr 1, 2025**:
- 🔒 Critical security only
- ❌ Nothing else

---

## Next Steps

1. **Complete TN-46 to TN-60** (Publishing) - Priority 1
2. **Shift traffic to 95% Go** - Week of Feb 1
3. **Complete TN-76 to TN-85** (Dashboard) - Priority 2
4. **Final sunset** - April 1, 2025

---

**Questions?** See [MIGRATION.md](../../MIGRATION.md)
