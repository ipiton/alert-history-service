package filters

import (
	"fmt"
	"net/url"
	
	"github.com/vitaliisemenov/alert-history/pkg/history/query"
)

// GeneratorURLFilter filters alerts by generator URL
type GeneratorURLFilter struct {
	url string
}

// NewGeneratorURLFilter creates a new generator URL filter
func NewGeneratorURLFilter(params map[string]interface{}) (Filter, error) {
	urlStr, ok := params["url"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid generator_url filter params: expected string")
	}
	
	if urlStr == "" {
		return nil, fmt.Errorf("generator_url filter requires non-empty URL")
	}
	
	// Validate URL format
	_, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid generator_url format: %w", err)
	}
	
	return &GeneratorURLFilter{url: urlStr}, nil
}

func (f *GeneratorURLFilter) Type() FilterType {
	return FilterTypeGeneratorURL
}

func (f *GeneratorURLFilter) Validate() error {
	if f.url == "" {
		return fmt.Errorf("generator_url filter requires non-empty URL")
	}
	
	_, err := url.Parse(f.url)
	if err != nil {
		return fmt.Errorf("invalid generator_url format: %w", err)
	}
	
	return nil
}

func (f *GeneratorURLFilter) ApplyToQuery(qb *query.Builder) error {
	// Exact match on generator_url column
	// Handle NULL values (some alerts may not have generator_url)
	qb.AddWhere("generator_url = ?", f.url)
	return nil
}

func (f *GeneratorURLFilter) CacheKey() string {
	return fmt.Sprintf("generator_url:%s", f.url)
}

