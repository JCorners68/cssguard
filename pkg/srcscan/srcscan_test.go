package srcscan

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple classes",
			input:    "flex space-y-2 md:hover:bg-red-500",
			expected: []string{"flex", "space-y-2", "md:hover:bg-red-500"},
		},
		{
			name:     "tailwind arbitrary values",
			input:    "w-[100px] bg-[#ff0000] text-[14px]",
			expected: []string{"w-[100px]", "bg-[#ff0000]", "text-[14px]"},
		},
		{
			name:     "negative margins",
			input:    "-mt-4 -translate-x-1/2",
			expected: []string{"-mt-4", "-translate-x-1/2"},
		},
		{
			name:     "skip long tokens",
			input:    "flex " + string(make([]byte, 150)),
			expected: []string{"flex"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			classes := make(map[string]struct{})
			extractTokens(tt.input, classes)

			for _, exp := range tt.expected {
				if _, ok := classes[exp]; !ok {
					t.Errorf("expected token %q not found", exp)
				}
			}

			if len(classes) != len(tt.expected) {
				t.Errorf("got %d tokens, expected %d", len(classes), len(tt.expected))
			}
		})
	}
}

func TestScanFile_TSX(t *testing.T) {
	// Create a temp TSX file
	content := `
import React from 'react';

export function Button() {
  return (
    <button className="flex space-y-2 md:hover:bg-red-500">
      Click me
    </button>
  );
}

export function Card() {
  return <div class="p-4 rounded-lg shadow">{children}</div>;
}
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.tsx")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	s := New(DefaultOptions())
	classes, err := s.scanFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"flex", "space-y-2", "md:hover:bg-red-500",
		"p-4", "rounded-lg", "shadow",
	}

	for _, exp := range expected {
		if _, ok := classes[exp]; !ok {
			t.Errorf("expected class %q not found", exp)
		}
	}
}

func TestScanFile_ClsxHelper(t *testing.T) {
	content := `
import { clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';

const className = clsx("a b", condition && "c");
const merged = twMerge("flex flex-col");
const named = classnames("p-4 m-2");
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.ts")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	s := New(DefaultOptions())
	classes, err := s.scanFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	// Should extract "a", "b" from clsx - NOT "c" (conditional)
	// Should extract "flex", "flex-col" from twMerge
	// Should extract "p-4", "m-2" from classnames
	expected := []string{"a", "b", "flex", "flex-col", "p-4", "m-2"}
	notExpected := []string{"c"} // conditional should NOT be extracted

	for _, exp := range expected {
		if _, ok := classes[exp]; !ok {
			t.Errorf("expected class %q not found", exp)
		}
	}

	for _, ne := range notExpected {
		if _, ok := classes[ne]; ok {
			t.Errorf("class %q should NOT have been extracted (conditional)", ne)
		}
	}
}

func TestScanFile_SkipsTemplateLiterals(t *testing.T) {
	// Use string concatenation to avoid Go raw string issues with backticks
	content := "const dynamic = `flex ${condition ? 'hidden' : 'block'}`;\n" +
		"const safe = className=\"static-class\";\n"
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.ts")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	s := New(DefaultOptions())
	classes, err := s.scanFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	// Should find static-class but skip the template literal line
	if _, ok := classes["static-class"]; !ok {
		t.Error("expected static-class")
	}

	// Should not extract from template literals
	if _, ok := classes["hidden"]; ok {
		t.Error("should not extract from template literals")
	}
}

func TestScanDir_ExcludesNodeModules(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a source file
	srcDir := filepath.Join(tmpDir, "src")
	if err := os.Mkdir(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	srcFile := filepath.Join(srcDir, "app.tsx")
	if err := os.WriteFile(srcFile, []byte(`<div className="from-src">`), 0644); err != nil {
		t.Fatal(err)
	}

	// Create node_modules with a file
	nmDir := filepath.Join(tmpDir, "node_modules")
	if err := os.Mkdir(nmDir, 0755); err != nil {
		t.Fatal(err)
	}
	nmFile := filepath.Join(nmDir, "lib.js")
	if err := os.WriteFile(nmFile, []byte(`<div className="from-node-modules">`), 0644); err != nil {
		t.Fatal(err)
	}

	s := New(DefaultOptions())
	classes, err := s.scanDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Should find from-src
	if _, ok := classes["from-src"]; !ok {
		t.Error("expected from-src from src directory")
	}

	// Should NOT find from-node-modules
	if _, ok := classes["from-node-modules"]; ok {
		t.Error("should not scan node_modules")
	}
}

func TestScanPaths_NoSrcProvided(t *testing.T) {
	s := New(DefaultOptions())
	classes, err := s.ScanPaths(nil)
	if err != nil {
		t.Fatal(err)
	}

	if len(classes) != 0 {
		t.Error("expected empty result when no paths provided")
	}
}

func TestParseExtensions(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", DefaultExtensions},
		{".js,.ts", []string{".js", ".ts"}},
		{"js,ts", []string{".js", ".ts"}},
		{" .jsx , .tsx ", []string{".jsx", ".tsx"}},
	}

	for _, tt := range tests {
		result := ParseExtensions(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("ParseExtensions(%q): got %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestParseExcludes(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", DefaultExcludes},
		{"node_modules,dist", []string{"node_modules", "dist"}},
		{" .next , build ", []string{".next", "build"}},
	}

	for _, tt := range tests {
		result := ParseExcludes(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("ParseExcludes(%q): got %v, want %v", tt.input, result, tt.expected)
		}
	}
}
