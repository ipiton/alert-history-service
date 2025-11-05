package inhibition

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/infrastructure/cache"
)

// InhibitionState represents the state of an inhibition relationship between alerts.
// It tracks which target alert is being inhibited by which source alert and under which rule.
type InhibitionState struct {
	// TargetFingerprint is the fingerprint of the inhibited (suppressed) alert
	TargetFingerprint string `json:"target_fingerprint"`

	// SourceFingerprint is the fingerprint of the source alert causing the inhibition
	SourceFingerprint string `json:"source_fingerprint"`

	// RuleName is the name of the inhibition rule that caused this inhibition
	RuleName string `json:"rule_name"`

	// InhibitedAt is when the inhibition relationship was established
	InhibitedAt time.Time `json:"inhibited_at"`

	// ExpiresAt is when this inhibition expires (optional)
	// If nil, the inhibition lasts until the source alert resolves
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// InhibitionStateManager manages the state of active inhibition relationships.
// It tracks which alerts are currently inhibited and by whom, providing both
// in-memory access and Redis persistence for high availability.
type InhibitionStateManager interface {
	// RecordInhibition records a new inhibition relationship.
	// This is called when an alert is determined to be inhibited by another alert.
	RecordInhibition(ctx context.Context, state *InhibitionState) error

	// RemoveInhibition removes an inhibition relationship.
	// This is called when the source alert resolves or the inhibition expires.
	RemoveInhibition(ctx context.Context, targetFingerprint string) error

	// GetActiveInhibitions returns all currently active inhibition relationships.
	GetActiveInhibitions(ctx context.Context) ([]*InhibitionState, error)

	// GetInhibitedAlerts returns a list of all currently inhibited alert fingerprints.
	// This is a convenience method for quick lookups.
	GetInhibitedAlerts(ctx context.Context) ([]string, error)

	// IsInhibited checks if a specific alert is currently inhibited.
	IsInhibited(ctx context.Context, targetFingerprint string) (bool, error)

	// GetInhibitionState retrieves the inhibition state for a specific target alert.
	// Returns nil if the alert is not inhibited.
	GetInhibitionState(ctx context.Context, targetFingerprint string) (*InhibitionState, error)
}

// DefaultStateManager implements InhibitionStateManager with in-memory storage
// and optional Redis persistence for high availability.
type DefaultStateManager struct {
	// In-memory state (fingerprint -> InhibitionState)
	states sync.Map

	// Redis store for persistence (optional)
	redisStore cache.Cache

	// Redis key prefix
	redisPrefix string

	// Redis TTL for state entries
	redisTTL time.Duration

	// Logger
	logger *slog.Logger

	// Metrics would go here (already added to pkg/metrics/business.go)
}

// NewDefaultStateManager creates a new DefaultStateManager instance.
//
// Parameters:
//   - redisStore: Optional Redis cache for persistence. If nil, only in-memory storage is used.
//   - logger: Logger instance for debug/error logging.
//
// Returns:
//   - *DefaultStateManager: Initialized state manager.
func NewDefaultStateManager(redisStore cache.Cache, logger *slog.Logger) *DefaultStateManager {
	if logger == nil {
		logger = slog.Default()
	}

	return &DefaultStateManager{
		states:      sync.Map{},
		redisStore:  redisStore,
		redisPrefix: "inhibition:state:",
		redisTTL:    24 * time.Hour,
		logger:      logger,
	}
}

// RecordInhibition records a new inhibition relationship in both memory and Redis.
func (sm *DefaultStateManager) RecordInhibition(ctx context.Context, state *InhibitionState) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}

	if state.TargetFingerprint == "" {
		return fmt.Errorf("target fingerprint cannot be empty")
	}

	if state.SourceFingerprint == "" {
		return fmt.Errorf("source fingerprint cannot be empty")
	}

	// Store in memory
	sm.states.Store(state.TargetFingerprint, state)

	sm.logger.Debug("Recorded inhibition",
		"target", state.TargetFingerprint,
		"source", state.SourceFingerprint,
		"rule", state.RuleName,
	)

	// Persist to Redis if available
	if sm.redisStore != nil {
		if err := sm.persistToRedis(ctx, state); err != nil {
			sm.logger.Warn("Failed to persist inhibition state to Redis",
				"error", err,
				"target", state.TargetFingerprint,
			)
			// Non-critical: in-memory state is still valid
		}
	}

	return nil
}

// RemoveInhibition removes an inhibition relationship from both memory and Redis.
func (sm *DefaultStateManager) RemoveInhibition(ctx context.Context, targetFingerprint string) error {
	if targetFingerprint == "" {
		return fmt.Errorf("target fingerprint cannot be empty")
	}

	// Remove from memory
	sm.states.Delete(targetFingerprint)

	sm.logger.Debug("Removed inhibition",
		"target", targetFingerprint,
	)

	// Remove from Redis if available
	if sm.redisStore != nil {
		key := sm.redisPrefix + targetFingerprint
		if err := sm.redisStore.Delete(ctx, key); err != nil {
			sm.logger.Warn("Failed to remove inhibition state from Redis",
				"error", err,
				"target", targetFingerprint,
			)
			// Non-critical: in-memory state is already removed
		}
	}

	return nil
}

// GetActiveInhibitions returns all currently active inhibition relationships.
func (sm *DefaultStateManager) GetActiveInhibitions(ctx context.Context) ([]*InhibitionState, error) {
	states := make([]*InhibitionState, 0)

	sm.states.Range(func(key, value interface{}) bool {
		if state, ok := value.(*InhibitionState); ok {
			// Filter out expired inhibitions
			if state.ExpiresAt == nil || time.Now().Before(*state.ExpiresAt) {
				states = append(states, state)
			} else {
				// Clean up expired state
				sm.states.Delete(key)
			}
		}
		return true
	})

	return states, nil
}

// GetInhibitedAlerts returns a list of all currently inhibited alert fingerprints.
func (sm *DefaultStateManager) GetInhibitedAlerts(ctx context.Context) ([]string, error) {
	fingerprints := make([]string, 0)

	sm.states.Range(func(key, value interface{}) bool {
		if fingerprint, ok := key.(string); ok {
			if state, ok := value.(*InhibitionState); ok {
				// Filter out expired inhibitions
				if state.ExpiresAt == nil || time.Now().Before(*state.ExpiresAt) {
					fingerprints = append(fingerprints, fingerprint)
				}
			}
		}
		return true
	})

	return fingerprints, nil
}

// IsInhibited checks if a specific alert is currently inhibited.
func (sm *DefaultStateManager) IsInhibited(ctx context.Context, targetFingerprint string) (bool, error) {
	if targetFingerprint == "" {
		return false, fmt.Errorf("target fingerprint cannot be empty")
	}

	value, ok := sm.states.Load(targetFingerprint)
	if !ok {
		return false, nil
	}

	state, ok := value.(*InhibitionState)
	if !ok {
		return false, nil
	}

	// Check if expired
	if state.ExpiresAt != nil && time.Now().After(*state.ExpiresAt) {
		// Clean up expired state
		sm.states.Delete(targetFingerprint)
		return false, nil
	}

	return true, nil
}

// GetInhibitionState retrieves the inhibition state for a specific target alert.
func (sm *DefaultStateManager) GetInhibitionState(ctx context.Context, targetFingerprint string) (*InhibitionState, error) {
	if targetFingerprint == "" {
		return nil, fmt.Errorf("target fingerprint cannot be empty")
	}

	value, ok := sm.states.Load(targetFingerprint)
	if !ok {
		// Try Redis fallback if available
		if sm.redisStore != nil {
			state, err := sm.loadFromRedis(ctx, targetFingerprint)
			if err != nil {
				return nil, nil // Not found
			}
			// Repopulate memory cache
			sm.states.Store(targetFingerprint, state)
			return state, nil
		}
		return nil, nil
	}

	state, ok := value.(*InhibitionState)
	if !ok {
		return nil, nil
	}

	// Check if expired
	if state.ExpiresAt != nil && time.Now().After(*state.ExpiresAt) {
		// Clean up expired state
		sm.states.Delete(targetFingerprint)
		return nil, nil
	}

	return state, nil
}

// persistToRedis persists an inhibition state to Redis.
func (sm *DefaultStateManager) persistToRedis(ctx context.Context, state *InhibitionState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	key := sm.redisPrefix + state.TargetFingerprint
	if err := sm.redisStore.Set(ctx, key, string(data), sm.redisTTL); err != nil {
		return fmt.Errorf("failed to set Redis key: %w", err)
	}

	return nil
}

// loadFromRedis loads an inhibition state from Redis.
func (sm *DefaultStateManager) loadFromRedis(ctx context.Context, targetFingerprint string) (*InhibitionState, error) {
	key := sm.redisPrefix + targetFingerprint

	var data string
	if err := sm.redisStore.Get(ctx, key, &data); err != nil {
		return nil, fmt.Errorf("failed to get Redis key: %w", err)
	}

	var state InhibitionState
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	return &state, nil
}
