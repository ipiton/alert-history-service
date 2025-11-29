// Package storage provides custom error types for storage operations.
package storage

import "fmt"

// ErrInvalidProfile indicates invalid deployment profile configuration.
// Returned when profile value is not "lite" or "standard",
// or when storage.backend doesn't match profile requirements.
type ErrInvalidProfile struct {
	Profile string // Profile value from config ("lite", "standard", or invalid)
	Cause   error  // Underlying validation error
}

func (e *ErrInvalidProfile) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("invalid deployment profile '%s': %v", e.Profile, e.Cause)
	}
	return fmt.Sprintf("invalid deployment profile: %s (must be 'lite' or 'standard')", e.Profile)
}

func (e *ErrInvalidProfile) Unwrap() error {
	return e.Cause
}

// ErrStorageInitFailed indicates storage backend initialization failure.
// Returned when SQLite file creation fails, Postgres connection fails,
// or schema initialization fails.
type ErrStorageInitFailed struct {
	Backend string // Storage backend name ("sqlite", "postgres")
	Profile string // Deployment profile ("lite", "standard")
	Cause   error  // Underlying error (connection, file I/O, etc.)
}

func (e *ErrStorageInitFailed) Error() string {
	return fmt.Sprintf("storage initialization failed (backend=%s, profile=%s): %v",
		e.Backend, e.Profile, e.Cause)
}

func (e *ErrStorageInitFailed) Unwrap() error {
	return e.Cause
}

// ErrInvalidFilePath indicates invalid SQLite file path.
// Returned when path contains "..", forbidden prefixes (/etc, /sys, /proc),
// or is empty (Lite profile).
type ErrInvalidFilePath struct {
	Path   string // Invalid path value
	Reason string // Why it's invalid (e.g., "contains '..'", "forbidden prefix")
}

func (e *ErrInvalidFilePath) Error() string {
	return fmt.Sprintf("invalid file path '%s': %s", e.Path, e.Reason)
}

// ErrConnectionFailed indicates storage connection failure.
// Returned when:
//   - SQLite file cannot be opened (permissions, disk full)
//   - Postgres connection times out or fails
//   - Connection pool exhausted
type ErrConnectionFailed struct {
	Backend string // "sqlite" or "postgres"
	Cause   error  // Underlying error (network, file I/O, etc.)
}

func (e *ErrConnectionFailed) Error() string {
	return fmt.Errorf("storage connection failed (%s): %w", e.Backend, e.Cause)
}

func (e *ErrConnectionFailed) Unwrap() error {
	return e.Cause
}

// ErrSchemaInitFailed indicates database schema initialization failure.
// Returned when:
//   - SQLite schema creation fails (table/index creation)
//   - Postgres migration fails
//   - Foreign key constraint violations
type ErrSchemaInitFailed struct {
	Backend string // "sqlite" or "postgres"
	Table   string // Table name that failed (optional)
	Cause   error  // Underlying SQL error
}

func (e *ErrSchemaInitFailed) Error() string {
	if e.Table != "" {
		return fmt.Sprintf("schema initialization failed (%s, table=%s): %v",
			e.Backend, e.Table, e.Cause)
	}
	return fmt.Sprintf("schema initialization failed (%s): %v", e.Backend, e.Cause)
}

func (e *ErrSchemaInitFailed) Unwrap() error {
	return e.Cause
}

// ErrDiskFull indicates disk space exhaustion (SQLite only).
// Returned when SQLite write fails due to insufficient disk space.
// Recommended action: Clean up old alerts, expand PVC, or upgrade to Standard profile.
type ErrDiskFull struct {
	Path      string // SQLite file path
	FileSize  int64  // Current file size (bytes)
	Available int64  // Available disk space (bytes), 0 if unknown
}

func (e *ErrDiskFull) Error() string {
	if e.Available > 0 {
		return fmt.Sprintf("disk full: SQLite file %s (size=%d bytes, available=%d bytes)",
			e.Path, e.FileSize, e.Available)
	}
	return fmt.Sprintf("disk full: SQLite file %s (size=%d bytes)", e.Path, e.FileSize)
}

// Error type classification for metrics
const (
	ErrorTypeConnection  = "connection"   // Connection/network failures
	ErrorTypeTimeout     = "timeout"      // Operation timeouts
	ErrorTypeNotFound    = "not_found"    // Alert not found errors
	ErrorTypeValidation  = "validation"   // Input validation errors
	ErrorTypeDiskFull    = "disk_full"    // Disk space exhaustion (SQLite)
	ErrorTypeSchema      = "schema"       // Schema initialization errors
	ErrorTypeUnknown     = "unknown"      // Uncategorized errors
)

// ClassifyError classifies error for metrics labeling.
// Returns error type constant (connection, timeout, etc.).
func ClassifyError(err error) string {
	switch {
	case err == nil:
		return ""
	case IsConnectionError(err):
		return ErrorTypeConnection
	case IsTimeoutError(err):
		return ErrorTypeTimeout
	case IsNotFoundError(err):
		return ErrorTypeNotFound
	case IsValidationError(err):
		return ErrorTypeValidation
	case IsDiskFullError(err):
		return ErrorTypeDiskFull
	case IsSchemaError(err):
		return ErrorTypeSchema
	default:
		return ErrorTypeUnknown
	}
}

// Error type checks for classification

func IsConnectionError(err error) bool {
	_, ok := err.(*ErrConnectionFailed)
	return ok
}

func IsTimeoutError(err error) bool {
	// TODO: Check for context.DeadlineExceeded or network timeout errors
	return false
}

func IsNotFoundError(err error) bool {
	// TODO: Check for core.ErrAlertNotFound
	return false
}

func IsValidationError(err error) bool {
	_, ok := err.(*ErrInvalidFilePath)
	if ok {
		return true
	}
	_, ok = err.(*ErrInvalidProfile)
	return ok
}

func IsDiskFullError(err error) bool {
	_, ok := err.(*ErrDiskFull)
	return ok
}

func IsSchemaError(err error) bool {
	_, ok := err.(*ErrSchemaInitFailed)
	return ok
}
