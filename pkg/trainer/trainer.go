// Package trainer generates regex patterns from CSS class names.
package trainer

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

// Pattern represents a learned regex pattern with metadata.
type Pattern struct {
	Name        string   `json:"name"`
	Regex       string   `json:"regex"`
	Description string   `json:"description"`
	Examples    []string `json:"examples"`
	Count       int      `json:"count"`
}

// Config represents the trained configuration.
type Config struct {
	Version       string    `json:"version"`
	Patterns      []Pattern `json:"patterns"`
	LiteralClasses []string `json:"literal_classes"` // Classes that don't fit patterns
	Ignored       []string  `json:"ignored"`         // Classes to always ignore
}

// Trainer learns regex patterns from CSS class names.
type Trainer struct {
	classes map[string]struct{}
	config  *Config
}

// New creates a new trainer.
func New() *Trainer {
	return &Trainer{
		classes: make(map[string]struct{}),
		config: &Config{
			Version: "1.0.0",
		},
	}
}

// AddClasses adds CSS classes to the training set.
func (t *Trainer) AddClasses(classes map[string]struct{}) {
	for class := range classes {
		t.classes[class] = struct{}{}
	}
}

// Train generates regex patterns from the collected classes.
func (t *Trainer) Train() *Config {
	// Group classes by prefix patterns
	prefixGroups := t.groupByPrefix()

	// Generate patterns for each group
	for prefix, classes := range prefixGroups {
		if len(classes) >= 3 { // Only create patterns for groups with 3+ classes
			pattern := t.generatePattern(prefix, classes)
			if pattern != nil {
				t.config.Patterns = append(t.config.Patterns, *pattern)
			}
		} else {
			// Add as literal classes
			t.config.LiteralClasses = append(t.config.LiteralClasses, classes...)
		}
	}

	// Add well-known Tailwind patterns
	t.addTailwindPatterns()

	// Sort for deterministic output
	sort.Slice(t.config.Patterns, func(i, j int) bool {
		return t.config.Patterns[i].Name < t.config.Patterns[j].Name
	})
	sort.Strings(t.config.LiteralClasses)

	return t.config
}

// groupByPrefix groups classes by their prefix (before first number or dash-number).
func (t *Trainer) groupByPrefix() map[string][]string {
	groups := make(map[string][]string)
	prefixRegex := regexp.MustCompile(`^([a-zA-Z-]+?)(?:-?\d|$)`)

	for class := range t.classes {
		match := prefixRegex.FindStringSubmatch(class)
		var prefix string
		if len(match) > 1 {
			prefix = match[1]
		} else {
			prefix = class
		}
		groups[prefix] = append(groups[prefix], class)
	}

	return groups
}

// generatePattern creates a regex pattern for a group of similar classes.
func (t *Trainer) generatePattern(prefix string, classes []string) *Pattern {
	// Analyze suffixes
	suffixes := make(map[string]struct{})
	for _, class := range classes {
		suffix := strings.TrimPrefix(class, prefix)
		suffix = strings.TrimPrefix(suffix, "-")
		if suffix != "" {
			suffixes[suffix] = struct{}{}
		}
	}

	if len(suffixes) == 0 {
		return nil
	}

	// Build regex based on suffix patterns
	var regexParts []string
	hasNumbers := false
	hasWords := false

	for suffix := range suffixes {
		if regexp.MustCompile(`^\d+$`).MatchString(suffix) {
			hasNumbers = true
		} else if regexp.MustCompile(`^[a-zA-Z]+$`).MatchString(suffix) {
			hasWords = true
			regexParts = append(regexParts, suffix)
		} else {
			regexParts = append(regexParts, regexp.QuoteMeta(suffix))
		}
	}

	var regex string
	cleanPrefix := regexp.QuoteMeta(prefix)

	if hasNumbers && hasWords {
		sort.Strings(regexParts)
		regex = fmt.Sprintf(`^%s-?(\d+|%s)$`, cleanPrefix, strings.Join(regexParts, "|"))
	} else if hasNumbers {
		regex = fmt.Sprintf(`^%s-?\d+$`, cleanPrefix)
	} else if len(regexParts) > 0 {
		sort.Strings(regexParts)
		regex = fmt.Sprintf(`^%s-?(%s)$`, cleanPrefix, strings.Join(regexParts, "|"))
	} else {
		return nil
	}

	// Get examples (up to 5)
	examples := classes
	if len(examples) > 5 {
		examples = examples[:5]
	}

	return &Pattern{
		Name:        prefix,
		Regex:       regex,
		Description: fmt.Sprintf("Matches %s-* utility classes", prefix),
		Examples:    examples,
		Count:       len(classes),
	}
}

// addTailwindPatterns adds well-known Tailwind utility patterns.
func (t *Trainer) addTailwindPatterns() {
	tailwindPatterns := []Pattern{
		{Name: "spacing", Regex: `^(m|p)(t|r|b|l|x|y)?-(\d+|auto|px)$`, Description: "Margin and padding utilities"},
		{Name: "sizing", Regex: `^(w|h|min-w|min-h|max-w|max-h)-(\d+|auto|full|screen|min|max|fit)$`, Description: "Width and height utilities"},
		{Name: "flex", Regex: `^(flex|grow|shrink|basis)-?(.*)$`, Description: "Flexbox utilities"},
		{Name: "grid", Regex: `^(grid|col|row|gap)-?(.*)$`, Description: "Grid utilities"},
		{Name: "text", Regex: `^text-(xs|sm|base|lg|xl|\d*xl|left|center|right|justify|[a-z]+-\d+)$`, Description: "Text utilities"},
		{Name: "font", Regex: `^font-(sans|serif|mono|thin|light|normal|medium|semibold|bold|extrabold|black)$`, Description: "Font utilities"},
		{Name: "bg", Regex: `^bg-(transparent|current|black|white|[a-z]+-\d+|gradient-.+)$`, Description: "Background utilities"},
		{Name: "border", Regex: `^border(-[trbl])?(-\d+)?(-[a-z]+-\d+)?$`, Description: "Border utilities"},
		{Name: "rounded", Regex: `^rounded(-[tlrb]{1,2})?(-none|-sm|-md|-lg|-xl|-2xl|-3xl|-full)?$`, Description: "Border radius utilities"},
		{Name: "shadow", Regex: `^shadow(-none|-sm|-md|-lg|-xl|-2xl|-inner)?$`, Description: "Shadow utilities"},
		{Name: "opacity", Regex: `^opacity-\d+$`, Description: "Opacity utilities"},
		{Name: "z-index", Regex: `^z-(\d+|auto)$`, Description: "Z-index utilities"},
		{Name: "transition", Regex: `^transition(-all|-colors|-opacity|-shadow|-transform|-none)?$`, Description: "Transition utilities"},
		{Name: "duration", Regex: `^duration-\d+$`, Description: "Duration utilities"},
		{Name: "ease", Regex: `^ease-(linear|in|out|in-out)$`, Description: "Easing utilities"},
		{Name: "translate", Regex: `^-?translate-[xy]-(\d+|full|px)$`, Description: "Transform translate utilities"},
		{Name: "rotate", Regex: `^-?rotate-\d+$`, Description: "Transform rotate utilities"},
		{Name: "scale", Regex: `^scale-[xy]?-?\d+$`, Description: "Transform scale utilities"},
		{Name: "animate", Regex: `^animate-(none|spin|ping|pulse|bounce)$`, Description: "Animation utilities"},
		{Name: "cursor", Regex: `^cursor-(auto|default|pointer|wait|text|move|not-allowed)$`, Description: "Cursor utilities"},
		{Name: "select", Regex: `^select-(none|text|all|auto)$`, Description: "User select utilities"},
		{Name: "overflow", Regex: `^overflow(-[xy])?-(auto|hidden|visible|scroll)$`, Description: "Overflow utilities"},
		{Name: "position", Regex: `^(static|fixed|absolute|relative|sticky)$`, Description: "Position utilities"},
		{Name: "inset", Regex: `^(inset|top|right|bottom|left)-(\d+|auto|px|full)$`, Description: "Position inset utilities"},
		{Name: "display", Regex: `^(block|inline-block|inline|flex|inline-flex|grid|inline-grid|hidden)$`, Description: "Display utilities"},
		{Name: "visibility", Regex: `^(visible|invisible)$`, Description: "Visibility utilities"},
	}

	// Only add patterns that aren't already covered
	existingNames := make(map[string]struct{})
	for _, p := range t.config.Patterns {
		existingNames[p.Name] = struct{}{}
	}

	for _, p := range tailwindPatterns {
		if _, exists := existingNames[p.Name]; !exists {
			t.config.Patterns = append(t.config.Patterns, p)
		}
	}
}

// SaveConfig saves the configuration to a file.
func (t *Trainer) SaveConfig(path string) error {
	data, err := json.MarshalIndent(t.config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadConfig loads a configuration from a file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
