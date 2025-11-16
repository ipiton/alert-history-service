package filters

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vitaliisemenov/alert-history/pkg/history/query"
)

// NamespaceFilter filters alerts by namespace
type NamespaceFilter struct {
	values []string
}

// NewNamespaceFilter creates a new namespace filter
func NewNamespaceFilter(params map[string]interface{}) (Filter, error) {
	values, ok := params["values"].([]string)
	if !ok {
		return nil, fmt.Errorf("invalid namespace filter params: expected []string")
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("namespace filter requires at least one value")
	}

	filter := &NamespaceFilter{
		values: make([]string, 0, len(values)),
	}

	for _, v := range values {
		if v == "" {
			continue // Skip empty values
		}
		if len(v) > 255 {
			return nil, fmt.Errorf("namespace too long: max 255 characters")
		}
		filter.values = append(filter.values, v)
	}

	if len(filter.values) == 0 {
		return nil, fmt.Errorf("namespace filter requires at least one non-empty value")
	}

	return filter, nil
}

func (f *NamespaceFilter) Type() FilterType {
	return FilterTypeNamespace
}

func (f *NamespaceFilter) Validate() error {
	if len(f.values) == 0 {
		return fmt.Errorf("namespace filter requires at least one value")
	}

	for _, v := range f.values {
		if len(v) > 255 {
			return fmt.Errorf("namespace too long: max 255 characters")
		}
	}

	return nil
}

func (f *NamespaceFilter) ApplyToQuery(qb *query.Builder) error {
	// Mark for GIN index usage (labels JSONB field)
	qb.MarkGINIndexUsage()

	if len(f.values) == 1 {
		// Single value: use equality
		qb.AddWhere("labels->>'namespace' = ?", f.values[0])
	} else {
		// Multiple values: use IN operator
		placeholders := make([]string, len(f.values))
		args := make([]interface{}, len(f.values))
		for i, v := range f.values {
			placeholders[i] = "?"
			args[i] = v
		}
		qb.AddWhere(fmt.Sprintf("labels->>'namespace' IN (%s)", strings.Join(placeholders, ",")), args...)
	}
	return nil
}

func (f *NamespaceFilter) CacheKey() string {
	values := make([]string, len(f.values))
	copy(values, f.values)
	sort.Strings(values) // Sort for consistent cache keys
	return fmt.Sprintf("namespace:%s", strings.Join(values, ","))
}
