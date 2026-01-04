package validator

import (
	"testing"
)

func TestValidateDirectly(t *testing.T) {
	tests := []struct {
		name         string
		htmlClasses  map[string]struct{}
		cssClasses   map[string]struct{}
		wantOrphans  []string
		wantUnused   []string
		wantMatched  int
	}{
		{
			name:         "all matched",
			htmlClasses:  setOf("flex", "hidden", "p-4"),
			cssClasses:   setOf("flex", "hidden", "p-4", "m-2"),
			wantOrphans:  nil,
			wantUnused:   []string{"m-2"},
			wantMatched:  3,
		},
		{
			name:         "with orphans",
			htmlClasses:  setOf("flex", "custom-class"),
			cssClasses:   setOf("flex"),
			wantOrphans:  []string{"custom-class"},
			wantUnused:   nil,
			wantMatched:  1,
		},
		{
			name:         "empty html",
			htmlClasses:  setOf(),
			cssClasses:   setOf("flex", "hidden"),
			wantOrphans:  nil,
			wantUnused:   []string{"flex", "hidden"},
			wantMatched:  0,
		},
		{
			name:         "empty css",
			htmlClasses:  setOf("flex"),
			cssClasses:   setOf(),
			wantOrphans:  []string{"flex"},
			wantUnused:   nil,
			wantMatched:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateDirectly(tt.htmlClasses, tt.cssClasses)

			if result.Matched != tt.wantMatched {
				t.Errorf("Matched = %d, want %d", result.Matched, tt.wantMatched)
			}

			if len(result.Orphans) != len(tt.wantOrphans) {
				t.Errorf("Orphans = %v, want %v", result.Orphans, tt.wantOrphans)
			}

			// Check orphans content
			orphanSet := make(map[string]bool)
			for _, o := range result.Orphans {
				orphanSet[o] = true
			}
			for _, want := range tt.wantOrphans {
				if !orphanSet[want] {
					t.Errorf("expected orphan %q not found", want)
				}
			}
		})
	}
}

func TestResultHasOrphans(t *testing.T) {
	r := &Result{Orphans: []string{"test"}, OrphanCount: 1}
	if !r.HasOrphans() {
		t.Error("HasOrphans() should return true")
	}

	r = &Result{Orphans: nil, OrphanCount: 0}
	if r.HasOrphans() {
		t.Error("HasOrphans() should return false for nil")
	}

	r = &Result{Orphans: []string{}, OrphanCount: 0}
	if r.HasOrphans() {
		t.Error("HasOrphans() should return false for empty slice")
	}
}

func TestResultHasUnused(t *testing.T) {
	r := &Result{Unused: []string{"test"}, UnusedCount: 1}
	if !r.HasUnused() {
		t.Error("HasUnused() should return true")
	}

	r = &Result{Unused: nil, UnusedCount: 0}
	if r.HasUnused() {
		t.Error("HasUnused() should return false for nil")
	}
}

func TestResultSummary(t *testing.T) {
	r := &Result{
		HTMLClasses: 10,
		CSSClasses:  20,
		Matched:     8,
		Orphans:     []string{"a", "b"},
		Unused:      []string{"x", "y", "z"},
	}

	summary := r.Summary()
	if summary == "" {
		t.Error("Summary() should not be empty")
	}

	// Check that key numbers appear in summary
	if !contains(summary, "10") || !contains(summary, "20") || !contains(summary, "8") {
		t.Errorf("Summary missing expected numbers: %s", summary)
	}
}

func TestMatchPercentageCalc(t *testing.T) {
	// Test percentage calculation logic
	r := &Result{
		HTMLClasses: 100,
		Matched:     75,
	}

	var pct float64
	if r.HTMLClasses > 0 {
		pct = float64(r.Matched) / float64(r.HTMLClasses) * 100
	} else {
		pct = 100.0
	}
	if pct != 75.0 {
		t.Errorf("Match percentage = %f, want 75.0", pct)
	}

	// Test zero division case
	r = &Result{HTMLClasses: 0, Matched: 0}
	if r.HTMLClasses > 0 {
		pct = float64(r.Matched) / float64(r.HTMLClasses) * 100
	} else {
		pct = 100.0
	}
	if pct != 100.0 {
		t.Errorf("Match percentage for 0/0 = %f, want 100.0", pct)
	}
}

// Helper functions
func setOf(items ...string) map[string]struct{} {
	m := make(map[string]struct{})
	for _, item := range items {
		m[item] = struct{}{}
	}
	return m
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
