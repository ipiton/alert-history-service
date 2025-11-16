package filters

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vitaliisemenov/alert-history/pkg/history/query"
)

// FingerprintFilter filters alerts by fingerprint (SHA-256 hash)
type FingerprintFilter struct {
	values []string
}

// NewFingerprintFilter creates a new fingerprint filter
func NewFingerprintFilter(params map[string]interface{}) (Filter, error) {
	values, ok := params["values"].([]string)
	if !ok {
		return nil, fmt.Errorf("invalid fingerprint filter params: expected []string")
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("fingerprint filter requires at least one value")
	}

	filter := &FingerprintFilter{
		values: make([]string, 0, len(values)),
	}

	for _, v := range values {
		// Validate fingerprint format (64 hex characters)
		if len(v) != 64 {
			return nil, fmt.Errorf("invalid fingerprint format: %s (must be 64 hex characters)", v)
		}
		// Basic hex validation
		for _, c := range v {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return nil, fmt.Errorf("invalid fingerprint format: %s (must contain only hex characters)", v)
			}
		}
		filter.values = append(filter.values, v)
	}

	return filter, nil
}

func (f *FingerprintFilter) Type() FilterType {
	return FilterTypeFingerprint
}

func (f *FingerprintFilter) Validate() error {
	if len(f.values) == 0 {
		return fmt.Errorf("fingerprint filter requires at least one value")
	}

	for _, v := range f.values {
		if len(v) != 64 {
			return fmt.Errorf("invalid fingerprint format: %s (must be 64 hex characters)", v)
		}
	}

	return nil
}

func (f *FingerprintFilter) ApplyToQuery(qb *query.Builder) error {
	// Fingerprint uses indexed column, no special index hints needed
	if len(f.values) == 1 {
		// Single value: use equality
		qb.AddWhere("fingerprint = ?", f.values[0])
	} else {
		// Multiple values: use IN operator
		placeholders := make([]string, len(f.values))
		args := make([]interface{}, len(f.values))
		for i, v := range f.values {
			placeholders[i] = "?"
			args[i] = v
		}
		qb.AddWhere(fmt.Sprintf("fingerprint IN (%s)", strings.Join(placeholders, ",")), args...)
	}
	return nil
}

func (f *FingerprintFilter) CacheKey() string {
	values := make([]string, len(f.values))
	copy(values, f.values)
	sort.Strings(values) // Sort for consistent cache keys
	return fmt.Sprintf("fingerprint:%s", strings.Join(values, ","))
}
