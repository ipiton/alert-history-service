package filters

import (
	"fmt"
	"strings"
	"time"

	"github.com/vitaliisemenov/alert-history/pkg/history/query"
)

// TimeRangeFilter filters alerts by time range (from/to timestamps)
type TimeRangeFilter struct {
	from *time.Time
	to   *time.Time
}

// NewTimeRangeFilter creates a new time range filter
func NewTimeRangeFilter(params map[string]interface{}) (Filter, error) {
	filter := &TimeRangeFilter{}

	// Parse "from" timestamp
	if fromStr, ok := params["from"].(string); ok && fromStr != "" {
		from, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return nil, fmt.Errorf("invalid 'from' timestamp format: %w (expected RFC3339)", err)
		}
		filter.from = &from
	}

	// Parse "to" timestamp
	if toStr, ok := params["to"].(string); ok && toStr != "" {
		to, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			return nil, fmt.Errorf("invalid 'to' timestamp format: %w (expected RFC3339)", err)
		}
		filter.to = &to
	}

	// Validate time range
	if filter.from != nil && filter.to != nil {
		if filter.from.After(*filter.to) {
			return nil, fmt.Errorf("invalid time range: 'from' (%s) must be before 'to' (%s)",
				filter.from.Format(time.RFC3339), filter.to.Format(time.RFC3339))
		}

		// Check max time range (90 days)
		duration := filter.to.Sub(*filter.from)
		maxDuration := 90 * 24 * time.Hour
		if duration > maxDuration {
			return nil, fmt.Errorf("time range too large: %v (max 90 days)", duration)
		}
	}

	// At least one timestamp must be provided
	if filter.from == nil && filter.to == nil {
		return nil, fmt.Errorf("time_range filter requires at least one of 'from' or 'to'")
	}

	return filter, nil
}

func (f *TimeRangeFilter) Type() FilterType {
	return FilterTypeTimeRange
}

func (f *TimeRangeFilter) Validate() error {
	if f.from == nil && f.to == nil {
		return fmt.Errorf("time_range filter requires at least one of 'from' or 'to'")
	}

	if f.from != nil && f.to != nil {
		if f.from.After(*f.to) {
			return fmt.Errorf("invalid time range: 'from' must be before 'to'")
		}

		// Check max time range (90 days)
		duration := f.to.Sub(*f.from)
		maxDuration := 90 * 24 * time.Hour
		if duration > maxDuration {
			return fmt.Errorf("time range too large: max 90 days")
		}
	}

	return nil
}

func (f *TimeRangeFilter) ApplyToQuery(qb *query.Builder) error {
	// Time range uses indexed starts_at column
	if f.from != nil {
		qb.AddWhere("starts_at >= ?", *f.from)
	}
	if f.to != nil {
		qb.AddWhere("starts_at <= ?", *f.to)
	}
	return nil
}

func (f *TimeRangeFilter) CacheKey() string {
	var parts []string
	if f.from != nil {
		parts = append(parts, fmt.Sprintf("from:%s", f.from.Format(time.RFC3339)))
	}
	if f.to != nil {
		parts = append(parts, fmt.Sprintf("to:%s", f.to.Format(time.RFC3339)))
	}
	return fmt.Sprintf("time_range:%s", strings.Join(parts, ","))
}
