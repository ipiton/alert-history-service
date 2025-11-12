package middleware

import (
	"testing"
)

func TestPathNormalizer_NormalizePath(t *testing.T) {
	normalizer := NewPathNormalizer()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "UUID in path",
			input:    "/api/alerts/123e4567-e89b-12d3-a456-426614174000",
			expected: "/api/alerts/:id",
		},
		{
			name:     "Multiple UUIDs",
			input:    "/api/alerts/123e4567-e89b-12d3-a456-426614174000/comments/987fcdeb-51a2-43f7-8a9b-123456789abc",
			expected: "/api/alerts/:id/comments/:id",
		},
		{
			name:     "Numeric ID",
			input:    "/api/alerts/12345",
			expected: "/api/alerts/:id",
		},
		{
			name:     "Multiple numeric IDs",
			input:    "/api/alerts/12345/comments/67890",
			expected: "/api/alerts/:id/comments/:id",
		},
		{
			name:     "Mixed UUID and numeric ID",
			input:    "/api/alerts/123e4567-e89b-12d3-a456-426614174000/actions/12345",
			expected: "/api/alerts/:id/actions/:id",
		},
		{
			name:     "Static path unchanged",
			input:    "/api/health",
			expected: "/api/health",
		},
		{
			name:     "Static path with segments",
			input:    "/api/v1/alerts/recent",
			expected: "/api/v1/alerts/recent",
		},
		{
			name:     "Long numeric ID (int64)",
			input:    "/api/alerts/9223372036854775807",
			expected: "/api/alerts/:id",
		},
		{
			name:     "Short numeric ID",
			input:    "/api/alerts/1",
			expected: "/api/alerts/:id",
		},
		{
			name:     "Path with trailing slash",
			input:    "/api/alerts/12345/",
			expected: "/api/alerts/:id",
		},
		{
			name:     "Root path",
			input:    "/",
			expected: "/",
		},
		{
			name:     "Empty path",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizer.NormalizePath(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizePath(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func BenchmarkPathNormalizer_NormalizePath(b *testing.B) {
	normalizer := NewPathNormalizer()
	path := "/api/alerts/123e4567-e89b-12d3-a456-426614174000/comments/12345"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = normalizer.NormalizePath(path)
	}
}

func BenchmarkPathNormalizer_NormalizePath_Static(b *testing.B) {
	normalizer := NewPathNormalizer()
	path := "/api/health"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = normalizer.NormalizePath(path)
	}
}
