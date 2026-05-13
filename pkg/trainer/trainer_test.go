package trainer

import (
	"regexp"
	"testing"
)

func TestTrainKeepsClassesUncoveredByPatterns(t *testing.T) {
	classes := map[string]struct{}{
		"bg-blue-500/10": {},
		"bg-blue-500/20": {},
		"bg-gray-900/50": {},
		"h-1.5":          {},
		"left-1/2":       {},
		"top-1/3":        {},
	}

	tr := New()
	tr.AddClasses(classes)
	config := tr.Train()

	for class := range classes {
		if !configCoversClass(t, config, class) {
			t.Fatalf("trained config does not cover class %q", class)
		}
	}
}

func configCoversClass(t *testing.T, config *Config, class string) bool {
	t.Helper()

	for _, literal := range config.LiteralClasses {
		if literal == class {
			return true
		}
	}

	for _, pattern := range config.Patterns {
		re, err := regexp.Compile(pattern.Regex)
		if err != nil {
			t.Fatalf("invalid regex %q: %v", pattern.Regex, err)
		}
		if re.MatchString(class) {
			return true
		}
	}

	return false
}
