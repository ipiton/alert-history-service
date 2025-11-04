package grouping

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/core"
	"github.com/vitaliisemenov/alert-history/pkg/metrics"
)

// DefaultGroupManager is an in-memory implementation of AlertGroupManager.
//
// Thread-safety: All methods are thread-safe via sync.RWMutex.
// Concurrent reads and writes are properly synchronized.
//
// Performance:
//   - AddAlertToGroup: O(1) map lookup + lock
//   - GetGroup: O(1) map lookup + RLock
//   - ListGroups: O(n) iteration with filtering
//   - RemoveAlertFromGroup: O(1) map deletion + lock
//   - CleanupExpiredGroups: O(n) iteration
//
// Memory: ~5KB per group (target: <10KB baseline, <5KB at 150%)
type DefaultGroupManager struct {
	// storage persists alert groups (Redis primary + Memory fallback) (TN-125)
	// Replaces in-memory groups map for distributed state management
	storage GroupStorage

	// fingerprintIndex is a reverse index for fast lookup: map[fingerprint]GroupKey
	// 150% Enhancement: Enables O(1) lookup of group by alert fingerprint
	// NOTE: This remains in-memory for performance. Groups are in storage.
	fingerprintIndex map[string]GroupKey

	// mu protects concurrent access to fingerprintIndex
	// NOTE: Groups are protected by storage's internal locking
	mu sync.RWMutex

	// keyGenerator generates group keys from alert labels (from TN-122)
	keyGenerator *GroupKeyGenerator

	// config is the grouping configuration (from TN-121)
	config *GroupingConfig

	// timerManager manages group timers (group_wait, group_interval) (TN-124)
	// Optional: can be nil for backwards compatibility
	timerManager GroupTimerManager

	// logger for structured logging
	logger *slog.Logger

	// metrics for Prometheus integration
	metrics *metrics.BusinessMetrics

	// stats tracks operation statistics
	stats *groupStats
}

// groupStats stores internal statistics for operations.
//
// Thread-safety: Protected by its own mutex for lock-free access from methods.
type groupStats struct {
	totalAdds       int64
	totalRemoves    int64
	totalCleanups   int64
	totalUpdates    int64
	lastCleanupTime time.Time
	mu              sync.RWMutex
}

// NewDefaultGroupManager creates a new in-memory group manager.
//
// Example:
//
//	manager, err := NewDefaultGroupManager(DefaultGroupManagerConfig{
//	    KeyGenerator: keyGen,
//	    Config:       config,
//	    Logger:       slog.Default(),
//	    Metrics:      businessMetrics,
//	})
func NewDefaultGroupManager(ctx context.Context, cfg DefaultGroupManagerConfig) (*DefaultGroupManager, error) {
	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// Set defaults
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	// Storage is required for distributed state (TN-125)
	if cfg.Storage == nil {
		return nil, fmt.Errorf("storage cannot be nil (TN-125 requirement)")
	}

	mgr := &DefaultGroupManager{
		storage:          cfg.Storage,
		fingerprintIndex: make(map[string]GroupKey),
		keyGenerator:     cfg.KeyGenerator,
		config:           cfg.Config,
		timerManager:     cfg.TimerManager, // Optional (TN-124)
		logger:           cfg.Logger,
		metrics:          cfg.Metrics,
		stats:            &groupStats{},
	}

	// Register timer callbacks if timer manager is configured (TN-124)
	if err := mgr.registerTimerCallbacks(); err != nil {
		return nil, fmt.Errorf("register timer callbacks: %w", err)
	}

	// Restore groups from storage on startup (TN-125)
	if err := mgr.restoreGroupsFromStorage(ctx); err != nil {
		return nil, fmt.Errorf("restore groups from storage: %w", err)
	}

	return mgr, nil
}

// === Lifecycle Management Implementation ===

// AddAlertToGroup implements AlertGroupManager.AddAlertToGroup.
func (m *DefaultGroupManager) AddAlertToGroup(
	ctx context.Context,
	alert *core.Alert,
	groupKey GroupKey,
) (*AlertGroup, error) {
	startTime := time.Now()

	// Validation
	if alert == nil {
		return nil, &InvalidAlertError{Reason: "alert is nil"}
	}
	if alert.Fingerprint == "" {
		return nil, &InvalidAlertError{Reason: "alert fingerprint is empty"}
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Get or create group
	group, exists := m.groups[groupKey]
	if !exists {
		group = m.createNewGroupUnsafe(groupKey)
		m.groups[groupKey] = group

		m.logger.Info("created new alert group",
			"group_key", groupKey,
			"alert", alert.AlertName,
			"fingerprint", alert.Fingerprint)

		// Metric: new group created
		if m.metrics != nil {
			m.metrics.IncActiveGroups()
		}

		// Start group_wait timer for new group (TN-124)
		if err := m.startGroupWaitTimer(ctx, groupKey); err != nil {
			// Log error but don't fail the operation (timer is optional)
			m.logger.Warn("failed to start group_wait timer for new group",
				"group_key", groupKey,
				"error", err)
		}
	}

	// Add alert to group (thread-safe)
	group.mu.Lock()
	isNewAlert := group.Alerts[alert.Fingerprint] == nil
	group.Alerts[alert.Fingerprint] = alert
	group.mu.Unlock()

	// Update fingerprint index
	m.fingerprintIndex[alert.Fingerprint] = groupKey

	// Update group state
	m.updateGroupStateUnsafe(group)

	// Update stats
	m.stats.mu.Lock()
	m.stats.totalAdds++
	m.stats.mu.Unlock()

	// Metrics
	if m.metrics != nil {
		m.recordAddMetrics(groupKey, isNewAlert, time.Since(startTime))
	}

	m.logger.Debug("added alert to group",
		"group_key", groupKey,
		"alert", alert.AlertName,
		"fingerprint", alert.Fingerprint,
		"group_size", len(group.Alerts),
		"is_new", isNewAlert,
		"state", group.Metadata.State)

	// Return shallow copy (150% enhancement: prevent external mutation)
	return group.Clone(), nil
}

// RemoveAlertFromGroup implements AlertGroupManager.RemoveAlertFromGroup.
func (m *DefaultGroupManager) RemoveAlertFromGroup(
	ctx context.Context,
	fingerprint string,
	groupKey GroupKey,
) (bool, error) {
	startTime := time.Now()

	// Check context cancellation
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Find group
	group, exists := m.groups[groupKey]
	if !exists {
		return false, &GroupNotFoundError{Key: groupKey}
	}

	// Remove alert from group
	group.mu.Lock()
	_, existed := group.Alerts[fingerprint]
	delete(group.Alerts, fingerprint)
	groupSize := len(group.Alerts)
	group.mu.Unlock()

	if !existed {
		return false, nil // Alert wasn't in the group
	}

	// Remove from fingerprint index
	delete(m.fingerprintIndex, fingerprint)

	// If group is empty - delete group
	if groupSize == 0 {
		delete(m.groups, groupKey)

		m.logger.Info("deleted empty alert group",
			"group_key", groupKey)

		// Metric: group deleted
		if m.metrics != nil {
			m.metrics.DecActiveGroups()
		}

		// Cancel all timers for this group (TN-124)
		m.cancelGroupTimers(ctx, groupKey)
	} else {
		// Update group state
		m.updateGroupStateUnsafe(group)
	}

	// Update stats
	m.stats.mu.Lock()
	m.stats.totalRemoves++
	m.stats.mu.Unlock()

	// Metrics
	if m.metrics != nil {
		m.recordRemoveMetrics(groupKey, time.Since(startTime))
	}

	m.logger.Debug("removed alert from group",
		"group_key", groupKey,
		"fingerprint", fingerprint,
		"group_size", groupSize,
		"group_deleted", groupSize == 0)

	return true, nil
}

// UpdateGroupState implements AlertGroupManager.UpdateGroupState.
func (m *DefaultGroupManager) UpdateGroupState(
	ctx context.Context,
	groupKey GroupKey,
) (*AlertGroup, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Find group
	group, exists := m.groups[groupKey]
	if !exists {
		return nil, &GroupNotFoundError{Key: groupKey}
	}

	// Update state
	m.updateGroupStateUnsafe(group)

	// Update stats
	m.stats.mu.Lock()
	m.stats.totalUpdates++
	m.stats.mu.Unlock()

	m.logger.Debug("updated group state",
		"group_key", groupKey,
		"state", group.Metadata.State,
		"firing_count", group.Metadata.FiringCount,
		"resolved_count", group.Metadata.ResolvedCount)

	return group.Clone(), nil
}

// CleanupExpiredGroups implements AlertGroupManager.CleanupExpiredGroups.
func (m *DefaultGroupManager) CleanupExpiredGroups(
	ctx context.Context,
	maxAge time.Duration,
) (int, error) {
	startTime := time.Now()

	// Check context cancellation
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Find expired groups
	expiredKeys := make([]GroupKey, 0)
	for key, group := range m.groups {
		if group.IsExpired(maxAge) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	// Delete expired groups
	for _, key := range expiredKeys {
		group := m.groups[key]

		// Remove all fingerprints from index
		group.mu.RLock()
		for fingerprint := range group.Alerts {
			delete(m.fingerprintIndex, fingerprint)
		}
		group.mu.RUnlock()

		// Delete group
		delete(m.groups, key)
	}

	deletedCount := len(expiredKeys)

	// Update stats
	m.stats.mu.Lock()
	m.stats.totalCleanups += int64(deletedCount)
	m.stats.lastCleanupTime = startTime
	m.stats.mu.Unlock()

	// Metrics
	if m.metrics != nil {
		m.recordCleanupMetrics(deletedCount, time.Since(startTime))
	}

	m.logger.Info("cleaned up expired groups",
		"deleted_count", deletedCount,
		"max_age", maxAge,
		"duration", time.Since(startTime))

	return deletedCount, nil
}

// === Query Operations Implementation ===

// GetGroup implements AlertGroupManager.GetGroup.
func (m *DefaultGroupManager) GetGroup(
	ctx context.Context,
	groupKey GroupKey,
) (*AlertGroup, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	group, exists := m.groups[groupKey]
	if !exists {
		return nil, &GroupNotFoundError{Key: groupKey}
	}

	// Return shallow copy (150% enhancement: prevent external mutation)
	return group.Clone(), nil
}

// ListGroups implements AlertGroupManager.ListGroups.
func (m *DefaultGroupManager) ListGroups(
	ctx context.Context,
	filters *GroupFilters,
) ([]*AlertGroup, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Pre-allocate result slice (150% optimization)
	result := make([]*AlertGroup, 0, len(m.groups))

	// Apply filters and collect matching groups
	offset := 0
	limit := 0
	if filters != nil {
		limit = filters.Limit
	}

	for _, group := range m.groups {
		// Check if group matches filters
		if filters != nil && !filters.Matches(group) {
			continue
		}

		// Apply offset (pagination)
		if filters != nil && filters.Offset > 0 && offset < filters.Offset {
			offset++
			continue
		}

		// Add group clone to result
		result = append(result, group.Clone())

		// Apply limit (pagination)
		if limit > 0 && len(result) >= limit {
			break
		}
	}

	return result, nil
}

// GetGroupByFingerprint implements AlertGroupManager.GetGroupByFingerprint.
//
// 150% Enhancement: Reverse lookup using fingerprint index.
func (m *DefaultGroupManager) GetGroupByFingerprint(
	ctx context.Context,
	fingerprint string,
) (GroupKey, *AlertGroup, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return "", nil, ctx.Err()
	default:
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Lookup in fingerprint index
	groupKey, exists := m.fingerprintIndex[fingerprint]
	if !exists {
		return "", nil, &GroupNotFoundError{Key: GroupKey(fmt.Sprintf("fingerprint=%s", fingerprint))}
	}

	// Get group
	group, exists := m.groups[groupKey]
	if !exists {
		// Index inconsistency (should not happen)
		m.logger.Error("fingerprint index inconsistency",
			"fingerprint", fingerprint,
			"group_key", groupKey)
		return "", nil, &GroupNotFoundError{Key: groupKey}
	}

	return groupKey, group.Clone(), nil
}

// === Metrics & Observability Implementation ===

// GetMetrics implements AlertGroupManager.GetMetrics.
func (m *DefaultGroupManager) GetMetrics(ctx context.Context) (*GroupMetrics, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Collect metrics
	alertsPerGroup := make(map[string]int, len(m.groups))
	sizeDistribution := map[string]int{
		"1-10":     0,
		"11-50":    0,
		"51-100":   0,
		"101-500":  0,
		"501-1000": 0,
		"1000+":    0,
	}

	for key, group := range m.groups {
		size := group.Size()
		alertsPerGroup[string(key)] = size

		// Calculate size distribution
		switch {
		case size <= 10:
			sizeDistribution["1-10"]++
		case size <= 50:
			sizeDistribution["11-50"]++
		case size <= 100:
			sizeDistribution["51-100"]++
		case size <= 500:
			sizeDistribution["101-500"]++
		case size <= 1000:
			sizeDistribution["501-1000"]++
		default:
			sizeDistribution["1000+"]++
		}
	}

	// Get operation stats
	m.stats.mu.RLock()
	operations := map[string]int64{
		"add":     m.stats.totalAdds,
		"remove":  m.stats.totalRemoves,
		"cleanup": m.stats.totalCleanups,
	}
	m.stats.mu.RUnlock()

	return &GroupMetrics{
		ActiveGroups:     len(m.groups),
		AlertsPerGroup:   alertsPerGroup,
		SizeDistribution: sizeDistribution,
		Operations:       operations,
		Timestamp:        time.Now(),
	}, nil
}

// GetStats implements AlertGroupManager.GetStats.
//
// 150% Enhancement: Extended statistics for advanced monitoring.
func (m *DefaultGroupManager) GetStats(ctx context.Context) (*GroupStats, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Calculate totals
	totalAlerts := 0
	firingAlerts := 0
	resolvedAlerts := 0

	for _, group := range m.groups {
		group.mu.RLock()
		totalAlerts += len(group.Alerts)
		firingAlerts += group.Metadata.FiringCount
		resolvedAlerts += group.Metadata.ResolvedCount
		group.mu.RUnlock()
	}

	// Estimate memory usage (approximate)
	// ~5KB per group: struct overhead + alerts map + metadata
	estimatedMemory := int64(len(m.groups) * 5 * 1024)

	// Get operation stats
	m.stats.mu.RLock()
	stats := &GroupStats{
		TotalAdds:            m.stats.totalAdds,
		TotalRemoves:         m.stats.totalRemoves,
		TotalCleanups:        m.stats.totalCleanups,
		TotalUpdates:         m.stats.totalUpdates,
		LastCleanupTime:      m.stats.lastCleanupTime,
		ActiveGroups:         len(m.groups),
		TotalAlerts:          totalAlerts,
		FiringAlerts:         firingAlerts,
		ResolvedAlerts:       resolvedAlerts,
		EstimatedMemoryBytes: estimatedMemory,
		Timestamp:            time.Now(),
	}
	m.stats.mu.RUnlock()

	return stats, nil
}

// === Internal Helper Methods ===

// createNewGroupUnsafe creates a new empty group.
//
// Caller must hold write lock (m.mu.Lock).
func (m *DefaultGroupManager) createNewGroupUnsafe(groupKey GroupKey) *AlertGroup {
	now := time.Now()

	return &AlertGroup{
		Key:    groupKey,
		Alerts: make(map[string]*core.Alert),
		Metadata: &GroupMetadata{
			State:     GroupStateFiring, // Initial state (will be updated)
			CreatedAt: now,
			UpdatedAt: now,
			GroupBy:   m.config.Route.GroupBy,
			Version:   1,
		},
	}
}

// updateGroupStateUnsafe updates the state of a group based on alert statuses.
//
// Caller must hold write lock (m.mu.Lock).
func (m *DefaultGroupManager) updateGroupStateUnsafe(group *AlertGroup) {
	group.mu.Lock()
	defer group.mu.Unlock()

	group.Metadata.UpdateState(group.Alerts)
}

// recordAddMetrics records Prometheus metrics for AddAlertToGroup operation.
func (m *DefaultGroupManager) recordAddMetrics(groupKey GroupKey, isNew bool, duration time.Duration) {
	m.metrics.RecordGroupOperation("add", "success")
	m.metrics.RecordGroupOperationDuration("add", duration)

	// Record group size histogram (async to avoid lock contention)
	// Note: This is a simplified version. Real implementation would be in pkg/metrics/business.go
}

// recordRemoveMetrics records Prometheus metrics for RemoveAlertFromGroup operation.
func (m *DefaultGroupManager) recordRemoveMetrics(groupKey GroupKey, duration time.Duration) {
	m.metrics.RecordGroupOperation("remove", "success")
	m.metrics.RecordGroupOperationDuration("remove", duration)
}

// recordCleanupMetrics records Prometheus metrics for CleanupExpiredGroups operation.
func (m *DefaultGroupManager) recordCleanupMetrics(deletedCount int, duration time.Duration) {
	m.metrics.RecordGroupOperation("cleanup", "success")
	m.metrics.RecordGroupOperationDuration("cleanup", duration)
	m.metrics.RecordGroupsCleanedUp(deletedCount)
}

// === Timer Integration (TN-124) ===

// startGroupWaitTimer starts a group_wait timer for a newly created group.
// This timer delays the first notification until group_wait duration elapses.
//
// Called when a new group is created in AddAlertToGroup.
func (m *DefaultGroupManager) startGroupWaitTimer(ctx context.Context, groupKey GroupKey) error {
	if m.timerManager == nil {
		return nil // Timer functionality disabled (backwards compatible)
	}

	// Get group_wait duration from config (default: 30s)
	duration := 30 * time.Second
	if m.config != nil && m.config.Route != nil && m.config.Route.GroupWait != nil {
		duration = m.config.Route.GroupWait.Duration
	}

	// Start group_wait timer
	_, err := m.timerManager.StartTimer(ctx, groupKey, GroupWaitTimer, duration)
	if err != nil {
		m.logger.Error("failed to start group_wait timer",
			"group_key", groupKey,
			"duration", duration,
			"error", err)
		return fmt.Errorf("start group_wait timer: %w", err)
	}

	m.logger.Debug("started group_wait timer",
		"group_key", groupKey,
		"duration", duration)

	return nil
}

// startGroupIntervalTimer starts a group_interval timer for an existing group.
// This timer ensures minimum time between notifications for the same group.
//
// Called after a notification is sent for a group.
func (m *DefaultGroupManager) startGroupIntervalTimer(ctx context.Context, groupKey GroupKey) error {
	if m.timerManager == nil {
		return nil // Timer functionality disabled
	}

	// Get group_interval duration from config (default: 5m)
	duration := 5 * time.Minute
	if m.config != nil && m.config.Route != nil && m.config.Route.GroupInterval != nil {
		duration = m.config.Route.GroupInterval.Duration
	}

	// Start group_interval timer
	_, err := m.timerManager.StartTimer(ctx, groupKey, GroupIntervalTimer, duration)
	if err != nil {
		m.logger.Error("failed to start group_interval timer",
			"group_key", groupKey,
			"duration", duration,
			"error", err)
		return fmt.Errorf("start group_interval timer: %w", err)
	}

	m.logger.Debug("started group_interval timer",
		"group_key", groupKey,
		"duration", duration)

	return nil
}

// cancelGroupTimers cancels all timers for a group.
// Called when a group is deleted (empty after alert removal).
func (m *DefaultGroupManager) cancelGroupTimers(ctx context.Context, groupKey GroupKey) {
	if m.timerManager == nil {
		return // Timer functionality disabled
	}

	// Cancel group_wait timer (if exists)
	if _, err := m.timerManager.CancelTimer(ctx, groupKey); err != nil {
		m.logger.Warn("failed to cancel group timer",
			"group_key", groupKey,
			"error", err)
	} else {
		m.logger.Debug("cancelled group timers",
			"group_key", groupKey)
	}
}

// onGroupWaitExpired is the callback for group_wait timer expiration.
// This sends the first notification for a group after the initial delay.
func (m *DefaultGroupManager) onGroupWaitExpired(ctx context.Context, groupKey GroupKey, timerType TimerType, group *AlertGroup) error {
	m.logger.Info("group_wait timer expired, ready to send first notification",
		"group_key", groupKey,
		"alert_count", len(group.Alerts))

	// TODO: Trigger notification here (will be implemented in TN-125)
	// For now, just start the group_interval timer

	// Start group_interval timer for subsequent notifications
	if err := m.startGroupIntervalTimer(ctx, groupKey); err != nil {
		m.logger.Error("failed to start group_interval timer after group_wait",
			"group_key", groupKey,
			"error", err)
		return err
	}

	return nil
}

// onGroupIntervalExpired is the callback for group_interval timer expiration.
// This allows sending subsequent notifications for a group.
func (m *DefaultGroupManager) onGroupIntervalExpired(ctx context.Context, groupKey GroupKey, timerType TimerType, group *AlertGroup) error {
	m.logger.Info("group_interval timer expired, ready to send update notification",
		"group_key", groupKey,
		"alert_count", len(group.Alerts))

	// TODO: Trigger notification here (will be implemented in TN-125)
	// For now, just restart the group_interval timer if group still has alerts

	// Check if group still exists and has alerts
	m.mu.RLock()
	currentGroup, exists := m.groups[groupKey]
	m.mu.RUnlock()

	if !exists || len(currentGroup.Alerts) == 0 {
		m.logger.Debug("group no longer exists or is empty, not restarting timer",
			"group_key", groupKey)
		return nil
	}

	// Restart group_interval timer for next notification
	if err := m.startGroupIntervalTimer(ctx, groupKey); err != nil {
		m.logger.Error("failed to restart group_interval timer",
			"group_key", groupKey,
			"error", err)
		return err
	}

	return nil
}

// registerTimerCallbacks registers timer expiration callbacks with the timer manager.
// This should be called during manager initialization.
func (m *DefaultGroupManager) registerTimerCallbacks() error {
	if m.timerManager == nil {
		return nil // Timer functionality disabled
	}

	// Register callback for all timer types
	m.timerManager.OnTimerExpired(func(ctx context.Context, groupKey GroupKey, timerType TimerType, group *AlertGroup) error {
		switch timerType {
		case GroupWaitTimer:
			return m.onGroupWaitExpired(ctx, groupKey, timerType, group)
		case GroupIntervalTimer:
			return m.onGroupIntervalExpired(ctx, groupKey, timerType, group)
		case RepeatIntervalTimer:
			// RepeatInterval not yet implemented (future enhancement)
			m.logger.Debug("repeat_interval timer expired (not implemented)",
				"group_key", groupKey)
			return nil
		default:
			m.logger.Warn("unknown timer type expired",
				"group_key", groupKey,
				"timer_type", timerType)
			return fmt.Errorf("unknown timer type: %s", timerType)
		}
	})

	m.logger.Info("registered timer expiration callbacks")
	return nil
}
