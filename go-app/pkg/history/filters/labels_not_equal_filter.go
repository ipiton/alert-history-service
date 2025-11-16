package filters

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vitaliisemenov/alert-history/pkg/history/query"
)

// LabelsNotEqualFilter filters alerts by label key-value not equal (!=)
type LabelsNotEqualFilter struct {
	labels map[string]string
}

// NewLabelsNotEqualFilter creates a new labels not equal filter
func NewLabelsNotEqualFilter(params map[string]interface{}) (Filter, error) {
	labels, ok := params["labels"].(map[string]string)
	if !ok {
		return nil, fmt.Errorf("invalid labels_ne filter params: expected map[string]string")
	}

	if len(labels) == 0 {
		return nil, fmt.Errorf("labels_ne filter requires at least one label")
	}

	if len(labels) > 20 {
		return nil, fmt.Errorf("too many labels: max 20, got %d", len(labels))
	}

	// Validate label keys and values
	for key, value := range labels {
		if len(key) == 0 {
			return nil, fmt.Errorf("label key cannot be empty")
		}
		if len(key) > 255 {
			return nil, fmt.Errorf("label key too long: max 255 characters")
		}
		if len(value) > 255 {
			return nil, fmt.Errorf("label value too long: max 255 characters")
		}
	}

	return &LabelsNotEqualFilter{labels: labels}, nil
}

func (f *LabelsNotEqualFilter) Type() FilterType {
	return FilterTypeLabelsNotEqual
}

func (f *LabelsNotEqualFilter) Validate() error {
	if len(f.labels) == 0 {
		return fmt.Errorf("labels_ne filter requires at least one label")
	}
	if len(f.labels) > 20 {
		return fmt.Errorf("too many labels: max 20")
	}

	for key, value := range f.labels {
		if len(key) == 0 {
			return fmt.Errorf("label key cannot be empty")
		}
		if len(key) > 255 {
			return fmt.Errorf("label key too long: max 255 characters")
		}
		if len(value) > 255 {
			return fmt.Errorf("label value too long: max 255 characters")
		}
	}

	return nil
}

func (f *LabelsNotEqualFilter) ApplyToQuery(qb *query.Builder) error {
	// Mark for GIN index usage (labels JSONB field)
	qb.MarkGINIndexUsage()

	// Use NOT JSONB containment for not equal
	// Multiple labels use AND logic (all must not match)
	for key, value := range f.labels {
		qb.AddWhere("NOT (labels @> jsonb_build_object(?, ?))", key, value)
	}

	return nil
}

func (f *LabelsNotEqualFilter) CacheKey() string {
	// Sort keys for consistent cache keys
	keys := make([]string, 0, len(f.labels))
	for k := range f.labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s!=%s", k, f.labels[k]))
	}
	return fmt.Sprintf("labels_ne:%s", strings.Join(parts, ","))
}
