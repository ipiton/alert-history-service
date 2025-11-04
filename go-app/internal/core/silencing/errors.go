package silencing

import "errors"

// Silence validation errors

var (
	// ErrSilenceInvalidID indicates that the silence ID is not a valid UUID v4.
	// The ID must be a valid UUID format (e.g., "550e8400-e29b-41d4-a716-446655440000").
	ErrSilenceInvalidID = errors.New("invalid silence ID: must be a valid UUID v4")

	// ErrSilenceInvalidCreatedBy indicates that the CreatedBy field is invalid.
	// The field must be non-empty and not exceed 255 characters.
	ErrSilenceInvalidCreatedBy = errors.New("invalid createdBy: must be non-empty and at most 255 characters")

	// ErrSilenceInvalidComment indicates that the Comment field is invalid.
	// The comment must be at least 3 characters and at most 1024 characters.
	ErrSilenceInvalidComment = errors.New("invalid comment: must be between 3 and 1024 characters")

	// ErrSilenceInvalidTimeRange indicates that the time range is invalid.
	// The EndsAt timestamp must be strictly after StartsAt.
	ErrSilenceInvalidTimeRange = errors.New("invalid time range: endsAt must be after startsAt")

	// ErrSilenceNoMatchers indicates that no matchers were provided.
	// At least one matcher is required to define which alerts to silence.
	ErrSilenceNoMatchers = errors.New("no matchers provided: at least one matcher is required")

	// ErrSilenceTooManyMatchers indicates that too many matchers were provided.
	// Maximum 100 matchers are allowed to prevent DoS attacks.
	ErrSilenceTooManyMatchers = errors.New("too many matchers: maximum 100 matchers allowed")
)

// Matcher validation errors

var (
	// ErrMatcherInvalidName indicates that the matcher's label name is invalid.
	// Label names must follow Prometheus format: [a-zA-Z_][a-zA-Z0-9_]*
	// Examples of invalid names: "9name" (starts with digit), "label-name" (contains hyphen)
	ErrMatcherInvalidName = errors.New("invalid matcher name: must follow Prometheus label format [a-zA-Z_][a-zA-Z0-9_]*")

	// ErrMatcherEmptyValue indicates that the matcher's value is empty.
	// The value field must contain at least one character.
	ErrMatcherEmptyValue = errors.New("invalid matcher value: cannot be empty")

	// ErrMatcherValueTooLong indicates that the matcher's value exceeds the maximum length.
	// The value must not exceed 1024 characters.
	ErrMatcherValueTooLong = errors.New("invalid matcher value: maximum 1024 characters allowed")

	// ErrMatcherInvalidType indicates that the matcher type is not one of the valid types.
	// Valid types are: = (equal), != (not equal), =~ (regex), !~ (not regex)
	ErrMatcherInvalidType = errors.New("invalid matcher type: must be one of =, !=, =~, !~")

	// ErrMatcherInvalidRegex indicates that the regex pattern in a regex matcher is invalid.
	// This error is returned when Type is =~ or !~ and the Value is not a valid regex pattern.
	ErrMatcherInvalidRegex = errors.New("invalid regex pattern in matcher")
)
