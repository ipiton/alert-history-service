package filters

import (
	"fmt"
	
	"github.com/vitaliisemenov/alert-history/pkg/history/query"
)

// SearchFilter filters alerts by full-text search across alert_name, annotations
type SearchFilter struct {
	query string
}

// NewSearchFilter creates a new search filter
func NewSearchFilter(params map[string]interface{}) (Filter, error) {
	queryStr, ok := params["query"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid search filter params: expected string")
	}
	
	if queryStr == "" {
		return nil, fmt.Errorf("search filter requires non-empty query")
	}
	
	if len(queryStr) > 500 {
		return nil, fmt.Errorf("search query too long: max 500 characters")
	}
	
	return &SearchFilter{query: queryStr}, nil
}

func (f *SearchFilter) Type() FilterType {
	return FilterTypeSearch
}

func (f *SearchFilter) Validate() error {
	if f.query == "" {
		return fmt.Errorf("search filter requires non-empty query")
	}
	if len(f.query) > 500 {
		return fmt.Errorf("search query too long: max 500 characters")
	}
	return nil
}

func (f *SearchFilter) ApplyToQuery(qb *query.Builder) error {
	// Full-text search across multiple fields using ILIKE (case-insensitive)
	// Search in: alert_name, annotations->>'summary', annotations->>'description'
	searchPattern := "%" + f.query + "%"
	qb.AddWhere(`(
		alert_name ILIKE ? OR
		annotations->>'summary' ILIKE ? OR
		annotations->>'description' ILIKE ?
	)`, searchPattern, searchPattern, searchPattern)
	return nil
}

func (f *SearchFilter) CacheKey() string {
	return fmt.Sprintf("search:%s", f.query)
}

