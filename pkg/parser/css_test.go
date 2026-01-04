package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFromFile(t *testing.T) {
	tests := []struct {
		name     string
		css      string
		expected []string
	}{
		{
			name:     "basic classes",
			css:      ".foo { color: red; } .bar { color: blue; }",
			expected: []string{"foo", "bar"},
		},
		{
			name:     "hyphenated classes",
			css:      ".text-gray-500 { color: #6b7280; } .bg-red-100 { background: #fee2e2; }",
			expected: []string{"text-gray-500", "bg-red-100"},
		},
		{
			name:     "pseudo-classes",
			css:      ".btn:hover { opacity: 0.8; } .link:focus { outline: none; }",
			expected: []string{"btn", "link"},
		},
		{
			name:     "combined selectors",
			css:      ".card .card-body { padding: 1rem; }",
			expected: []string{"card", "card-body"},
		},
		{
			name:     "escaped characters tailwind",
			css:      `.md\:flex { display: flex; } .hover\:bg-blue-500:hover { background: blue; }`,
			expected: []string{"md:flex", "hover:bg-blue-500"}, // Parser unescapes colons
		},
		{
			name:     "negative classes",
			css:      ".-mt-4 { margin-top: -1rem; }",
			expected: []string{"-mt-4"},
		},
		{
			name:     "underscore classes",
			css:      "._hidden { display: none; } .__container { width: 100%; }",
			expected: []string{"_hidden", "__container"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.css")
			if err := os.WriteFile(tmpFile, []byte(tt.css), 0644); err != nil {
				t.Fatal(err)
			}

			classes, err := ParseFromFile(tmpFile)
			if err != nil {
				t.Fatal(err)
			}

			classSet := make(map[string]bool)
			for _, c := range classes {
				classSet[c] = true
			}

			for _, exp := range tt.expected {
				if !classSet[exp] {
					t.Errorf("expected class %q not found in %v", exp, classes)
				}
			}
		})
	}
}

func TestParseFromDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create main.css
	mainCSS := ".flex { display: flex; } .hidden { display: none; }"
	if err := os.WriteFile(filepath.Join(tmpDir, "main.css"), []byte(mainCSS), 0644); err != nil {
		t.Fatal(err)
	}

	// Create utils.css
	utilsCSS := ".p-4 { padding: 1rem; } .m-2 { margin: 0.5rem; }"
	if err := os.WriteFile(filepath.Join(tmpDir, "utils.css"), []byte(utilsCSS), 0644); err != nil {
		t.Fatal(err)
	}

	// Create non-css file (should be ignored)
	if err := os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("not css"), 0644); err != nil {
		t.Fatal(err)
	}

	classes, err := ParseFromDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"flex", "hidden", "p-4", "m-2"}
	for _, exp := range expected {
		if _, ok := classes[exp]; !ok {
			t.Errorf("expected class %q not found", exp)
		}
	}

	if len(classes) != 4 {
		t.Errorf("expected 4 classes, got %d", len(classes))
	}
}

func TestParseMinified(t *testing.T) {
	// Minified CSS typically has no whitespace
	css := ".a{color:red}.b{color:blue}.c,.d{margin:0}"
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "min.css")
	if err := os.WriteFile(tmpFile, []byte(css), 0644); err != nil {
		t.Fatal(err)
	}

	classes, err := ParseFromFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"a", "b", "c", "d"}
	classSet := make(map[string]bool)
	for _, c := range classes {
		classSet[c] = true
	}

	for _, exp := range expected {
		if !classSet[exp] {
			t.Errorf("expected class %q not found in minified CSS", exp)
		}
	}
}
