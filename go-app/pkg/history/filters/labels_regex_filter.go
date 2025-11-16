package filters

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/vitaliisemenov/alert-history/pkg/history/query"
)

// LabelsRegexFilter filters alerts by label regex pattern (=~)
type LabelsRegexFilter struct {
	labels map[string]*regexp.Regexp
	patterns map[string]string // Store original patterns for SQL
}

// NewLabelsRegexFilter creates a new labels regex filter
func NewLabelsRegexFilter(params map[string]interface{}) (Filter, error) {
	labels, ok := params["labels"].(map[string]string)
	if !ok {
		return nil, fmt.Errorf("invalid labels_regex filter params: expected map[string]string")
	}

	if len(labels) == 0 {
		return nil, fmt.Errorf("labels_regex filter requires at least one label")
	}

	if len(labels) > 20 {
		return nil, fmt.Errorf("too many labels: max 20, got %d", len(labels))
	}

	compiled := make(map[string]*regexp.Regexp)
	patterns := make(map[string]string)

	// Validate and compile regex patterns
	for key, pattern := range labels {
		if len(key) == 0 {
			return nil, fmt.Errorf("label key cannot be empty")
		}
		if len(key) > 255 {
			return nil, fmt.Errorf("label key too long: max 255 characters")
		}
		if len(pattern) > 500 {
			return nil, fmt.Errorf("regex pattern too long: max 500 characters")
		}

		// Compile regex with timeout to prevent ReDoS
		done := make(chan bool, 1)
		var compiledRegex *regexp.Regexp
		var compileErr error

		go func() {
			compiledRegex, compileErr = regexp.Compile(pattern)
			done <- true
		}()

		select {
		case <-done:
			if compileErr != nil {
				return nil, fmt.Errorf("invalid regex pattern for label %s: %w", key, compileErr)
			}
		case <-time.After(5 * time.Second):
			return nil, fmt.Errorf("regex compilation timeout for label %s (possible ReDoS attack)", key)
		}

		compiled[key] = compiledRegex
		patterns[key] = pattern
	}

	return &LabelsRegexFilter{
		labels:   compiled,
		patterns: patterns,
	}, nil
}

func (f *LabelsRegexFilter) Type() FilterType {
	return FilterTypeLabelsRegex
}

func (f *LabelsRegexFilter) Validate() error {
	if len(f.labels) == 0 {
		return fmt.Errorf("labels_regex filter requires at least one label")
	}
	if len(f.labels) > 20 {
		return fmt.Errorf("too many labels: max 20")
	}

	for key := range f.labels {
		if len(key) == 0 {
			return fmt.Errorf("label key cannot be empty")
		}
		if len(key) > 255 {
			return fmt.Errorf("label key too long: max 255 characters")
		}
	}

	return nil
}

func (f *LabelsRegexFilter) ApplyToQuery(qb *query.Builder) error {
	// Mark for GIN index usage (labels JSONB field)
	qb.MarkGINIndexUsage()

	// Use PostgreSQL regex operator (~) on label values
	// Multiple labels use AND logic (all must match)
	for key, pattern := range f.patterns {
		qb.AddWhere("labels->>? ~ ?", key, pattern)
	}

	return nil
}

func (f *LabelsRegexFilter) CacheKey() string {
	// Sort keys for consistent cache keys
	keys := make([]string, 0, len(f.patterns))
	for k := range f.patterns {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=~%s", k, f.patterns[k]))
	}
	return fmt.Sprintf("labels_regex:%s", strings.Join(parts, ","))
}
