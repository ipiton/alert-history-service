package filters

import (
	"testing"
	"time"

	"github.com/vitaliisemenov/alert-history/pkg/history/query"
)

// TestStatusFilter tests StatusFilter functionality
func TestStatusFilter(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid firing status",
			params: map[string]interface{}{
				"values": []string{"firing"},
			},
			wantErr: false,
		},
		{
			name: "valid resolved status",
			params: map[string]interface{}{
				"values": []string{"resolved"},
			},
			wantErr: false,
		},
		{
			name: "invalid status",
			params: map[string]interface{}{
				"values": []string{"invalid"},
			},
			wantErr: true,
		},
		{
			name: "missing status value",
			params: map[string]interface{}{
				"values": []string{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewStatusFilter(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewStatusFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && filter != nil {
				if err := filter.Validate(); err != nil {
					t.Errorf("Validate() error = %v", err)
				}

				// Test ApplyToQuery
				qb := query.NewBuilder()
				if err := filter.ApplyToQuery(qb); err != nil {
					t.Errorf("ApplyToQuery() error = %v", err)
				}
			}
		})
	}
}

// TestSeverityFilter tests SeverityFilter functionality
func TestSeverityFilter(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid single severity",
			params: map[string]interface{}{
				"values": []string{"critical"},
			},
			wantErr: false,
		},
		{
			name: "valid multiple severities",
			params: map[string]interface{}{
				"values": []string{"critical", "warning"},
			},
			wantErr: false,
		},
		{
			name: "invalid severity",
			params: map[string]interface{}{
				"values": []string{"invalid"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewSeverityFilter(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSeverityFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && filter != nil {
				if err := filter.Validate(); err != nil {
					t.Errorf("Validate() error = %v", err)
				}

				qb := query.NewBuilder()
				if err := filter.ApplyToQuery(qb); err != nil {
					t.Errorf("ApplyToQuery() error = %v", err)
				}
			}
		})
	}
}

// TestNamespaceFilter tests NamespaceFilter functionality
func TestNamespaceFilter(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid single namespace",
			params: map[string]interface{}{
				"values": []string{"production"},
			},
			wantErr: false,
		},
		{
			name: "valid multiple namespaces",
			params: map[string]interface{}{
				"values": []string{"production", "staging"},
			},
			wantErr: false,
		},
		{
			name: "missing namespace",
			params: map[string]interface{}{
				"values": []string{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewNamespaceFilter(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewNamespaceFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && filter != nil {
				if err := filter.Validate(); err != nil {
					t.Errorf("Validate() error = %v", err)
				}

				qb := query.NewBuilder()
				if err := filter.ApplyToQuery(qb); err != nil {
					t.Errorf("ApplyToQuery() error = %v", err)
				}
			}
		})
	}
}

// TestTimeRangeFilter tests TimeRangeFilter functionality
func TestTimeRangeFilter(t *testing.T) {
	now := time.Now()
	from := now.Add(-24 * time.Hour)
	to := now

	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid time range",
			params: map[string]interface{}{
				"from": from.Format(time.RFC3339),
				"to":   to.Format(time.RFC3339),
			},
			wantErr: false,
		},
		{
			name: "valid from only",
			params: map[string]interface{}{
				"from": from.Format(time.RFC3339),
			},
			wantErr: false,
		},
		{
			name: "valid to only",
			params: map[string]interface{}{
				"to": to.Format(time.RFC3339),
			},
			wantErr: false,
		},
		{
			name: "invalid time range (from after to)",
			params: map[string]interface{}{
				"from": to.Format(time.RFC3339),
				"to":   from.Format(time.RFC3339),
			},
			wantErr: true,
		},
		{
			name: "invalid format",
			params: map[string]interface{}{
				"from": "invalid-date",
			},
			wantErr: true,
		},
		{
			name:    "missing both from and to",
			params:  map[string]interface{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewTimeRangeFilter(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTimeRangeFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && filter != nil {
				if err := filter.Validate(); err != nil {
					t.Errorf("Validate() error = %v", err)
				}

				qb := query.NewBuilder()
				if err := filter.ApplyToQuery(qb); err != nil {
					t.Errorf("ApplyToQuery() error = %v", err)
				}
			}
		})
	}
}

// TestFingerprintFilter tests FingerprintFilter functionality
func TestFingerprintFilter(t *testing.T) {
	// Generate valid 64-character hex fingerprint
	validFingerprint := "a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef1234567890"
	
	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid fingerprint",
			params: map[string]interface{}{
				"values": []string{validFingerprint},
			},
			wantErr: false,
		},
		{
			name: "invalid fingerprint length",
			params: map[string]interface{}{
				"values": []string{"short"},
			},
			wantErr: true,
		},
		{
			name: "missing values",
			params: map[string]interface{}{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewFingerprintFilter(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFingerprintFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && filter != nil {
				if err := filter.Validate(); err != nil {
					t.Errorf("Validate() error = %v", err)
				}
			}
		})
	}
}

// TestAlertNameFilter tests AlertNameFilter functionality
func TestAlertNameFilter(t *testing.T) {
	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid alert name",
			params: map[string]interface{}{
				"value": "HighCPUUsage",
			},
			wantErr: false,
		},
		{
			name: "empty alert name",
			params: map[string]interface{}{
				"value": "",
			},
			wantErr: true,
		},
		{
			name: "too long alert name",
			params: map[string]interface{}{
				"value": string(make([]byte, 256)),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewAlertNameFilter(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAlertNameFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && filter != nil {
				if err := filter.Validate(); err != nil {
					t.Errorf("Validate() error = %v", err)
				}
			}
		})
	}
}
