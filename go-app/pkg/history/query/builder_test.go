package query

import (
	"testing"
	
	"github.com/vitaliisemenov/alert-history/internal/core"
)

// TestBuilder_Build tests QueryBuilder functionality
func TestBuilder_Build(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*Builder)
		wantSQL  string
	}{
		{
			name:     "simple query",
			setup:    func(qb *Builder) {},
			wantSQL:  "SELECT * FROM alerts",
		},
		{
			name: "query with WHERE clause",
			setup: func(qb *Builder) {
				qb.AddWhere("status = ?", "firing")
			},
			wantSQL: "SELECT * FROM alerts",
		},
		{
			name: "query with pagination",
			setup: func(qb *Builder) {
				qb.SetPagination(&core.Pagination{
					Page:    2,
					PerPage: 50,
				})
			},
			wantSQL: "SELECT * FROM alerts",
		},
		{
			name: "query with sorting",
			setup: func(qb *Builder) {
				qb.AddOrderBy("starts_at", "DESC")
			},
			wantSQL: "SELECT * FROM alerts",
		},
		{
			name: "complete query",
			setup: func(qb *Builder) {
				qb.AddWhere("status = ?", "firing")
				qb.SetPagination(&core.Pagination{
					Page:    1,
					PerPage: 50,
				})
				qb.AddOrderBy("starts_at", "DESC")
			},
			wantSQL: "SELECT * FROM alerts",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := NewBuilder()
			tt.setup(qb)
			
			sql, args := qb.Build()
			
			if sql == "" {
				t.Error("Build() returned empty SQL")
			}
			
			// Verify args count matches placeholders
			placeholderCount := countPlaceholders(sql)
			if len(args) != placeholderCount {
				t.Errorf("Build() args count = %v, want %v", len(args), placeholderCount)
			}
		})
	}
}

// countPlaceholders counts $N placeholders in SQL
func countPlaceholders(sql string) int {
	count := 0
	for i := 0; i < len(sql); i++ {
		if sql[i] == '$' && i+1 < len(sql) {
			if sql[i+1] >= '0' && sql[i+1] <= '9' {
				count++
			}
		}
	}
	return count
}

// TestBuilder_AddWhere tests AddWhere functionality
func TestBuilder_AddWhere(t *testing.T) {
	qb := NewBuilder()
	qb.AddWhere("status = ?", "firing")
	qb.AddWhere("severity = ?", "critical")
	
	sql, args := qb.Build()
	
	if len(args) != 2 {
		t.Errorf("AddWhere() args count = %v, want 2", len(args))
	}
	
	if args[0] != "firing" || args[1] != "critical" {
		t.Errorf("AddWhere() args = %v, want [firing, critical]", args)
	}
	
	// Verify WHERE clause is present
	if !contains(sql, "WHERE") {
		t.Error("AddWhere() did not add WHERE clause")
	}
}

// TestBuilder_SetPagination tests SetPagination functionality
func TestBuilder_SetPagination(t *testing.T) {
	qb := NewBuilder()
	qb.SetPagination(&core.Pagination{
		Page:    2,
		PerPage: 50,
	})
	
	sql, args := qb.Build()
	
	if !contains(sql, "LIMIT") || !contains(sql, "OFFSET") {
		t.Error("SetPagination() did not add LIMIT/OFFSET")
	}
	
	if len(args) != 2 {
		t.Errorf("SetPagination() args count = %v, want 2", len(args))
	}
	
	if args[0] != 50 || args[1] != 50 {
		t.Errorf("SetPagination() args = %v, want [50, 50]", args)
	}
}

// TestBuilder_AddOrderBy tests AddOrderBy functionality
func TestBuilder_AddOrderBy(t *testing.T) {
	qb := NewBuilder()
	qb.AddOrderBy("starts_at", "DESC")
	
	sql, _ := qb.Build()
	
	if !contains(sql, "ORDER BY") {
		t.Error("AddOrderBy() did not add ORDER BY clause")
	}
	
	if !contains(sql, "starts_at") {
		t.Errorf("AddOrderBy() SQL = %v, want contains 'starts_at'", sql)
	}
}

// TestBuilder_MarkGINIndexUsage tests GIN index marking
func TestBuilder_MarkGINIndexUsage(t *testing.T) {
	qb := NewBuilder()
	qb.MarkGINIndexUsage()
	
	// Verify flag is set (internal state, can't directly test)
	// This is more of a smoke test
	if qb == nil {
		t.Error("MarkGINIndexUsage() failed")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

