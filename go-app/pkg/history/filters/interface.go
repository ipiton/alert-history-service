package filters

import (
	"github.com/vitaliisemenov/alert-history/pkg/history/query"
)

// Filter interface defines common operations for all filters
type Filter interface {
	// Type returns the filter type
	Type() FilterType
	
	// Validate validates the filter parameters
	Validate() error
	
	// ApplyToQuery applies the filter to a query builder
	ApplyToQuery(qb *query.Builder) error
	
	// CacheKey returns a cache key representation of the filter
	// This is used for generating cache keys for query results
	CacheKey() string
}

// FilterFactory creates filter instances from parameters
type FilterFactory func(params map[string]interface{}) (Filter, error)

