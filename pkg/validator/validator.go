// Package validator provides CSS class validation against trained patterns.
package validator

import (
	"fmt"
	"regexp"
	"sort"

	"github.com/voxell-ai/cssguard/pkg/trainer"
)

// Result represents the validation result.
type Result struct {
	HTMLClasses    int      `json:"html_classes"`
	CSSClasses     int      `json:"css_classes"`
	Orphans        []string `json:"orphans"`         // HTML classes with no CSS
	Unused         []string `json:"unused"`          // CSS classes not in HTML
	Matched        int      `json:"matched"`         // Classes in both
	OrphanCount    int      `json:"orphan_count"`
	UnusedCount    int      `json:"unused_count"`
	CoveragePercent float64 `json:"coverage_percent"` // Matched / HTML classes
}

// Validator validates HTML classes against CSS or trained patterns.
type Validator struct {
	config          *trainer.Config
	compiledPatterns []*regexp.Regexp
	literalSet       map[string]struct{}
	ignoredSet       map[string]struct{}
}

// New creates a validator from a trained config.
func New(config *trainer.Config) (*Validator, error) {
	v := &Validator{
		config:     config,
		literalSet: make(map[string]struct{}),
		ignoredSet: make(map[string]struct{}),
	}

	// Compile regex patterns
	for _, p := range config.Patterns {
		re, err := regexp.Compile(p.Regex)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", p.Name, err)
		}
		v.compiledPatterns = append(v.compiledPatterns, re)
	}

	// Build literal class set
	for _, class := range config.LiteralClasses {
		v.literalSet[class] = struct{}{}
	}

	// Build ignored class set
	for _, class := range config.Ignored {
		v.ignoredSet[class] = struct{}{}
	}

	return v, nil
}

// ValidateAgainstPatterns checks HTML classes against trained patterns.
func (v *Validator) ValidateAgainstPatterns(htmlClasses map[string]struct{}) *Result {
	result := &Result{
		HTMLClasses: len(htmlClasses),
	}

	for class := range htmlClasses {
		// Skip ignored classes
		if _, ignored := v.ignoredSet[class]; ignored {
			result.Matched++
			continue
		}

		// Check literal classes first
		if _, found := v.literalSet[class]; found {
			result.Matched++
			continue
		}

		// Check against patterns
		matched := false
		for _, re := range v.compiledPatterns {
			if re.MatchString(class) {
				matched = true
				break
			}
		}

		if matched {
			result.Matched++
		} else {
			result.Orphans = append(result.Orphans, class)
		}
	}

	result.OrphanCount = len(result.Orphans)
	if result.HTMLClasses > 0 {
		result.CoveragePercent = float64(result.Matched) / float64(result.HTMLClasses) * 100
	}

	sort.Strings(result.Orphans)
	return result
}

// ValidateDirectly compares HTML classes directly against CSS classes (no patterns).
func ValidateDirectly(htmlClasses, cssClasses map[string]struct{}) *Result {
	result := &Result{
		HTMLClasses: len(htmlClasses),
		CSSClasses:  len(cssClasses),
	}

	// Find orphans (HTML classes not in CSS)
	for class := range htmlClasses {
		if _, found := cssClasses[class]; found {
			result.Matched++
		} else {
			result.Orphans = append(result.Orphans, class)
		}
	}

	// Find unused (CSS classes not in HTML)
	for class := range cssClasses {
		if _, found := htmlClasses[class]; !found {
			result.Unused = append(result.Unused, class)
		}
	}

	result.OrphanCount = len(result.Orphans)
	result.UnusedCount = len(result.Unused)
	if result.HTMLClasses > 0 {
		result.CoveragePercent = float64(result.Matched) / float64(result.HTMLClasses) * 100
	}

	sort.Strings(result.Orphans)
	sort.Strings(result.Unused)
	return result
}

// Summary returns a human-readable summary of the result.
func (r *Result) Summary() string {
	var s string
	s += fmt.Sprintf("HTML Classes: %d\n", r.HTMLClasses)
	if r.CSSClasses > 0 {
		s += fmt.Sprintf("CSS Classes:  %d\n", r.CSSClasses)
	}
	s += fmt.Sprintf("Matched:      %d (%.1f%%)\n", r.Matched, r.CoveragePercent)
	s += fmt.Sprintf("Orphans:      %d (HTML classes with no CSS)\n", r.OrphanCount)
	if r.UnusedCount > 0 {
		s += fmt.Sprintf("Unused:       %d (CSS classes not in HTML)\n", r.UnusedCount)
	}
	return s
}

// HasOrphans returns true if there are orphan classes.
func (r *Result) HasOrphans() bool {
	return r.OrphanCount > 0
}

// HasUnused returns true if there are unused CSS classes.
func (r *Result) HasUnused() bool {
	return r.UnusedCount > 0
}
