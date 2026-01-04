package main

import (
	"testing"
)

func TestDetectRedundancy(t *testing.T) {
	tests := []struct {
		name       string
		files      map[string]map[string]struct{}
		threshold  float64
		wantCount  int
	}{
		{
			name: "no redundancy",
			files: map[string]map[string]struct{}{
				"a.css": {"foo": {}, "bar": {}},
				"b.css": {"baz": {}, "qux": {}},
			},
			threshold: 80.0,
			wantCount: 0,
		},
		{
			name: "full redundancy",
			files: map[string]map[string]struct{}{
				"main.css":   {"flex": {}, "hidden": {}, "p-4": {}},
				"vendor.css": {"flex": {}, "hidden": {}, "p-4": {}},
			},
			threshold: 80.0,
			wantCount: 2, // Both files are 100% covered by each other
		},
		{
			name: "partial redundancy above threshold",
			files: map[string]map[string]struct{}{
				"main.css":   {"a": {}, "b": {}, "c": {}, "d": {}, "e": {}},
				"vendor.css": {"a": {}, "b": {}, "c": {}, "d": {}}, // 4/4 = 100% covered, but 4/5 = 80% other way
			},
			threshold: 80.0,
			wantCount: 2, // vendor 100% covered by main, main 80% covered by vendor
		},
		{
			name: "partial redundancy below threshold",
			files: map[string]map[string]struct{}{
				"main.css":   {"a": {}, "b": {}, "c": {}},
				"vendor.css": {"a": {}, "x": {}, "y": {}, "z": {}}, // 1/4 = 25% overlap
			},
			threshold: 80.0,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectRedundancy(tt.files, tt.threshold)
			if len(result) != tt.wantCount {
				t.Errorf("detectRedundancy() returned %d items, want %d: %v",
					len(result), tt.wantCount, result)
			}
		})
	}
}

func TestExpandGlob(t *testing.T) {
	// Test with non-existent pattern
	result := expandGlob("/nonexistent/*.css")
	if result != nil && len(result) > 0 {
		t.Errorf("expandGlob() should return nil for non-existent path, got %v", result)
	}
}
