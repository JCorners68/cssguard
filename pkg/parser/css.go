// Package parser provides CSS class selector parsing utilities.
package parser

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// classRegex matches CSS class selectors (e.g., .foo, .bar-baz, .text-gray-500)
// Handles escaped characters like \/ \: \[ \] in Tailwind classes
var classRegex = regexp.MustCompile(`\.(-?[_a-zA-Z][_a-zA-Z0-9-]*(?:\\[/:.\[\]()%][_a-zA-Z0-9-]*)*)`)

// pseudoCleanRegex removes pseudo-classes/elements from selectors
var pseudoCleanRegex = regexp.MustCompile(`::?[a-zA-Z-]+(\([^)]*\))?`)

// ParseFromFile extracts all CSS class selectors from a CSS file.
func ParseFromFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ParseFromReader(f)
}

// ParseFromReader extracts all CSS class selectors from a CSS reader.
func ParseFromReader(r io.Reader) ([]string, error) {
	classes := make(map[string]struct{})

	scanner := bufio.NewScanner(r)
	// Increase buffer size for minified CSS
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024) // 10MB max

	for scanner.Scan() {
		line := scanner.Text()
		// Remove pseudo-classes/elements to get clean class names
		cleaned := pseudoCleanRegex.ReplaceAllString(line, "")
		matches := classRegex.FindAllStringSubmatch(cleaned, -1)
		for _, match := range matches {
			if len(match) > 1 {
				className := match[1]
				// Skip Tailwind's escaped characters (e.g., \:, \/)
				className = unescapeClassName(className)
				if className != "" && !strings.HasPrefix(className, "-") || isValidNegativeClass(className) {
					classes[className] = struct{}{}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	result := make([]string, 0, len(classes))
	for class := range classes {
		result = append(result, class)
	}
	return result, nil
}

// unescapeClassName handles Tailwind's escaped class names
func unescapeClassName(name string) string {
	// Handle common escapes: \: -> :, \/ -> /, \. -> .
	replacer := strings.NewReplacer(
		`\:`, `:`,
		`\/`, `/`,
		`\.`, `.`,
		`\[`, `[`,
		`\]`, `]`,
		`\(`, `(`,
		`\)`, `)`,
		`\,`, `,`,
		`\%`, `%`,
	)
	return replacer.Replace(name)
}

// isValidNegativeClass checks if a negative class is valid (e.g., -translate-x-full)
func isValidNegativeClass(name string) bool {
	if !strings.HasPrefix(name, "-") {
		return true
	}
	// Valid negative Tailwind utilities
	validPrefixes := []string{
		"-translate", "-rotate", "-skew", "-scale",
		"-m-", "-mx-", "-my-", "-mt-", "-mr-", "-mb-", "-ml-",
		"-p-", "-px-", "-py-", "-pt-", "-pr-", "-pb-", "-pl-",
		"-inset", "-top-", "-right-", "-bottom-", "-left-",
		"-z-", "-order-", "-tracking-", "-indent-",
	}
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// ParseFromDir extracts classes from all CSS files in a directory.
func ParseFromDir(dir string) (map[string]struct{}, error) {
	classes := make(map[string]struct{})

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(path), ".css") {
			return nil
		}

		fileClasses, err := ParseFromFile(path)
		if err != nil {
			return err
		}
		for _, class := range fileClasses {
			classes[class] = struct{}{}
		}
		return nil
	})

	return classes, err
}

// ParseFromFiles extracts classes from multiple CSS files.
func ParseFromFiles(paths []string) (map[string]struct{}, error) {
	classes := make(map[string]struct{})

	for _, path := range paths {
		fileClasses, err := ParseFromFile(path)
		if err != nil {
			continue // Skip files that can't be parsed
		}
		for _, class := range fileClasses {
			classes[class] = struct{}{}
		}
	}
	return classes, nil
}
