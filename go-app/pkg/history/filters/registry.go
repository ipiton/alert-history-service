package filters

import (
	"fmt"
	"log/slog"
)

// Registry manages all available filters
type Registry struct {
	factories map[FilterType]FilterFactory
	logger    *slog.Logger
}

// NewRegistry creates a new filter registry with all filters registered
func NewRegistry(logger *slog.Logger) *Registry {
	if logger == nil {
		logger = slog.Default()
	}

	registry := &Registry{
		factories: make(map[FilterType]FilterFactory),
		logger:    logger,
	}

	// Register all filter factories
	registry.Register(FilterTypeStatus, NewStatusFilter)
	registry.Register(FilterTypeSeverity, NewSeverityFilter)
	registry.Register(FilterTypeNamespace, NewNamespaceFilter)
	registry.Register(FilterTypeFingerprint, NewFingerprintFilter)
	registry.Register(FilterTypeAlertName, NewAlertNameFilter)
	registry.Register(FilterTypeAlertNamePattern, NewAlertNamePatternFilter)
	registry.Register(FilterTypeAlertNameRegex, NewAlertNameRegexFilter)
	registry.Register(FilterTypeLabelsExact, NewLabelsExactFilter)
	registry.Register(FilterTypeLabelsNotEqual, NewLabelsNotEqualFilter)
	registry.Register(FilterTypeLabelsRegex, NewLabelsRegexFilter)
	registry.Register(FilterTypeLabelsNotRegex, NewLabelsNotRegexFilter)
	registry.Register(FilterTypeLabelsExists, NewLabelsExistsFilter)
	registry.Register(FilterTypeLabelsNotExists, NewLabelsNotExistsFilter)
	registry.Register(FilterTypeTimeRange, NewTimeRangeFilter)
	registry.Register(FilterTypeSearch, NewSearchFilter)
	registry.Register(FilterTypeDuration, NewDurationFilter)
	registry.Register(FilterTypeGeneratorURL, NewGeneratorURLFilter)
	registry.Register(FilterTypeIsFlapping, NewIsFlappingFilter)
	registry.Register(FilterTypeIsResolved, NewIsResolvedFilter)

	// Legacy support: register "labels" as "labels_exact"
	registry.Register(FilterTypeLabels, NewLabelsExactFilter)

	return registry
}

// Register adds a filter factory to the registry
func (r *Registry) Register(typ FilterType, factory FilterFactory) {
	if !typ.IsValid() {
		r.logger.Warn("Registering invalid filter type", "type", typ)
	}
	r.factories[typ] = factory
}

// Create creates a filter instance from parameters
func (r *Registry) Create(typ FilterType, params map[string]interface{}) (Filter, error) {
	factory, ok := r.factories[typ]
	if !ok {
		return nil, fmt.Errorf("unknown filter type: %s", typ)
	}

	filter, err := factory(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create filter %s: %w", typ, err)
	}

	return filter, nil
}

// CreateFromQueryParams creates filters from HTTP query parameters
func (r *Registry) CreateFromQueryParams(queryParams map[string][]string) ([]Filter, error) {
	var filters []Filter

	// Parse status filter
	if values, ok := queryParams["status"]; ok && len(values) > 0 {
		filter, err := r.Create(FilterTypeStatus, map[string]interface{}{
			"values": values,
		})
		if err != nil {
			return nil, err
		}
		if err := filter.Validate(); err != nil {
			return nil, fmt.Errorf("invalid status filter: %w", err)
		}
		filters = append(filters, filter)
	}

	// Parse severity filter
	if values, ok := queryParams["severity"]; ok && len(values) > 0 {
		filter, err := r.Create(FilterTypeSeverity, map[string]interface{}{
			"values": values,
		})
		if err != nil {
			return nil, err
		}
		if err := filter.Validate(); err != nil {
			return nil, fmt.Errorf("invalid severity filter: %w", err)
		}
		filters = append(filters, filter)
	}

	// Parse namespace filter
	if values, ok := queryParams["namespace"]; ok && len(values) > 0 {
		filter, err := r.Create(FilterTypeNamespace, map[string]interface{}{
			"values": values,
		})
		if err != nil {
			return nil, err
		}
		if err := filter.Validate(); err != nil {
			return nil, fmt.Errorf("invalid namespace filter: %w", err)
		}
		filters = append(filters, filter)
	}

	// Parse fingerprint filter
	if values, ok := queryParams["fingerprints"]; ok && len(values) > 0 {
		filter, err := r.Create(FilterTypeFingerprint, map[string]interface{}{
			"values": values,
		})
		if err != nil {
			return nil, err
		}
		if err := filter.Validate(); err != nil {
			return nil, fmt.Errorf("invalid fingerprint filter: %w", err)
		}
		filters = append(filters, filter)
	}

	// Parse alert_name filter
	if values, ok := queryParams["alert_name"]; ok && len(values) > 0 && len(values[0]) > 0 {
		filter, err := r.Create(FilterTypeAlertName, map[string]interface{}{
			"value": values[0],
		})
		if err != nil {
			return nil, err
		}
		if err := filter.Validate(); err != nil {
			return nil, fmt.Errorf("invalid alert_name filter: %w", err)
		}
		filters = append(filters, filter)
	}

	// Parse alert_name_pattern filter
	if values, ok := queryParams["alert_name_pattern"]; ok && len(values) > 0 && len(values[0]) > 0 {
		filter, err := r.Create(FilterTypeAlertNamePattern, map[string]interface{}{
			"pattern": values[0],
		})
		if err != nil {
			return nil, err
		}
		if err := filter.Validate(); err != nil {
			return nil, fmt.Errorf("invalid alert_name_pattern filter: %w", err)
		}
		filters = append(filters, filter)
	}

	// Parse alert_name_regex filter
	if values, ok := queryParams["alert_name_regex"]; ok && len(values) > 0 && len(values[0]) > 0 {
		filter, err := r.Create(FilterTypeAlertNameRegex, map[string]interface{}{
			"pattern": values[0],
		})
		if err != nil {
			return nil, err
		}
		if err := filter.Validate(); err != nil {
			return nil, fmt.Errorf("invalid alert_name_regex filter: %w", err)
		}
		filters = append(filters, filter)
	}

	// Parse labels filters (exact match - legacy support)
	if labels, ok := queryParams["labels"]; ok && len(labels) > 0 {
		// Parse label key-value pairs from format: "key=value" or "key:value"
		labelMap := make(map[string]string)
		for _, label := range labels {
			// Support both "key=value" and "key:value" formats
			var key, value string
			if idx := findChar(label, '=', ':'); idx > 0 {
				key = label[:idx]
				value = label[idx+1:]
			} else {
				// If no separator, treat as key only (exists filter)
				key = label
				value = ""
			}
			if key != "" {
				labelMap[key] = value
			}
		}

		if len(labelMap) > 0 {
			filter, err := r.Create(FilterTypeLabelsExact, map[string]interface{}{
				"labels": labelMap,
			})
			if err != nil {
				return nil, err
			}
			if err := filter.Validate(); err != nil {
				return nil, fmt.Errorf("invalid labels filter: %w", err)
			}
			filters = append(filters, filter)
		}
	}

	// Parse labels_exact filter (new format: labels_exact[key]=value)
	labelExactMap := parseLabelFilters(queryParams, "labels_exact")
	if len(labelExactMap) > 0 {
		filter, err := r.Create(FilterTypeLabelsExact, map[string]interface{}{
			"labels": labelExactMap,
		})
		if err != nil {
			return nil, err
		}
		if err := filter.Validate(); err != nil {
			return nil, fmt.Errorf("invalid labels_exact filter: %w", err)
		}
		filters = append(filters, filter)
	}

	// Parse labels_ne filter
	labelNeMap := parseLabelFilters(queryParams, "labels_ne")
	if len(labelNeMap) > 0 {
		filter, err := r.Create(FilterTypeLabelsNotEqual, map[string]interface{}{
			"labels": labelNeMap,
		})
		if err != nil {
			return nil, err
		}
		if err := filter.Validate(); err != nil {
			return nil, fmt.Errorf("invalid labels_ne filter: %w", err)
		}
		filters = append(filters, filter)
	}

	// Parse labels_regex filter
	labelRegexMap := parseLabelFilters(queryParams, "labels_regex")
	if len(labelRegexMap) > 0 {
		filter, err := r.Create(FilterTypeLabelsRegex, map[string]interface{}{
			"labels": labelRegexMap,
		})
		if err != nil {
			return nil, err
		}
		if err := filter.Validate(); err != nil {
			return nil, fmt.Errorf("invalid labels_regex filter: %w", err)
		}
		filters = append(filters, filter)
	}

	// Parse labels_not_regex filter
	labelNotRegexMap := parseLabelFilters(queryParams, "labels_not_regex")
	if len(labelNotRegexMap) > 0 {
		filter, err := r.Create(FilterTypeLabelsNotRegex, map[string]interface{}{
			"labels": labelNotRegexMap,
		})
		if err != nil {
			return nil, err
		}
		if err := filter.Validate(); err != nil {
			return nil, fmt.Errorf("invalid labels_not_regex filter: %w", err)
		}
		filters = append(filters, filter)
	}

	// Parse labels_exists filter
	if values, ok := queryParams["labels_exists"]; ok && len(values) > 0 {
		filter, err := r.Create(FilterTypeLabelsExists, map[string]interface{}{
			"keys": values,
		})
		if err != nil {
			return nil, err
		}
		if err := filter.Validate(); err != nil {
			return nil, fmt.Errorf("invalid labels_exists filter: %w", err)
		}
		filters = append(filters, filter)
	}

	// Parse labels_not_exists filter
	if values, ok := queryParams["labels_not_exists"]; ok && len(values) > 0 {
		filter, err := r.Create(FilterTypeLabelsNotExists, map[string]interface{}{
			"keys": values,
		})
		if err != nil {
			return nil, err
		}
		if err := filter.Validate(); err != nil {
			return nil, fmt.Errorf("invalid labels_not_exists filter: %w", err)
		}
		filters = append(filters, filter)
	}

	// Parse time range filter
	hasTimeRange := false
	timeRangeParams := make(map[string]interface{})
	if values, ok := queryParams["from"]; ok && len(values) > 0 && len(values[0]) > 0 {
		timeRangeParams["from"] = values[0]
		hasTimeRange = true
	}
	if values, ok := queryParams["to"]; ok && len(values) > 0 && len(values[0]) > 0 {
		timeRangeParams["to"] = values[0]
		hasTimeRange = true
	}
	if hasTimeRange {
		filter, err := r.Create(FilterTypeTimeRange, timeRangeParams)
		if err != nil {
			return nil, err
		}
		if err := filter.Validate(); err != nil {
			return nil, fmt.Errorf("invalid time_range filter: %w", err)
		}
		filters = append(filters, filter)
	}

	// Parse search filter
	if values, ok := queryParams["search"]; ok && len(values) > 0 && len(values[0]) > 0 {
		filter, err := r.Create(FilterTypeSearch, map[string]interface{}{
			"query": values[0],
		})
		if err != nil {
			return nil, err
		}
		if err := filter.Validate(); err != nil {
			return nil, fmt.Errorf("invalid search filter: %w", err)
		}
		filters = append(filters, filter)
	}

	// Parse duration filters
	durationParams := make(map[string]interface{})
	if values, ok := queryParams["duration_min"]; ok && len(values) > 0 && len(values[0]) > 0 {
		durationParams["min"] = values[0]
	}
	if values, ok := queryParams["duration_max"]; ok && len(values) > 0 && len(values[0]) > 0 {
		durationParams["max"] = values[0]
	}
	if len(durationParams) > 0 {
		filter, err := r.Create(FilterTypeDuration, durationParams)
		if err != nil {
			return nil, err
		}
		if err := filter.Validate(); err != nil {
			return nil, fmt.Errorf("invalid duration filter: %w", err)
		}
		filters = append(filters, filter)
	}

	// Parse generator_url filter
	if values, ok := queryParams["generator_url"]; ok && len(values) > 0 && len(values[0]) > 0 {
		filter, err := r.Create(FilterTypeGeneratorURL, map[string]interface{}{
			"url": values[0],
		})
		if err != nil {
			return nil, err
		}
		if err := filter.Validate(); err != nil {
			return nil, fmt.Errorf("invalid generator_url filter: %w", err)
		}
		filters = append(filters, filter)
	}

	// Parse is_flapping filter
	if values, ok := queryParams["is_flapping"]; ok && len(values) > 0 && len(values[0]) > 0 {
		filter, err := r.Create(FilterTypeIsFlapping, map[string]interface{}{
			"value": values[0],
		})
		if err != nil {
			return nil, err
		}
		if err := filter.Validate(); err != nil {
			return nil, fmt.Errorf("invalid is_flapping filter: %w", err)
		}
		filters = append(filters, filter)
	}

	// Parse is_resolved filter
	if values, ok := queryParams["is_resolved"]; ok && len(values) > 0 && len(values[0]) > 0 {
		filter, err := r.Create(FilterTypeIsResolved, map[string]interface{}{
			"value": values[0],
		})
		if err != nil {
			return nil, err
		}
		if err := filter.Validate(); err != nil {
			return nil, fmt.Errorf("invalid is_resolved filter: %w", err)
		}
		filters = append(filters, filter)
	}

	return filters, nil
}

// parseLabelFilters parses label filters from query parameters
// Supports formats: labels_exact[key]=value or labels_exact[]=key=value
func parseLabelFilters(queryParams map[string][]string, prefix string) map[string]string {
	result := make(map[string]string)

	// Try format: labels_exact[key]=value
	for key, values := range queryParams {
		if len(key) > len(prefix)+2 && key[:len(prefix)] == prefix && key[len(prefix)] == '[' {
			// Extract label key from brackets: labels_exact[pod] -> pod
			labelKey := key[len(prefix)+1 : len(key)-1]
			if len(values) > 0 && len(values[0]) > 0 {
				result[labelKey] = values[0]
			}
		}
	}

	// Try format: labels_exact[]=key=value
	if values, ok := queryParams[prefix+"[]"]; ok {
		for _, value := range values {
			if idx := findChar(value, '=', ':'); idx > 0 {
				key := value[:idx]
				val := value[idx+1:]
				if key != "" {
					result[key] = val
				}
			}
		}
	}

	return result
}

// findChar finds the first occurrence of any of the given characters
func findChar(s string, chars ...rune) int {
	for i, r := range s {
		for _, c := range chars {
			if r == c {
				return i
			}
		}
	}
	return -1
}
