package filters

import (
	"testing"
)

// TestFilterRegistry tests FilterRegistry functionality
func TestFilterRegistry(t *testing.T) {
	registry := NewRegistry(nil)
	
	tests := []struct {
		name    string
		typ     FilterType
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "create status filter",
			typ:  FilterTypeStatus,
			params: map[string]interface{}{
				"values": []string{"firing"},
			},
			wantErr: false,
		},
		{
			name: "create severity filter",
			typ:  FilterTypeSeverity,
			params: map[string]interface{}{
				"values": []string{"critical"},
			},
			wantErr: false,
		},
		{
			name: "create unknown filter",
			typ:  FilterType("unknown"),
			params: map[string]interface{}{},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := registry.Create(tt.typ, tt.params)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("Registry.Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && filter != nil {
				if filter.Type() != tt.typ {
					t.Errorf("Filter.Type() = %v, want %v", filter.Type(), tt.typ)
				}
			}
		})
	}
}

// TestFilterRegistry_CreateFromQueryParams tests query parameter parsing
func TestFilterRegistry_CreateFromQueryParams(t *testing.T) {
	registry := NewRegistry(nil)
	
	tests := []struct {
		name       string
		queryParams map[string][]string
		wantErr    bool
		wantCount  int
	}{
		{
			name: "parse status filter",
			queryParams: map[string][]string{
				"status": {"firing"},
			},
			wantErr:   false,
			wantCount: 1,
		},
		{
			name: "parse multiple filters",
			queryParams: map[string][]string{
				"status":   {"firing"},
				"severity": {"critical"},
				"namespace": {"production"},
			},
			wantErr:   false,
			wantCount: 3,
		},
		{
			name: "parse invalid status",
			queryParams: map[string][]string{
				"status": {"invalid"},
			},
			wantErr: true,
		},
		{
			name:       "empty query params",
			queryParams: map[string][]string{},
			wantErr:    false,
			wantCount:  0,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filters, err := registry.CreateFromQueryParams(tt.queryParams)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateFromQueryParams() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(filters) != tt.wantCount {
					t.Errorf("CreateFromQueryParams() count = %v, want %v", len(filters), tt.wantCount)
				}
			}
		})
	}
}

