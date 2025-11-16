package filters

import (
	"fmt"
	"strings"
	"time"
	
	"github.com/vitaliisemenov/alert-history/pkg/history/query"
)

// DurationFilter filters alerts by duration (ends_at - starts_at)
type DurationFilter struct {
	min *time.Duration
	max *time.Duration
}

// NewDurationFilter creates a new duration filter
func NewDurationFilter(params map[string]interface{}) (Filter, error) {
	filter := &DurationFilter{}
	
	// Parse "min" duration
	if minStr, ok := params["min"].(string); ok && minStr != "" {
		min, err := time.ParseDuration(minStr)
		if err != nil {
			return nil, fmt.Errorf("invalid 'duration_min' format: %w (expected Go duration format like '5m', '1h')", err)
		}
		if min < 0 {
			return nil, fmt.Errorf("duration_min must be non-negative")
		}
		filter.min = &min
	}
	
	// Parse "max" duration
	if maxStr, ok := params["max"].(string); ok && maxStr != "" {
		max, err := time.ParseDuration(maxStr)
		if err != nil {
			return nil, fmt.Errorf("invalid 'duration_max' format: %w (expected Go duration format like '5m', '1h')", err)
		}
		if max < 0 {
			return nil, fmt.Errorf("duration_max must be non-negative")
		}
		filter.max = &max
	}
	
	// Validate duration range
	if filter.min != nil && filter.max != nil {
		if *filter.min > *filter.max {
			return nil, fmt.Errorf("invalid duration range: min (%v) must be <= max (%v)", *filter.min, *filter.max)
		}
	}
	
	// At least one duration must be provided
	if filter.min == nil && filter.max == nil {
		return nil, fmt.Errorf("duration filter requires at least one of 'duration_min' or 'duration_max'")
	}
	
	return filter, nil
}

func (f *DurationFilter) Type() FilterType {
	return FilterTypeDuration
}

func (f *DurationFilter) Validate() error {
	if f.min == nil && f.max == nil {
		return fmt.Errorf("duration filter requires at least one of 'min' or 'max'")
	}
	
	if f.min != nil && *f.min < 0 {
		return fmt.Errorf("duration_min must be non-negative")
	}
	
	if f.max != nil && *f.max < 0 {
		return fmt.Errorf("duration_max must be non-negative")
	}
	
	if f.min != nil && f.max != nil {
		if *f.min > *f.max {
			return fmt.Errorf("invalid duration range: min must be <= max")
		}
	}
	
	return nil
}

func (f *DurationFilter) ApplyToQuery(qb *query.Builder) error {
	// Calculate duration: COALESCE(ends_at, NOW()) - starts_at
	// Handle null ends_at by using NOW() for firing alerts
	if f.min != nil {
		minSeconds := f.min.Seconds()
		qb.AddWhere("EXTRACT(EPOCH FROM (COALESCE(ends_at, NOW()) - starts_at)) >= ?", minSeconds)
	}
	if f.max != nil {
		maxSeconds := f.max.Seconds()
		qb.AddWhere("EXTRACT(EPOCH FROM (COALESCE(ends_at, NOW()) - starts_at)) <= ?", maxSeconds)
	}
	return nil
}

func (f *DurationFilter) CacheKey() string {
	var parts []string
	if f.min != nil {
		parts = append(parts, fmt.Sprintf("min:%v", *f.min))
	}
	if f.max != nil {
		parts = append(parts, fmt.Sprintf("max:%v", *f.max))
	}
	return fmt.Sprintf("duration:%s", strings.Join(parts, ","))
}

