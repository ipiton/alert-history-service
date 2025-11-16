package filters

import (
	"fmt"

	"github.com/vitaliisemenov/alert-history/pkg/history/query"
)

// AlertNamePatternFilter filters alerts by alert name pattern (LIKE)
type AlertNamePatternFilter struct {
	pattern string
}

// NewAlertNamePatternFilter creates a new alert name pattern filter
func NewAlertNamePatternFilter(params map[string]interface{}) (Filter, error) {
	pattern, ok := params["pattern"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid alert_name_pattern filter params: expected string")
	}

	if pattern == "" {
		return nil, fmt.Errorf("alert_name_pattern filter requires non-empty pattern")
	}

	if len(pattern) > 255 {
		return nil, fmt.Errorf("alert_name_pattern too long: max 255 characters")
	}

	return &AlertNamePatternFilter{pattern: pattern}, nil
}

func (f *AlertNamePatternFilter) Type() FilterType {
	return FilterTypeAlertNamePattern
}

func (f *AlertNamePatternFilter) Validate() error {
	if f.pattern == "" {
		return fmt.Errorf("alert_name_pattern filter requires non-empty pattern")
	}
	if len(f.pattern) > 255 {
		return fmt.Errorf("alert_name_pattern too long: max 255 characters")
	}
	return nil
}

func (f *AlertNamePatternFilter) ApplyToQuery(qb *query.Builder) error {
	// LIKE pattern matching (case-sensitive)
	// Uses B-tree index on alert_name (PostgreSQL can use index for prefix patterns)
	qb.AddWhere("alert_name LIKE ?", f.pattern)
	return nil
}

func (f *AlertNamePatternFilter) CacheKey() string {
	return fmt.Sprintf("alert_name_pattern:%s", f.pattern)
}
