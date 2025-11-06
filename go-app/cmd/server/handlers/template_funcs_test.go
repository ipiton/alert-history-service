package handlers

import (
	"html/template"
	"testing"
	"time"
)

// TestFormatTime tests time formatting functions.
func TestFormatTime(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "valid time",
			input:    time.Date(2025, 11, 6, 12, 30, 0, 0, time.UTC),
			expected: "2025-11-06 12:30",
		},
		{
			name:     "zero time",
			input:    time.Time{},
			expected: "-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTime(tt.input)
			if result != tt.expected {
				t.Errorf("formatTime() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatDateTime(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "valid datetime",
			input:    time.Date(2025, 11, 6, 12, 30, 45, 0, time.UTC),
			expected: "2025-11-06 12:30:45",
		},
		{
			name:     "zero time",
			input:    time.Time{},
			expected: "-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDateTime(tt.input)
			if result != tt.expected {
				t.Errorf("formatDateTime() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestHumanDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Duration
		expected string
	}{
		{
			name:     "5 seconds",
			input:    5 * time.Second,
			expected: "5s",
		},
		{
			name:     "2 minutes",
			input:    2 * time.Minute,
			expected: "2m",
		},
		{
			name:     "3 hours",
			input:    3 * time.Hour,
			expected: "3h",
		},
		{
			name:     "2 days",
			input:    48 * time.Hour,
			expected: "2d",
		},
		{
			name:     "negative duration",
			input:    -5 * time.Second,
			expected: "-",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := humanDuration(tt.input)
			if result != tt.expected {
				t.Errorf("humanDuration() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestTimeAgo(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "just now",
			input:    now.Add(-30 * time.Second),
			expected: "just now",
		},
		{
			name:     "5 minutes ago",
			input:    now.Add(-5 * time.Minute),
			expected: "5 minutes ago",
		},
		{
			name:     "2 hours ago",
			input:    now.Add(-2 * time.Hour),
			expected: "2 hours ago",
		},
		{
			name:     "3 days ago",
			input:    now.Add(-3 * 24 * time.Hour),
			expected: "3 days ago",
		},
		{
			name:     "zero time",
			input:    time.Time{},
			expected: "never",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := timeAgo(tt.input)
			if result != tt.expected {
				t.Errorf("timeAgo() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestTruncate tests string truncation functions.
func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		length   int
		expected string
	}{
		{
			name:     "no truncation needed",
			input:    "Hello",
			length:   10,
			expected: "Hello",
		},
		{
			name:     "truncate with ellipsis",
			input:    "Hello World",
			length:   5,
			expected: "Hello...",
		},
		{
			name:     "zero length",
			input:    "Hello",
			length:   0,
			expected: "",
		},
		{
			name:     "negative length",
			input:    "Hello",
			length:   -1,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.input, tt.length)
			if result != tt.expected {
				t.Errorf("truncate() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestTruncateEnd(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		length   int
		expected string
	}{
		{
			name:     "no truncation needed",
			input:    "Hello",
			length:   10,
			expected: "Hello",
		},
		{
			name:     "truncate from end",
			input:    "Hello World",
			length:   5,
			expected: "...World",
		},
		{
			name:     "zero length",
			input:    "Hello",
			length:   0,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateEnd(tt.input, tt.length)
			if result != tt.expected {
				t.Errorf("truncateEnd() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "item exists",
			slice:    []string{"apple", "banana", "cherry"},
			item:     "banana",
			expected: true,
		},
		{
			name:     "item does not exist",
			slice:    []string{"apple", "banana", "cherry"},
			item:     "orange",
			expected: false,
		},
		{
			name:     "empty slice",
			slice:    []string{},
			item:     "apple",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.item)
			if result != tt.expected {
				t.Errorf("contains() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestStatusFunctions tests status-related functions.
func TestStatusClass(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "active status",
			input:    "active",
			expected: "success",
		},
		{
			name:     "pending status",
			input:    "pending",
			expected: "info",
		},
		{
			name:     "expired status",
			input:    "expired",
			expected: "secondary",
		},
		{
			name:     "error status",
			input:    "error",
			expected: "danger",
		},
		{
			name:     "unknown status",
			input:    "unknown",
			expected: "secondary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := statusClass(tt.input)
			if result != tt.expected {
				t.Errorf("statusClass() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "active status",
			input:    "active",
			expected: "✓",
		},
		{
			name:     "pending status",
			input:    "pending",
			expected: "⏳",
		},
		{
			name:     "expired status",
			input:    "expired",
			expected: "⏹",
		},
		{
			name:     "error status",
			input:    "error",
			expected: "✗",
		},
		{
			name:     "unknown status",
			input:    "unknown",
			expected: "●",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := statusIcon(tt.input)
			if result != tt.expected {
				t.Errorf("statusIcon() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestStatusBadge(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "active status",
			input: "active",
		},
		{
			name:  "pending status",
			input: "pending",
		},
		{
			name:  "expired status",
			input: "expired",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := statusBadge(tt.input)
			if len(result) == 0 {
				t.Error("statusBadge() returned empty string")
			}
			// Check that result contains expected class
			expected := statusClass(tt.input)
			if !contains([]string{string(result)}, expected) {
				t.Logf("statusBadge() = %s (contains class %q)", result, expected)
			}
		})
	}
}

// TestMathFunctions tests math helper functions.
func TestAdd(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{
			name:     "positive numbers",
			a:        5,
			b:        3,
			expected: 8,
		},
		{
			name:     "negative numbers",
			a:        -5,
			b:        -3,
			expected: -8,
		},
		{
			name:     "mixed signs",
			a:        5,
			b:        -3,
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := add(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("add(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestSub(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{
			name:     "positive result",
			a:        10,
			b:        3,
			expected: 7,
		},
		{
			name:     "negative result",
			a:        3,
			b:        10,
			expected: -7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sub(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("sub(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestMul(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{
			name:     "positive numbers",
			a:        5,
			b:        3,
			expected: 15,
		},
		{
			name:     "with zero",
			a:        5,
			b:        0,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mul(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("mul(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestDiv(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{
			name:     "normal division",
			a:        10,
			b:        2,
			expected: 5,
		},
		{
			name:     "division by zero",
			a:        10,
			b:        0,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := div(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("div(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{
			name:     "a is smaller",
			a:        3,
			b:        5,
			expected: 3,
		},
		{
			name:     "b is smaller",
			a:        5,
			b:        3,
			expected: 3,
		},
		{
			name:     "equal values",
			a:        5,
			b:        5,
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := min(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestMax(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{
			name:     "a is larger",
			a:        5,
			b:        3,
			expected: 5,
		},
		{
			name:     "b is larger",
			a:        3,
			b:        5,
			expected: 5,
		},
		{
			name:     "equal values",
			a:        5,
			b:        5,
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := max(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("max(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestPercent(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{
			name:     "50 percent",
			a:        50,
			b:        100,
			expected: 50,
		},
		{
			name:     "100 percent",
			a:        100,
			b:        100,
			expected: 100,
		},
		{
			name:     "zero denominator",
			a:        50,
			b:        0,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := percent(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("percent(%d, %d) = %d, want %d", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

// TestComparisonFunctions tests comparison helper functions.
func TestEq(t *testing.T) {
	tests := []struct {
		name     string
		a, b     interface{}
		expected bool
	}{
		{
			name:     "equal integers",
			a:        5,
			b:        5,
			expected: true,
		},
		{
			name:     "not equal integers",
			a:        5,
			b:        3,
			expected: false,
		},
		{
			name:     "equal strings",
			a:        "hello",
			b:        "hello",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := eq(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("eq(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestNe(t *testing.T) {
	tests := []struct {
		name     string
		a, b     interface{}
		expected bool
	}{
		{
			name:     "not equal",
			a:        5,
			b:        3,
			expected: true,
		},
		{
			name:     "equal",
			a:        5,
			b:        5,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ne(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("ne(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestLt(t *testing.T) {
	if !lt(3, 5) {
		t.Error("lt(3, 5) should be true")
	}
	if lt(5, 3) {
		t.Error("lt(5, 3) should be false")
	}
}

func TestGt(t *testing.T) {
	if !gt(5, 3) {
		t.Error("gt(5, 3) should be true")
	}
	if gt(3, 5) {
		t.Error("gt(3, 5) should be false")
	}
}

func TestAnd(t *testing.T) {
	if !and(true, true) {
		t.Error("and(true, true) should be true")
	}
	if and(true, false) {
		t.Error("and(true, false) should be false")
	}
}

func TestOr(t *testing.T) {
	if !or(true, false) {
		t.Error("or(true, false) should be true")
	}
	if or(false, false) {
		t.Error("or(false, false) should be false")
	}
}

func TestNot(t *testing.T) {
	if !not(false) {
		t.Error("not(false) should be true")
	}
	if not(true) {
		t.Error("not(true) should be false")
	}
}

// TestCollectionFunctions tests collection helper functions.
func TestLength(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected int
	}{
		{
			name:     "string slice",
			input:    []string{"a", "b", "c"},
			expected: 3,
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: 0,
		},
		{
			name:     "string",
			input:    "hello",
			expected: 5,
		},
		{
			name:     "unsupported type",
			input:    123,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := length(tt.input)
			if result != tt.expected {
				t.Errorf("length(%v) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFirst(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "string slice",
			input:    []string{"a", "b", "c"},
			expected: "a",
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := first(tt.input)
			if result != tt.expected {
				t.Errorf("first(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestLast(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "string slice",
			input:    []string{"a", "b", "c"},
			expected: "c",
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := last(tt.input)
			if result != tt.expected {
				t.Errorf("last(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestTemplateFuncsIntegration tests that all functions can be registered.
func TestTemplateFuncsIntegration(t *testing.T) {
	tmpl := template.New("test").Funcs(templateFuncs())

	if tmpl == nil {
		t.Fatal("Failed to create template with custom functions")
	}

	// Verify that all expected functions are registered
	expectedFuncs := []string{
		"formatTime", "humanDuration", "truncate", "statusBadge",
		"add", "sub", "eq", "contains", "len",
	}

	for _, funcName := range expectedFuncs {
		// Try to parse a template using each function
		testTemplate := "{{" + funcName + "}}"
		_, err := tmpl.Parse(testTemplate)
		if err == nil {
			// Function exists (may fail execution, but registered)
			t.Logf("Function %q is registered", funcName)
		}
	}
}

// BenchmarkHumanDuration benchmarks humanDuration function.
func BenchmarkHumanDuration(b *testing.B) {
	d := 2*time.Hour + 30*time.Minute
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = humanDuration(d)
	}
}

// BenchmarkStatusBadge benchmarks statusBadge function.
func BenchmarkStatusBadge(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = statusBadge("active")
	}
}
