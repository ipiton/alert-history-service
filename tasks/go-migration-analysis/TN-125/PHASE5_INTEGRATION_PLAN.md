# TN-125 Phase 5: AlertGroupManager Integration Plan

**Status**: 🚧 IN PROGRESS (30% complete)
**Date**: 2025-11-04
**Estimated Completion**: 2-3 hours

---

## ✅ COMPLETED (Phase 5 - Part 1):

1. **DefaultGroupManager structure updated**:
   - Replaced `groups map[GroupKey]*AlertGroup` with `storage GroupStorage`
   - Updated comments to reflect distributed state
   - Updated mutex documentation (fingerprintIndex only)

2. **DefaultGroupManagerConfig updated**:
   - Added `Storage GroupStorage` field (required)
   - Updated validation logic (pending)

3. **Constructor signature updated**:
   - Changed `NewDefaultGroupManager(cfg)` → `NewDefaultGroupManager(ctx, cfg)`
   - Added context for storage operations
   - Added storage validation check
   - Added `restoreGroupsFromStorage()` call (pending implementation)

---

## 🚧 PENDING (Phase 5 - Part 2):

### 1. Validation Method
**File**: `manager.go`
**Method**: `DefaultGroupManagerConfig.Validate()`

```go
func (c *DefaultGroupManagerConfig) Validate() error {
    if c.KeyGenerator == nil {
        return fmt.Errorf("KeyGenerator cannot be nil")
    }
    if c.Config == nil {
        return fmt.Errorf("Config cannot be nil")
    }
    // ADD THIS:
    if c.Storage == nil {
        return fmt.Errorf("Storage cannot be nil (TN-125 requirement)")
    }
    return nil
}
```

### 2. Restore Method
**File**: `manager_impl.go`
**Method**: `restoreGroupsFromStorage()`

```go
// restoreGroupsFromStorage loads all groups from storage on startup (TN-125).
//
// This method is called during initialization to restore distributed state
// after service restart. It rebuilds the fingerprintIndex from stored groups.
//
// Performance: <500ms for 10,000 groups (parallel loading via storage.LoadAll)
func (m *DefaultGroupManager) restoreGroupsFromStorage(ctx context.Context) error {
    m.logger.Info("Restoring groups from storage...")

    groups, err := m.storage.LoadAll(ctx)
    if err != nil {
        return fmt.Errorf("load all groups: %w", err)
    }

    // Rebuild fingerprint index
    m.mu.Lock()
    defer m.mu.Unlock()

    for _, group := range groups {
        for fingerprint := range group.Alerts {
            m.fingerprintIndex[fingerprint] = group.Key
        }
    }

    m.logger.Info("Restored groups from storage",
        "count", len(groups),
        "fingerprints", len(m.fingerprintIndex))

    if m.metrics != nil {
        m.metrics.RecordGroupsRestored(len(groups))
    }

    return nil
}
```

### 3. Replace `m.groups` → `m.storage` (30+ locations)

**Files to update**: `manager_impl.go`

#### 3.1 `AddAlertToGroup` method
```go
// OLD:
m.mu.Lock()
group, exists := m.groups[groupKey]
if !exists {
    group = &AlertGroup{...}
    m.groups[groupKey] = group
}
m.mu.Unlock()

// NEW:
group, err := m.storage.Load(ctx, groupKey)
if err != nil {
    var notFoundErr *GroupNotFoundError
    if errors.As(err, &notFoundErr) {
        // Create new group
        group = &AlertGroup{...}
    } else {
        return nil, fmt.Errorf("load group: %w", err)
    }
}

// ... modify group ...

// Persist to storage
if err := m.storage.Store(ctx, group); err != nil {
    return nil, fmt.Errorf("store group: %w", err)
}
```

#### 3.2 `GetGroup` method
```go
// OLD:
m.mu.RLock()
group, exists := m.groups[groupKey]
m.mu.RUnlock()
if !exists {
    return nil, NewGroupNotFoundError(groupKey)
}
return group.Clone(), nil

// NEW:
group, err := m.storage.Load(ctx, groupKey)
if err != nil {
    return nil, err
}
return group.Clone(), nil
```

#### 3.3 `GetGroupByFingerprint` method
```go
// OLD:
m.mu.RLock()
groupKey, exists := m.fingerprintIndex[fingerprint]
m.mu.RUnlock()
if !exists {
    return nil, NewGroupNotFoundError("")
}
return m.GetGroup(ctx, groupKey)

// NEW: (same logic, but GetGroup now uses storage)
m.mu.RLock()
groupKey, exists := m.fingerprintIndex[fingerprint]
m.mu.RUnlock()
if !exists {
    return nil, NewGroupNotFoundError("")
}
return m.GetGroup(ctx, groupKey)
```

#### 3.4 `RemoveAlertFromGroup` method
```go
// OLD:
m.mu.Lock()
group, exists := m.groups[groupKey]
if !exists {
    m.mu.Unlock()
    return NewGroupNotFoundError(groupKey)
}
delete(group.Alerts, alert.Fingerprint)
m.mu.Unlock()

// NEW:
group, err := m.storage.Load(ctx, groupKey)
if err != nil {
    return err
}

group.mu.Lock()
delete(group.Alerts, alert.Fingerprint)
group.mu.Unlock()

if err := m.storage.Store(ctx, group); err != nil {
    return fmt.Errorf("store group: %w", err)
}
```

#### 3.5 `ListGroups` method
```go
// OLD:
m.mu.RLock()
defer m.mu.RUnlock()
for _, group := range m.groups {
    if opts != nil && !matchesFilters(group, opts) {
        continue
    }
    groups = append(groups, group.Clone())
}

// NEW:
keys, err := m.storage.ListKeys(ctx)
if err != nil {
    return nil, fmt.Errorf("list keys: %w", err)
}

for _, key := range keys {
    group, err := m.storage.Load(ctx, key)
    if err != nil {
        m.logger.Warn("Failed to load group during list", "key", key, "error", err)
        continue
    }

    if opts != nil && !matchesFilters(group, opts) {
        continue
    }
    groups = append(groups, group.Clone())
}
```

#### 3.6 `CleanupExpiredGroups` method
```go
// OLD:
m.mu.Lock()
defer m.mu.Unlock()
for key, group := range m.groups {
    if group.ShouldExpire(maxAge) {
        delete(m.groups, key)
        cleaned++
    }
}

// NEW:
keys, err := m.storage.ListKeys(ctx)
if err != nil {
    return 0, fmt.Errorf("list keys: %w", err)
}

for _, key := range keys {
    group, err := m.storage.Load(ctx, key)
    if err != nil {
        continue
    }

    if group.ShouldExpire(maxAge) {
        if err := m.storage.Delete(ctx, key); err != nil {
            m.logger.Error("Failed to delete expired group", "key", key, "error", err)
            continue
        }

        // Clean up fingerprint index
        m.mu.Lock()
        for fp := range group.Alerts {
            delete(m.fingerprintIndex, fp)
        }
        m.mu.Unlock()

        cleaned++
    }
}
```

### 4. Update All Callers

**Files to check**:
- `go-app/cmd/server/main.go`
- `go-app/internal/infrastructure/grouping/*_test.go`

```go
// OLD:
manager, err := grouping.NewDefaultGroupManager(grouping.DefaultGroupManagerConfig{
    KeyGenerator: keyGen,
    Config:       config,
    Logger:       logger,
    Metrics:      metrics,
})

// NEW:
manager, err := grouping.NewDefaultGroupManager(ctx, grouping.DefaultGroupManagerConfig{
    KeyGenerator: keyGen,
    Config:       config,
    Storage:      storageManager, // NEW!
    Logger:       logger,
    Metrics:      metrics,
})
```

---

## 📋 Testing Checklist

After integration:
- [ ] All unit tests pass
- [ ] Integration tests with Redis pass
- [ ] Benchmarks validate performance targets
- [ ] Test automatic fallback (Redis → Memory)
- [ ] Test automatic recovery (Memory → Redis)
- [ ] Test optimistic locking conflicts
- [ ] Test state restoration on startup
- [ ] Verify fingerprint index consistency

---

## 🎯 Performance Targets (150% Quality)

After integration, verify:
- AddAlertToGroup: <5ms (with Redis persistence)
- GetGroup: <2ms (Redis load)
- ListGroups: <100ms for 1,000 groups
- CleanupExpiredGroups: <500ms for 1,000 groups
- Startup restoration: <500ms for 10,000 groups

---

## ⚠️ Breaking Changes

**API Changes**:
```go
// BEFORE:
func NewDefaultGroupManager(cfg DefaultGroupManagerConfig) (*DefaultGroupManager, error)

// AFTER:
func NewDefaultGroupManager(ctx context.Context, cfg DefaultGroupManagerConfig) (*DefaultGroupManager, error)
```

**Config Changes**:
```go
// NEW REQUIRED FIELD:
type DefaultGroupManagerConfig struct {
    // ...
    Storage GroupStorage  // Required (was: none)
    // ...
}
```

---

## 📝 Commit Message (After Completion)

```
feat(go): TN-125 Phase 5 - AlertGroupManager integration with distributed storage

INTEGRATION:
- Replaced in-memory map with GroupStorage (Redis + Memory fallback)
- Added ctx to NewDefaultGroupManager signature
- Implemented restoreGroupsFromStorage for HA recovery
- Updated all methods: AddAlertToGroup, GetGroup, ListGroups, etc.
- Maintained fingerprint index in memory for O(1) lookup

CHANGES:
- 30+ locations updated (m.groups → m.storage)
- All methods now persist to storage
- Automatic state restoration on startup
- Optimistic locking support via AlertGroup.Version

BREAKING CHANGES:
- NewDefaultGroupManager now requires ctx parameter
- DefaultGroupManagerConfig now requires Storage field

TESTING:
- All existing tests updated
- New integration tests for distributed state
- Verified automatic fallback/recovery
- Performance targets met

TN-125: Group Storage (Redis Backend, distributed state)
Phase 5: AlertGroupManager Integration - COMPLETE
Files: 2 files modified (manager.go, manager_impl.go + main.go)
```

---

## 🚀 Next Steps

1. Complete validation method (5 min)
2. Implement restoreGroupsFromStorage (15 min)
3. Replace m.groups in AddAlertToGroup (30 min)
4. Replace m.groups in other methods (60 min)
5. Update all callers (15 min)
6. Run tests and fix issues (30 min)
7. Performance validation (15 min)

**Total estimated time**: 2-3 hours
