package filters

import (
	"fmt"
	"sort"
	"strings"
	
	"github.com/vitaliisemenov/alert-history/pkg/history/query"
)

// SeverityFilter filters alerts by severity (critical, warning, info, noise)
type SeverityFilter struct {
	values []string
}

// NewSeverityFilter creates a new severity filter
func NewSeverityFilter(params map[string]interface{}) (Filter, error) {
	values, ok := params["values"].([]string)
	if !ok {
		return nil, fmt.Errorf("invalid severity filter params: expected []string")
	}
	
	if len(values) == 0 {
		return nil, fmt.Errorf("severity filter requires at least one value")
	}
	
	filter := &SeverityFilter{
		values: make([]string, 0, len(values)),
	}
	
	validSeverities := map[string]bool{
		"critical": true,
		"warning":  true,
		"info":     true,
		"noise":    true,
	}
	
	for _, v := range values {
		if !validSeverities[v] {
			return nil, fmt.Errorf("invalid severity: %s (must be one of: critical, warning, info, noise)", v)
		}
		filter.values = append(filter.values, v)
	}
	
	return filter, nil
}

func (f *SeverityFilter) Type() FilterType {
	return FilterTypeSeverity
}

func (f *SeverityFilter) Validate() error {
	if len(f.values) == 0 {
		return fmt.Errorf("severity filter requires at least one value")
	}
	
	validSeverities := map[string]bool{
		"critical": true,
		"warning":  true,
		"info":     true,
		"noise":    true,
	}
	
	for _, v := range f.values {
		if !validSeverities[v] {
			return fmt.Errorf("invalid severity: %s", v)
		}
	}
	
	return nil
}

func (f *SeverityFilter) ApplyToQuery(qb *query.Builder) error {
	// Mark for GIN index usage (labels JSONB field)
	qb.MarkGINIndexUsage()
	
	if len(f.values) == 1 {
		// Single value: use equality
		qb.AddWhere("labels->>'severity' = ?", f.values[0])
	} else {
		// Multiple values: use IN operator
		placeholders := make([]string, len(f.values))
		args := make([]interface{}, len(f.values))
		for i, v := range f.values {
			placeholders[i] = "?"
			args[i] = v
		}
		qb.AddWhere(fmt.Sprintf("labels->>'severity' IN (%s)", strings.Join(placeholders, ",")), args...)
	}
	return nil
}

func (f *SeverityFilter) CacheKey() string {
	values := make([]string, len(f.values))
	copy(values, f.values)
	sort.Strings(values) // Sort for consistent cache keys
	return fmt.Sprintf("severity:%s", strings.Join(values, ","))
}

