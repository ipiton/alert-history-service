package filters

import (
	"fmt"
	"strconv"
	
	"github.com/vitaliisemenov/alert-history/pkg/history/query"
)

// IsFlappingFilter filters alerts by flapping status
// Note: This requires flapping score calculation, which may need to be computed
// For now, this is a placeholder that filters alerts with high transition count
type IsFlappingFilter struct {
	value bool
}

// NewIsFlappingFilter creates a new is_flapping filter
func NewIsFlappingFilter(params map[string]interface{}) (Filter, error) {
	valueStr, ok := params["value"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid is_flapping filter params: expected string")
	}
	
	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return nil, fmt.Errorf("invalid is_flapping value: %s (must be 'true' or 'false')", valueStr)
	}
	
	return &IsFlappingFilter{value: value}, nil
}

func (f *IsFlappingFilter) Type() FilterType {
	return FilterTypeIsFlapping
}

func (f *IsFlappingFilter) Validate() error {
	// Always valid (boolean value)
	return nil
}

func (f *IsFlappingFilter) ApplyToQuery(qb *query.Builder) error {
	// TODO: Implement flapping detection logic
	// For now, this is a placeholder
	// Flapping detection requires:
	// 1. Counting state transitions per fingerprint
	// 2. Calculating flapping score
	// 3. Filtering by threshold
	
	// Placeholder: This would need a subquery or CTE to calculate flapping
	// For now, return no-op (will be implemented when flapping detection is ready)
	// qb.AddWhere("... flapping detection logic ...")
	
	// Note: This filter may need to be implemented at the application level
	// rather than SQL level, depending on flapping calculation complexity
	return nil
}

func (f *IsFlappingFilter) CacheKey() string {
	return fmt.Sprintf("is_flapping:%v", f.value)
}

