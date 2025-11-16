package filters

import (
	"fmt"
	"sort"
	"strings"
	
	"github.com/vitaliisemenov/alert-history/pkg/history/query"
)

// LabelsExactFilter filters alerts by exact label key-value match (=)
type LabelsExactFilter struct {
	labels map[string]string
}

// NewLabelsExactFilter creates a new labels exact match filter
func NewLabelsExactFilter(params map[string]interface{}) (Filter, error) {
	labels, ok := params["labels"].(map[string]string)
	if !ok {
		return nil, fmt.Errorf("invalid labels_exact filter params: expected map[string]string")
	}
	
	if len(labels) == 0 {
		return nil, fmt.Errorf("labels_exact filter requires at least one label")
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
			return nil, fmt.Errorf("label key too long: max 255 characters, got %d", len(key))
		}
		if len(value) > 255 {
			return nil, fmt.Errorf("label value too long: max 255 characters, got %d", len(value))
		}
	}
	
	return &LabelsExactFilter{labels: labels}, nil
}

func (f *LabelsExactFilter) Type() FilterType {
	return FilterTypeLabelsExact
}

func (f *LabelsExactFilter) Validate() error {
	if len(f.labels) == 0 {
		return fmt.Errorf("labels_exact filter requires at least one label")
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

func (f *LabelsExactFilter) ApplyToQuery(qb *query.Builder) error {
	// Mark for GIN index usage (labels JSONB field)
	qb.MarkGINIndexUsage()
	
	// Use JSONB containment operator (@>) for exact match
	// Multiple labels use AND logic (all must match)
	for key, value := range f.labels {
		// Build JSONB object: {"key": "value"}
		qb.AddWhere("labels @> jsonb_build_object(?, ?)", key, value)
	}
	
	return nil
}

func (f *LabelsExactFilter) CacheKey() string {
	// Sort keys for consistent cache keys
	keys := make([]string, 0, len(f.labels))
	for k := range f.labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	
	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, f.labels[k]))
	}
	return fmt.Sprintf("labels_exact:%s", strings.Join(parts, ","))
}

