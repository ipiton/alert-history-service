package metrics

import (
	"time"
	
	"github.com/vitaliisemenov/alert-history/pkg/history/filters"
)

// RecordFilterOperation records a filter operation
func (m *HistoryMetrics) RecordFilterOperation(filterType filters.FilterType, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
		errorType := "unknown"
		if err != nil {
			errorType = "validation"
		}
		m.FilterErrorsTotal.WithLabelValues(string(filterType), errorType).Inc()
	}
	
	m.FilterOperationsTotal.WithLabelValues(string(filterType), status).Inc()
	m.FilterDuration.WithLabelValues(string(filterType)).Observe(duration.Seconds())
}

// RecordFiltersApplied records number of filters applied
func (m *HistoryMetrics) RecordFiltersApplied(endpoint string, count int) {
	m.FiltersApplied.WithLabelValues(endpoint).Observe(float64(count))
}

