package extractor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractFromDir_SingleFile(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected []string
	}{
		{
			name:     "basic classes",
			html:     `<div class="flex items-center">Hello</div>`,
			expected: []string{"flex", "items-center"},
		},
		{
			name:     "multiple elements",
			html:     `<div class="p-4"><span class="text-red-500">Red</span></div>`,
			expected: []string{"p-4", "text-red-500"},
		},
		{
			name:     "responsive prefixes",
			html:     `<div class="sm:flex md:hidden lg:block">Responsive</div>`,
			expected: []string{"sm:flex", "md:hidden", "lg:block"},
		},
		{
			name:     "hover states",
			html:     `<button class="hover:bg-blue-500 focus:ring-2">Click</button>`,
			expected: []string{"hover:bg-blue-500", "focus:ring-2"},
		},
		{
			name:     "arbitrary values",
			html:     `<div class="w-[200px] h-[50vh] bg-[#ff0000]">Custom</div>`,
			expected: []string{"w-[200px]", "h-[50vh]", "bg-[#ff0000]"},
		},
		{
			name:     "negative values",
			html:     `<div class="-mt-4 -translate-x-1/2">Negative</div>`,
			expected: []string{"-mt-4", "-translate-x-1/2"},
		},
		{
			name:     "group modifiers",
			html:     `<div class="group"><span class="group-hover:visible">Hover me</span></div>`,
			expected: []string{"group", "group-hover:visible"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.html")
			if err := os.WriteFile(tmpFile, []byte(tt.html), 0644); err != nil {
				t.Fatal(err)
			}

			classes, err := ExtractFromDir(tmpDir)
			if err != nil {
				t.Fatal(err)
			}

			for _, exp := range tt.expected {
				if _, ok := classes[exp]; !ok {
					t.Errorf("expected class %q not found", exp)
				}
			}
		})
	}
}

func TestExtractFromDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create index.html
	index := `<!DOCTYPE html><html><body><div class="container mx-auto">Index</div></body></html>`
	if err := os.WriteFile(filepath.Join(tmpDir, "index.html"), []byte(index), 0644); err != nil {
		t.Fatal(err)
	}

	// Create about.html
	about := `<!DOCTYPE html><html><body><section class="py-8 bg-gray-100">About</section></body></html>`
	if err := os.WriteFile(filepath.Join(tmpDir, "about.html"), []byte(about), 0644); err != nil {
		t.Fatal(err)
	}

	// Create nested directory
	subDir := filepath.Join(tmpDir, "pages")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	nested := `<article class="prose max-w-none">Nested</article>`
	if err := os.WriteFile(filepath.Join(subDir, "article.html"), []byte(nested), 0644); err != nil {
		t.Fatal(err)
	}

	classes, err := ExtractFromDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"container", "mx-auto", "py-8", "bg-gray-100", "prose", "max-w-none"}
	for _, exp := range expected {
		if _, ok := classes[exp]; !ok {
			t.Errorf("expected class %q not found", exp)
		}
	}
}

func TestExtractNoClasses(t *testing.T) {
	html := `<!DOCTYPE html><html><body><div>No classes here</div></body></html>`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.html")
	if err := os.WriteFile(tmpFile, []byte(html), 0644); err != nil {
		t.Fatal(err)
	}

	classes, err := ExtractFromDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(classes) != 0 {
		t.Errorf("expected 0 classes, got %d", len(classes))
	}
}

func TestExtractEmptyClass(t *testing.T) {
	html := `<div class="">Empty</div><div class="  ">Whitespace</div>`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.html")
	if err := os.WriteFile(tmpFile, []byte(html), 0644); err != nil {
		t.Fatal(err)
	}

	classes, err := ExtractFromDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(classes) != 0 {
		t.Errorf("expected 0 classes for empty class attributes, got %d", len(classes))
	}
}
