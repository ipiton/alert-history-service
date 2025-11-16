package filters

import (
	"fmt"
	"regexp"
	"time"

	"github.com/vitaliisemenov/alert-history/pkg/history/query"
)

// AlertNameRegexFilter filters alerts by alert name regex pattern
type AlertNameRegexFilter struct {
	pattern *regexp.Regexp
	patternStr string
}

// NewAlertNameRegexFilter creates a new alert name regex filter
func NewAlertNameRegexFilter(params map[string]interface{}) (Filter, error) {
	patternStr, ok := params["pattern"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid alert_name_regex filter params: expected string")
	}

	if patternStr == "" {
		return nil, fmt.Errorf("alert_name_regex filter requires non-empty pattern")
	}

	if len(patternStr) > 500 {
		return nil, fmt.Errorf("alert_name_regex pattern too long: max 500 characters")
	}

	// Validate and compile regex pattern
	// Use timeout to prevent ReDoS attacks
	done := make(chan bool, 1)
	var compiled *regexp.Regexp
	var compileErr error

	go func() {
		compiled, compileErr = regexp.Compile(patternStr)
		done <- true
	}()

	select {
	case <-done:
		if compileErr != nil {
			return nil, fmt.Errorf("invalid regex pattern: %w", compileErr)
		}
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("regex compilation timeout (possible ReDoS attack)")
	}

	return &AlertNameRegexFilter{
		pattern:    compiled,
		patternStr: patternStr,
	}, nil
}

func (f *AlertNameRegexFilter) Type() FilterType {
	return FilterTypeAlertNameRegex
}

func (f *AlertNameRegexFilter) Validate() error {
	if f.patternStr == "" {
		return fmt.Errorf("alert_name_regex filter requires non-empty pattern")
	}
	if f.pattern == nil {
		return fmt.Errorf("regex pattern not compiled")
	}
	return nil
}

func (f *AlertNameRegexFilter) ApplyToQuery(qb *query.Builder) error {
	// PostgreSQL regex operator (~)
	// Note: This doesn't use index, so it's slower than LIKE
	qb.AddWhere("alert_name ~ ?", f.patternStr)
	return nil
}

func (f *AlertNameRegexFilter) CacheKey() string {
	return fmt.Sprintf("alert_name_regex:%s", f.patternStr)
}
