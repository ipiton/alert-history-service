package filters

// FilterType represents a type of filter
type FilterType string

const (
	// Basic Filters (P0 - Critical)
	FilterTypeStatus       FilterType = "status"
	FilterTypeSeverity     FilterType = "severity"
	FilterTypeNamespace    FilterType = "namespace"
	FilterTypeTimeRange    FilterType = "time_range"
	FilterTypeLabels       FilterType = "labels"
	
	// Advanced Filters (P1 - High)
	FilterTypeFingerprint  FilterType = "fingerprint"
	FilterTypeAlertName    FilterType = "alert_name"
	FilterTypeAlertNamePattern FilterType = "alert_name_pattern"
	FilterTypeAlertNameRegex   FilterType = "alert_name_regex"
	
	// Label Operators (P1 - High)
	FilterTypeLabelsExact    FilterType = "labels_exact"
	FilterTypeLabelsNotEqual FilterType = "labels_not_equal"
	FilterTypeLabelsRegex    FilterType = "labels_regex"
	FilterTypeLabelsNotRegex FilterType = "labels_not_regex"
	FilterTypeLabelsExists   FilterType = "labels_exists"
	FilterTypeLabelsNotExists FilterType = "labels_not_exists"
	
	// Advanced Features (P2 - Medium)
	FilterTypeSearch      FilterType = "search"
	FilterTypeDuration    FilterType = "duration"
	FilterTypeGeneratorURL FilterType = "generator_url"
	FilterTypeIsFlapping  FilterType = "is_flapping"
	FilterTypeIsResolved  FilterType = "is_resolved"
)

// String returns the string representation of FilterType
func (ft FilterType) String() string {
	return string(ft)
}

// IsValid checks if the filter type is valid
func (ft FilterType) IsValid() bool {
	validTypes := map[FilterType]bool{
		FilterTypeStatus:          true,
		FilterTypeSeverity:        true,
		FilterTypeNamespace:       true,
		FilterTypeTimeRange:       true,
		FilterTypeLabels:          true,
		FilterTypeFingerprint:     true,
		FilterTypeAlertName:       true,
		FilterTypeAlertNamePattern: true,
		FilterTypeAlertNameRegex:  true,
		FilterTypeLabelsExact:     true,
		FilterTypeLabelsNotEqual:  true,
		FilterTypeLabelsRegex:     true,
		FilterTypeLabelsNotRegex:  true,
		FilterTypeLabelsExists:    true,
		FilterTypeLabelsNotExists: true,
		FilterTypeSearch:          true,
		FilterTypeDuration:        true,
		FilterTypeGeneratorURL:    true,
		FilterTypeIsFlapping:     true,
		FilterTypeIsResolved:     true,
	}
	return validTypes[ft]
}

