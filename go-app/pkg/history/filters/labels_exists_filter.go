package filters

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vitaliisemenov/alert-history/pkg/history/query"
)

// LabelsExistsFilter filters alerts by label key existence
type LabelsExistsFilter struct {
	keys []string
}

// NewLabelsExistsFilter creates a new labels exists filter
func NewLabelsExistsFilter(params map[string]interface{}) (Filter, error) {
	keys, ok := params["keys"].([]string)
	if !ok {
		return nil, fmt.Errorf("invalid labels_exists filter params: expected []string")
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("labels_exists filter requires at least one key")
	}

	if len(keys) > 20 {
		return nil, fmt.Errorf("too many label keys: max 20, got %d", len(keys))
	}

	// Validate keys
	for _, key := range keys {
		if len(key) == 0 {
			return nil, fmt.Errorf("label key cannot be empty")
		}
		if len(key) > 255 {
			return nil, fmt.Errorf("label key too long: max 255 characters")
		}
	}

	return &LabelsExistsFilter{keys: keys}, nil
}

func (f *LabelsExistsFilter) Type() FilterType {
	return FilterTypeLabelsExists
}

func (f *LabelsExistsFilter) Validate() error {
	if len(f.keys) == 0 {
		return fmt.Errorf("labels_exists filter requires at least one key")
	}
	if len(f.keys) > 20 {
		return fmt.Errorf("too many label keys: max 20")
	}

	for _, key := range f.keys {
		if len(key) == 0 {
			return fmt.Errorf("label key cannot be empty")
		}
		if len(key) > 255 {
			return fmt.Errorf("label key too long: max 255 characters")
		}
	}

	return nil
}

func (f *LabelsExistsFilter) ApplyToQuery(qb *query.Builder) error {
	// Mark for GIN index usage (labels JSONB field)
	qb.MarkGINIndexUsage()

	// Use JSONB key existence operator (?)
	// Multiple keys use AND logic (all must exist)
	for _, key := range f.keys {
		qb.AddWhere("labels ? ?", key)
	}

	return nil
}

func (f *LabelsExistsFilter) CacheKey() string {
	keys := make([]string, len(f.keys))
	copy(keys, f.keys)
	sort.Strings(keys) // Sort for consistent cache keys
	return fmt.Sprintf("labels_exists:%s", strings.Join(keys, ","))
}
