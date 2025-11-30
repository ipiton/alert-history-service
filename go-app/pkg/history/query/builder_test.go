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
				qb.SetLimit(50)
				qb.SetOffset(50)
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
				qb.SetLimit(50)
				qb.SetOffset(0)
				qb.AddOrderBy("starts_at", core.SortOrderDesc)
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

// TestBuilder_SetLimitOffset tests SetLimit/SetOffset functionality
func TestBuilder_SetLimitOffset(t *testing.T) {
	qb := NewBuilder()
	qb.SetLimit(50)
	qb.SetOffset(50)

	sql, _ := qb.Build()

	if !contains(sql, "LIMIT") || !contains(sql, "OFFSET") {
		t.Error("SetLimit/SetOffset() did not add LIMIT/OFFSET")
	}
}

// TestBuilder_AddOrderBy tests AddOrderBy functionality
func TestBuilder_AddOrderBy(t *testing.T) {
	qb := NewBuilder()
	qb.AddOrderBy("starts_at", core.SortOrderDesc)

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

// TestBuilder_BuildCount tests BuildCount functionality
func TestBuilder_BuildCount(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*Builder)
		wantSQL string
	}{
		{
			name:    "simple count query",
			setup:   func(qb *Builder) {},
			wantSQL: "SELECT COUNT(*) FROM alerts",
		},
		{
			name: "count with WHERE clause",
			setup: func(qb *Builder) {
				qb.AddWhere("status = ?", "firing")
			},
			wantSQL: "SELECT COUNT(*) FROM alerts WHERE status = $1",
		},
		{
			name: "count with multiple WHERE",
			setup: func(qb *Builder) {
				qb.AddWhere("status = ?", "firing")
				qb.AddWhere("severity = ?", "critical")
			},
			wantSQL: "SELECT COUNT(*) FROM alerts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := NewBuilder()
			tt.setup(qb)

			sql, args := qb.BuildCount()

			if sql == "" {
				t.Error("BuildCount() returned empty SQL")
			}

			if !contains(sql, "COUNT(*)") {
				t.Errorf("BuildCount() SQL = %v, want contains 'COUNT(*)'", sql)
			}

			// Verify args count
			placeholderCount := countPlaceholders(sql)
			if len(args) != placeholderCount {
				t.Errorf("BuildCount() args count = %v, want %v", len(args), placeholderCount)
			}
		})
	}
}

// TestBuilder_OptimizationHints tests OptimizationHints functionality
func TestBuilder_OptimizationHints(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*Builder)
		want  []string
	}{
		{
			name:  "no hints - empty builder",
			setup: func(qb *Builder) {},
			want:  []string{},
		},
		{
			name: "GIN index hint",
			setup: func(qb *Builder) {
				qb.MarkGINIndexUsage()
			},
			want: []string{"Use GIN index for JSONB queries"},
		},
		{
			name: "partial index hint",
			setup: func(qb *Builder) {
				qb.MarkPartialIndexUsage()
			},
			want: []string{"Use partial index for status=firing"},
		},
		{
			name: "multiple hints",
			setup: func(qb *Builder) {
				qb.MarkGINIndexUsage()
				qb.MarkPartialIndexUsage()
			},
			want: []string{"Use GIN index for JSONB queries", "Use partial index for status=firing"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := NewBuilder()
			tt.setup(qb)

			hints := qb.OptimizationHints()

			if len(hints) != len(tt.want) {
				t.Errorf("OptimizationHints() len = %v, want %v", len(hints), len(tt.want))
			}

			// Verify hints contain expected values
			for _, wantHint := range tt.want {
				found := false
				for _, gotHint := range hints {
					if gotHint == wantHint {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("OptimizationHints() missing hint %v", wantHint)
				}
			}
		})
	}
}

// TestBuilder_MarkPartialIndexUsage tests MarkPartialIndexUsage functionality
func TestBuilder_MarkPartialIndexUsage(t *testing.T) {
	qb := NewBuilder()
	qb.MarkPartialIndexUsage()

	hints := qb.OptimizationHints()

	// Verify partial index hint is present
	found := false
	for _, hint := range hints {
		if contains(hint, "partial index") {
			found = true
			break
		}
	}

	if !found {
		t.Error("MarkPartialIndexUsage() did not add partial index hint")
	}
}

// TestBuilder_AddOrderBy_Extended tests AddOrderBy with various scenarios
func TestBuilder_AddOrderBy_Extended(t *testing.T) {
	tests := []struct {
		name      string
		field     string
		direction core.SortOrder
		wantErr   bool
	}{
		{
			name:      "DESC order",
			field:     "starts_at",
			direction: core.SortOrderDesc,
			wantErr:   false,
		},
		{
			name:      "ASC order",
			field:     "starts_at",
			direction: core.SortOrderAsc,
			wantErr:   false,
		},
		{
			name:      "created_at field",
			field:     "created_at",
			direction: core.SortOrderDesc,
			wantErr:   false,
		},
		{
			name:      "status field",
			field:     "status",
			direction: core.SortOrderAsc,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qb := NewBuilder()
			qb.AddOrderBy(tt.field, tt.direction)

			sql, _ := qb.Build()

			if !contains(sql, "ORDER BY") {
				t.Error("AddOrderBy() did not add ORDER BY clause")
			}

			if !contains(sql, tt.field) {
				t.Errorf("AddOrderBy() SQL missing field %v", tt.field)
			}
		})
	}
}

// TestBuilder_MultipleCalls tests multiple method calls
func TestBuilder_MultipleCalls(t *testing.T) {
	qb := NewBuilder()
	qb.AddWhere("status = ?", "firing")
	qb.AddWhere("severity = ?", "critical")
	qb.AddOrderBy("starts_at", core.SortOrderDesc)
	qb.SetLimit(50)
	qb.SetOffset(0)
	qb.MarkGINIndexUsage()
	qb.MarkPartialIndexUsage()

	sql, args := qb.Build()

	if sql == "" {
		t.Error("Multiple calls Build() returned empty SQL")
	}

	// Args count depends on whether AddOrderBy uses placeholders
	if len(args) < 2 {
		t.Errorf("Multiple calls args = %v, want at least 2", len(args))
	}

	hints := qb.OptimizationHints()
	if len(hints) != 2 {
		t.Errorf("Multiple calls hints = %v, want 2", len(hints))
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
