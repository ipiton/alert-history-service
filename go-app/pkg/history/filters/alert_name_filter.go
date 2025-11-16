package filters

import (
	"fmt"

	"github.com/vitaliisemenov/alert-history/pkg/history/query"
)

// AlertNameFilter filters alerts by exact alert name match
type AlertNameFilter struct {
	value string
}

// NewAlertNameFilter creates a new alert name filter
func NewAlertNameFilter(params map[string]interface{}) (Filter, error) {
	value, ok := params["value"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid alert_name filter params: expected string")
	}

	if value == "" {
		return nil, fmt.Errorf("alert_name filter requires non-empty value")
	}

	if len(value) > 255 {
		return nil, fmt.Errorf("alert_name too long: max 255 characters")
	}

	return &AlertNameFilter{value: value}, nil
}

func (f *AlertNameFilter) Type() FilterType {
	return FilterTypeAlertName
}

func (f *AlertNameFilter) Validate() error {
	if f.value == "" {
		return fmt.Errorf("alert_name filter requires non-empty value")
	}
	if len(f.value) > 255 {
		return fmt.Errorf("alert_name too long: max 255 characters")
	}
	return nil
}

func (f *AlertNameFilter) ApplyToQuery(qb *query.Builder) error {
	// Alert name uses indexed column (idx_alerts_alert_name)
	qb.AddWhere("alert_name = ?", f.value)
	return nil
}

func (f *AlertNameFilter) CacheKey() string {
	return fmt.Sprintf("alert_name:%s", f.value)
}
