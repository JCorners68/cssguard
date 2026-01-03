// Package srcscan extracts CSS class tokens from source code files.
// This is a string-literal harvesting approach, not framework parsing.
package srcscan

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// DefaultExtensions are the file extensions to scan by default.
var DefaultExtensions = []string{
	".js", ".ts", ".jsx", ".tsx", ".astro", ".vue", ".svelte", ".md", ".mdx",
}

// DefaultExcludes are directories to exclude by default.
var DefaultExcludes = []string{
	"node_modules", "dist", ".next", "build", ".git", ".svelte-kit", ".nuxt",
}

// classTokenRegex matches valid CSS class tokens.
// Includes # for Tailwind arbitrary values like bg-[#ff0000]
var classTokenRegex = regexp.MustCompile(`^[A-Za-z0-9:_\-\[\]/#%.]+$`)

// Patterns for extracting class strings from source code.
var (
	// class="..." or className="..." (double or single quotes)
	classAttrRegex = regexp.MustCompile(`(?:class|className)\s*=\s*["']([^"']+)["']`)

	// clsx("..."), classnames("..."), twMerge("..."), cva("...")
	// Only captures string literal arguments
	helperRegex = regexp.MustCompile(`(?:clsx|classnames|twMerge|cva|cn)\s*\(\s*["']([^"']+)["']`)
)

// Options configures source scanning behavior.
type Options struct {
	Extensions []string // File extensions to scan (e.g., ".tsx")
	Excludes   []string // Directories to exclude (e.g., "node_modules")
}

// DefaultOptions returns the default scanning options.
func DefaultOptions() Options {
	return Options{
		Extensions: DefaultExtensions,
		Excludes:   DefaultExcludes,
	}
}

// Scanner extracts class tokens from source files.
type Scanner struct {
	opts Options
}

// New creates a new Scanner with the given options.
func New(opts Options) *Scanner {
	if len(opts.Extensions) == 0 {
		opts.Extensions = DefaultExtensions
	}
	if len(opts.Excludes) == 0 {
		opts.Excludes = DefaultExcludes
	}
	return &Scanner{opts: opts}
}

// ScanPaths scans the given paths (files or directories) and returns all found class tokens.
func (s *Scanner) ScanPaths(paths []string) (map[string]struct{}, error) {
	classes := make(map[string]struct{})

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			continue // Skip paths that don't exist
		}

		if info.IsDir() {
			dirClasses, err := s.scanDir(path)
			if err != nil {
				return nil, err
			}
			for c := range dirClasses {
				classes[c] = struct{}{}
			}
		} else {
			fileClasses, err := s.scanFile(path)
			if err != nil {
				continue // Skip files that can't be read
			}
			for c := range fileClasses {
				classes[c] = struct{}{}
			}
		}
	}

	return classes, nil
}

// scanDir recursively scans a directory for source files.
func (s *Scanner) scanDir(dir string) (map[string]struct{}, error) {
	classes := make(map[string]struct{})

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Check for excluded directories
		if info.IsDir() {
			for _, exclude := range s.opts.Excludes {
				if info.Name() == exclude {
					return filepath.SkipDir
				}
			}
			return nil
		}

		// Check file extension
		ext := strings.ToLower(filepath.Ext(path))
		hasExt := false
		for _, e := range s.opts.Extensions {
			if ext == e {
				hasExt = true
				break
			}
		}
		if !hasExt {
			return nil
		}

		// Scan the file
		fileClasses, err := s.scanFile(path)
		if err != nil {
			return nil // Skip files that can't be read
		}
		for c := range fileClasses {
			classes[c] = struct{}{}
		}

		return nil
	})

	return classes, err
}

// scanFile extracts class tokens from a single source file.
func (s *Scanner) scanFile(path string) (map[string]struct{}, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	classes := make(map[string]struct{})
	scanner := bufio.NewScanner(f)

	// Increase buffer for long lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024) // 1MB max line

	for scanner.Scan() {
		line := scanner.Text()

		// Extract from class/className attributes (only quoted strings, not template literals)
		matches := classAttrRegex.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) > 1 {
				// Skip if the captured value contains interpolation markers
				if strings.Contains(match[1], "${") || strings.Contains(match[1], "` +") {
					continue
				}
				extractTokens(match[1], classes)
			}
		}

		// Extract from helper functions (only string literal arguments)
		matches = helperRegex.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) > 1 {
				// Skip if the captured value contains interpolation markers
				if strings.Contains(match[1], "${") || strings.Contains(match[1], "` +") {
					continue
				}
				extractTokens(match[1], classes)
			}
		}
	}

	return classes, scanner.Err()
}

// extractTokens splits a class string and adds valid tokens to the set.
func extractTokens(s string, classes map[string]struct{}) {
	tokens := strings.Fields(s)
	for _, token := range tokens {
		token = strings.TrimSpace(token)

		// Skip empty tokens
		if token == "" {
			continue
		}

		// Skip tokens that are too long (sanity limit)
		if len(token) > 128 {
			continue
		}

		// Validate token format
		if !classTokenRegex.MatchString(token) {
			continue
		}

		classes[token] = struct{}{}
	}
}

// ParseExtensions parses a comma-separated list of extensions.
func ParseExtensions(s string) []string {
	if s == "" {
		return DefaultExtensions
	}
	parts := strings.Split(s, ",")
	exts := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			if !strings.HasPrefix(p, ".") {
				p = "." + p
			}
			exts = append(exts, p)
		}
	}
	return exts
}

// ParseExcludes parses a comma-separated list of excluded directories.
func ParseExcludes(s string) []string {
	if s == "" {
		return DefaultExcludes
	}
	parts := strings.Split(s, ",")
	excludes := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			excludes = append(excludes, p)
		}
	}
	return excludes
}
