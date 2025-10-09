package core

import "errors"

// Filter validation errors
var (
	// Limit/Offset errors
	ErrInvalidFilterLimit  = errors.New("filter limit must be >= 0")
	ErrFilterLimitTooLarge = errors.New("filter limit must be <= 1000")
	ErrInvalidFilterOffset = errors.New("filter offset must be >= 0")

	// Status errors
	ErrInvalidFilterStatus = errors.New("invalid filter status: must be 'firing' or 'resolved'")

	// Severity errors
	ErrInvalidFilterSeverity = errors.New("invalid filter severity: must be 'critical', 'warning', 'info', or 'noise'")

	// TimeRange errors
	ErrInvalidTimeRange = errors.New("invalid time range: 'from' must be before 'to'")

	// Label errors
	ErrTooManyLabels      = errors.New("too many label filters: maximum 20 labels allowed")
	ErrEmptyLabelKey      = errors.New("label key cannot be empty")
	ErrLabelKeyTooLong    = errors.New("label key too long: maximum 255 characters")
	ErrLabelValueTooLong  = errors.New("label value too long: maximum 255 characters")
)
