package cache

import (
	"errors"
	"testing"
)

// TestCacheError_Error tests CacheError.Error() method
func TestCacheError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *CacheError
		want string
	}{
		{
			name: "simple message",
			err:  &CacheError{Message: "cache miss"},
			want: "cache miss",
		},
		{
			name: "with cause",
			err: &CacheError{
				Message: "set failed",
				Cause:   errors.New("redis timeout"),
			},
			want: "set failed: redis timeout",
		},
		{
			name: "with type",
			err: &CacheError{
				Message: "not found",
				Type:    ErrTypeNotFound,
			},
			want: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("CacheError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestCacheError_Unwrap tests CacheError.Unwrap() method
func TestCacheError_Unwrap(t *testing.T) {
	tests := []struct {
		name    string
		err     *CacheError
		wantNil bool
	}{
		{
			name: "with cause",
			err: &CacheError{
				Message: "failed",
				Cause:   errors.New("inner error"),
			},
			wantNil: false,
		},
		{
			name: "without cause",
			err: &CacheError{
				Message: "failed",
				Cause:   nil,
			},
			wantNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Unwrap()
			if tt.wantNil && got != nil {
				t.Errorf("CacheError.Unwrap() = %v, want nil", got)
			}
			if !tt.wantNil && got == nil {
				t.Error("CacheError.Unwrap() = nil, want non-nil")
			}
		})
	}
}

// TestErrInvalidConfig tests ErrInvalidConfig constructor
func TestErrInvalidConfig(t *testing.T) {
	tests := []struct {
		name    string
		msg     string
		wantMsg string
	}{
		{
			name:    "simple message",
			msg:     "ttl must be positive",
			wantMsg: "ttl must be positive",
		},
		{
			name:    "complex message",
			msg:     "max_size_mb must be between 1 and 1000",
			wantMsg: "max_size_mb must be between 1 and 1000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ErrInvalidConfig(tt.msg)
			if err.Error() != tt.wantMsg {
				t.Errorf("ErrInvalidConfig() = %v, want %v", err.Error(), tt.wantMsg)
			}

			// Check type
			cacheErr, ok := err.(*CacheError)
			if !ok {
				t.Error("ErrInvalidConfig() should return *CacheError")
			}
			if cacheErr.Type != ErrTypeInvalidConfig {
				t.Errorf("ErrInvalidConfig().Type = %v, want %v", cacheErr.Type, ErrTypeInvalidConfig)
			}
		})
	}
}

// TestErrSerialization tests ErrSerialization constructor
func TestErrSerialization(t *testing.T) {
	tests := []struct {
		name    string
		msg     string
		cause   error
		wantMsg string
	}{
		{
			name:    "marshal error",
			msg:     "failed to marshal",
			cause:   errors.New("json error"),
			wantMsg: "failed to marshal: json error",
		},
		{
			name:    "unmarshal error",
			msg:     "failed to unmarshal",
			cause:   errors.New("invalid json"),
			wantMsg: "failed to unmarshal: invalid json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ErrSerialization(tt.msg, tt.cause)
			if err.Error() != tt.wantMsg {
				t.Errorf("ErrSerialization() = %v, want %v", err.Error(), tt.wantMsg)
			}

			// Check type
			cacheErr, ok := err.(*CacheError)
			if !ok {
				t.Error("ErrSerialization() should return *CacheError")
			}
			if cacheErr.Type != ErrTypeSerialization {
				t.Errorf("ErrSerialization().Type = %v, want %v", cacheErr.Type, ErrTypeSerialization)
			}

			// Check Unwrap
			if cacheErr.Unwrap() != tt.cause {
				t.Errorf("ErrSerialization().Unwrap() = %v, want %v", cacheErr.Unwrap(), tt.cause)
			}
		})
	}
}

// TestErrTimeout tests ErrTimeout constructor
func TestErrTimeout(t *testing.T) {
	tests := []struct {
		name    string
		msg     string
		wantMsg string
	}{
		{
			name:    "L1 timeout",
			msg:     "L1 cache timeout",
			wantMsg: "L1 cache timeout",
		},
		{
			name:    "L2 timeout",
			msg:     "L2 redis timeout",
			wantMsg: "L2 redis timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ErrTimeout(tt.msg)
			if err.Error() != tt.wantMsg {
				t.Errorf("ErrTimeout() = %v, want %v", err.Error(), tt.wantMsg)
			}

			// Check type
			cacheErr, ok := err.(*CacheError)
			if !ok {
				t.Error("ErrTimeout() should return *CacheError")
			}
			if cacheErr.Type != ErrTypeTimeout {
				t.Errorf("ErrTimeout().Type = %v, want %v", cacheErr.Type, ErrTypeTimeout)
			}
		})
	}
}

// TestPredefinedErrors tests predefined error variables
func TestPredefinedErrors(t *testing.T) {
	t.Run("ErrNotFound", func(t *testing.T) {
		if ErrNotFound.Message != "cache key not found" {
			t.Errorf("ErrNotFound.Message = %v, want 'cache key not found'", ErrNotFound.Message)
		}
		if ErrNotFound.Type != ErrTypeNotFound {
			t.Errorf("ErrNotFound.Type = %v, want %v", ErrNotFound.Type, ErrTypeNotFound)
		}
	})

	t.Run("ErrConnectionFailed", func(t *testing.T) {
		if ErrConnectionFailed.Message != "cache connection failed" {
			t.Errorf("ErrConnectionFailed.Message = %v, want 'cache connection failed'", ErrConnectionFailed.Message)
		}
		if ErrConnectionFailed.Type != ErrTypeConnectionError {
			t.Errorf("ErrConnectionFailed.Type = %v, want %v", ErrConnectionFailed.Type, ErrTypeConnectionError)
		}
	})
}
