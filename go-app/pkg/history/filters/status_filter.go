package filters

import (
	"fmt"
	"sort"
	"strings"
	
	"github.com/vitaliisemenov/alert-history/internal/core"
	"github.com/vitaliisemenov/alert-history/pkg/history/query"
)

// StatusFilter filters alerts by status (firing, resolved)
type StatusFilter struct {
	values []core.AlertStatus
}

// NewStatusFilter creates a new status filter
func NewStatusFilter(params map[string]interface{}) (Filter, error) {
	values, ok := params["values"].([]string)
	if !ok {
		return nil, fmt.Errorf("invalid status filter params: expected []string")
	}
	
	if len(values) == 0 {
		return nil, fmt.Errorf("status filter requires at least one value")
	}
	
	filter := &StatusFilter{
		values: make([]core.AlertStatus, 0, len(values)),
	}
	
	for _, v := range values {
		status := core.AlertStatus(v)
		if status != core.StatusFiring && status != core.StatusResolved {
			return nil, fmt.Errorf("invalid status: %s (must be 'firing' or 'resolved')", v)
		}
		filter.values = append(filter.values, status)
	}
	
	return filter, nil
}

func (f *StatusFilter) Type() FilterType {
	return FilterTypeStatus
}

func (f *StatusFilter) Validate() error {
	if len(f.values) == 0 {
		return fmt.Errorf("status filter requires at least one value")
	}
	
	for _, v := range f.values {
		if v != core.StatusFiring && v != core.StatusResolved {
			return fmt.Errorf("invalid status: %s", v)
		}
	}
	
	return nil
}

func (f *StatusFilter) ApplyToQuery(qb *query.Builder) error {
	if len(f.values) == 1 {
		// Single value: use equality
		qb.AddWhere("status = ?", f.values[0])
		// If filtering by firing, mark for partial index usage
		if f.values[0] == core.StatusFiring {
			qb.MarkPartialIndexUsage()
		}
	} else {
		// Multiple values: use IN operator
		placeholders := make([]string, len(f.values))
		args := make([]interface{}, len(f.values))
		for i, v := range f.values {
			placeholders[i] = "?"
			args[i] = v
		}
		qb.AddWhere(fmt.Sprintf("status IN (%s)", strings.Join(placeholders, ",")), args...)
	}
	return nil
}

func (f *StatusFilter) CacheKey() string {
	values := make([]string, len(f.values))
	for i, v := range f.values {
		values[i] = string(v)
	}
	sort.Strings(values) // Sort for consistent cache keys
	return fmt.Sprintf("status:%s", strings.Join(values, ","))
}

