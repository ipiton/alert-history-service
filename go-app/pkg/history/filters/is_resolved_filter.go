package filters

import (
	"fmt"
	"strconv"
	
	"github.com/vitaliisemenov/alert-history/pkg/history/query"
)

// IsResolvedFilter filters alerts by resolved status (ends_at != null)
type IsResolvedFilter struct {
	value bool
}

// NewIsResolvedFilter creates a new is_resolved filter
func NewIsResolvedFilter(params map[string]interface{}) (Filter, error) {
	valueStr, ok := params["value"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid is_resolved filter params: expected string")
	}
	
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return nil, fmt.Errorf("invalid is_resolved value: %s (must be 'true' or 'false')", valueStr)
	}
	
	return &IsResolvedFilter{value: value}, nil
}

func (f *IsResolvedFilter) Type() FilterType {
	return FilterTypeIsResolved
}

func (f *IsResolvedFilter) Validate() error {
	// Always valid (boolean value)
	return nil
}

func (f *IsResolvedFilter) ApplyToQuery(qb *query.Builder) error {
	if f.value {
		// Filter for resolved alerts (ends_at IS NOT NULL)
		qb.AddWhere("ends_at IS NOT NULL")
	} else {
		// Filter for unresolved alerts (ends_at IS NULL)
		qb.AddWhere("ends_at IS NULL")
	}
	return nil
}

func (f *IsResolvedFilter) CacheKey() string {
	return fmt.Sprintf("is_resolved:%v", f.value)
}

