package inhibition

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/core"
)

// DefaultInhibitionMatcher is the standard implementation of InhibitionMatcher.
//
// Thread-safety: Safe for concurrent use (all operations are read-only or use thread-safe cache).
// Performance: <1ms per inhibition check (p99), <10µs per rule matching.
//
// Example:
//
//	matcher := inhibition.NewMatcher(cache, rules, logger)
//	result, err := matcher.ShouldInhibit(ctx, targetAlert)
type DefaultInhibitionMatcher struct {
	cache  ActiveAlertCache
	rules  []InhibitionRule
	logger *slog.Logger
}

// NewMatcher creates a new InhibitionMatcher with the given configuration.
//
// Parameters:
//   - cache: cache for accessing firing alerts
//   - rules: list of inhibition rules to evaluate
//   - logger: structured logger for debugging
//
// Returns:
//   - *DefaultInhibitionMatcher: initialized matcher ready to use
//
// Example:
//
//	rules := []InhibitionRule{...}
//	matcher := inhibition.NewMatcher(cache, rules, logger)
func NewMatcher(cache ActiveAlertCache, rules []InhibitionRule, logger *slog.Logger) *DefaultInhibitionMatcher {
	if logger == nil {
		logger = slog.Default()
	}

	return &DefaultInhibitionMatcher{
		cache:  cache,
		rules:  rules,
		logger: logger,
	}
}

// ShouldInhibit implements InhibitionMatcher.ShouldInhibit.
//
// Returns the FIRST matching inhibition (early return optimization).
// For all matches, use FindInhibitors.
func (m *DefaultInhibitionMatcher) ShouldInhibit(
	ctx context.Context,
	targetAlert *core.Alert,
) (*MatchResult, error) {
	startTime := time.Now()

	// Get all firing alerts from cache
	firingAlerts, err := m.cache.GetFiringAlerts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get firing alerts: %w", err)
	}

	m.logger.Debug("Checking inhibition",
		"target_alert", targetAlert.Fingerprint,
		"firing_alerts_count", len(firingAlerts),
		"rules_count", len(m.rules))

	// Check each rule
	for i := range m.rules {
		rule := &m.rules[i]

		// Check each firing alert as potential source
		for _, sourceAlert := range firingAlerts {
			// Skip self-inhibition
			if sourceAlert.Fingerprint == targetAlert.Fingerprint {
				continue
			}

			// Check if rule matches
			if m.MatchRule(rule, sourceAlert, targetAlert) {
				duration := time.Since(startTime)

				m.logger.Info("Alert inhibited",
					"target", targetAlert.Fingerprint,
					"source", sourceAlert.Fingerprint,
					"rule", rule.Name,
					"duration", duration)

				return &MatchResult{
					Matched:       true,
					InhibitedBy:   sourceAlert,
					Rule:          rule,
					MatchDuration: duration,
				}, nil
			}
		}
	}

	// No match found
	duration := time.Since(startTime)

	m.logger.Debug("Alert not inhibited",
		"target", targetAlert.Fingerprint,
		"duration", duration)

	return &MatchResult{
		Matched:       false,
		MatchDuration: duration,
	}, nil
}

// FindInhibitors implements InhibitionMatcher.FindInhibitors.
//
// Returns ALL matching inhibitions (no early return).
func (m *DefaultInhibitionMatcher) FindInhibitors(
	ctx context.Context,
	targetAlert *core.Alert,
) ([]*MatchResult, error) {
	startTime := time.Now()

	// Get all firing alerts from cache
	firingAlerts, err := m.cache.GetFiringAlerts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get firing alerts: %w", err)
	}

	var results []*MatchResult

	// Check each rule
	for i := range m.rules {
		rule := &m.rules[i]

		// Check each firing alert as potential source
		for _, sourceAlert := range firingAlerts {
			// Skip self-inhibition
			if sourceAlert.Fingerprint == targetAlert.Fingerprint {
				continue
			}

			// Check if rule matches
			if m.MatchRule(rule, sourceAlert, targetAlert) {
				results = append(results, &MatchResult{
					Matched:       true,
					InhibitedBy:   sourceAlert,
					Rule:          rule,
					MatchDuration: time.Since(startTime),
				})
			}
		}
	}

	m.logger.Debug("Find inhibitors complete",
		"target", targetAlert.Fingerprint,
		"inhibitors_found", len(results),
		"duration", time.Since(startTime))

	return results, nil
}

// MatchRule implements InhibitionMatcher.MatchRule.
//
// Core matching logic (pure function, no I/O):
//  1. Check source_match (exact label matching)
//  2. Check source_match_re (regex label matching)
//  3. Check target_match (exact label matching)
//  4. Check target_match_re (regex label matching)
//  5. Check equal labels (must have same value in both alerts)
//
// All conditions must match (AND logic).
//
// Performance: <10µs per call (no allocations).
func (m *DefaultInhibitionMatcher) MatchRule(
	rule *InhibitionRule,
	sourceAlert, targetAlert *core.Alert,
) bool {
	// 1. Check source_match conditions (exact matching)
	if !matchLabels(sourceAlert.Labels, rule.SourceMatch) {
		return false
	}

	// 2. Check source_match_re conditions (regex matching)
	if !matchLabelsRE(sourceAlert.Labels, rule.SourceMatchRE, rule.compiledSourceRE) {
		return false
	}

	// 3. Check target_match conditions (exact matching)
	if !matchLabels(targetAlert.Labels, rule.TargetMatch) {
		return false
	}

	// 4. Check target_match_re conditions (regex matching)
	if !matchLabelsRE(targetAlert.Labels, rule.TargetMatchRE, rule.compiledTargetRE) {
		return false
	}

	// 5. Check equal labels (must match between source and target)
	for _, labelName := range rule.Equal {
		sourceVal, sourceOk := sourceAlert.Labels[labelName]
		targetVal, targetOk := targetAlert.Labels[labelName]

		// If label missing in either alert → no match
		if !sourceOk || !targetOk {
			return false
		}

		// If label values differ → no match
		if sourceVal != targetVal {
			return false
		}
	}

	// All conditions matched
	return true
}

// --- Helper functions (not exported) ---

// matchLabels checks if alert labels match the required label matchers (exact match).
//
// Parameters:
//   - alertLabels: labels from the alert
//   - matchers: required label matchers (key=value)
//
// Returns:
//   - bool: true if ALL matchers match
//
// Empty matchers returns true (no conditions to check).
func matchLabels(alertLabels map[string]string, matchers map[string]string) bool {
	// Empty matchers → always match
	if len(matchers) == 0 {
		return true
	}

	// Check each matcher
	for key, requiredValue := range matchers {
		actualValue, exists := alertLabels[key]

		// Label missing → no match
		if !exists {
			return false
		}

		// Value doesn't match → no match
		if actualValue != requiredValue {
			return false
		}
	}

	// All matchers matched
	return true
}

// matchLabelsRE checks if alert labels match the required label matchers (regex match).
//
// Parameters:
//   - alertLabels: labels from the alert
//   - matchers: required regex matchers (key=pattern)
//   - compiledRE: pre-compiled regex patterns (for performance)
//
// Returns:
//   - bool: true if ALL regex matchers match
//
// Empty matchers returns true (no conditions to check).
func matchLabelsRE(alertLabels map[string]string, matchers map[string]string, compiledRE map[string]*regexp.Regexp) bool {
	// Empty matchers → always match
	if len(matchers) == 0 {
		return true
	}

	// Check each regex matcher
	for key := range matchers {
		actualValue, exists := alertLabels[key]

		// Label missing → no match
		if !exists {
			return false
		}

		// Get pre-compiled regex
		re, hasRE := compiledRE[key]
		if !hasRE {
			// Regex not compiled (shouldn't happen if parser did its job)
			return false
		}

		// Check regex match
		if !re.MatchString(actualValue) {
			return false
		}
	}

	// All regex matchers matched
	return true
}
