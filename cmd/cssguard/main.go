// cssguard - Bidirectional CSS/HTML class validator
//
// A tool for detecting orphan CSS classes (used in HTML but not defined in CSS)
// and unused CSS classes (defined in CSS but not used in HTML).
//
// Copyright (c) 2025 Voxell AI. MIT License.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JCorners68/cssguard/pkg/extractor"
	"github.com/JCorners68/cssguard/pkg/parser"
	"github.com/JCorners68/cssguard/pkg/srcscan"
	"github.com/JCorners68/cssguard/pkg/trainer"
	"github.com/JCorners68/cssguard/pkg/validator"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "train":
		trainCmd(os.Args[2:])
	case "validate":
		validateCmd(os.Args[2:])
	case "direct":
		directCmd(os.Args[2:])
	case "redundancy":
		redundancyCmd(os.Args[2:])
	case "version":
		fmt.Printf("cssguard v%s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`cssguard - Bidirectional CSS/HTML class validator

USAGE:
    cssguard <command> [options]

COMMANDS:
    train       Train regex patterns from your CSS (run once after build)
    validate    Validate HTML classes against trained patterns (fast, for CI)
    direct      Direct comparison without patterns (slower but no training)
    redundancy  Find duplicate classes across CSS files (identify removable libraries)
    version     Print version
    help        Print this help

EXAMPLES:
    # One-time training after CSS purge
    cssguard train --css ./public/css --output cssguard.json

    # Fast CI validation using trained patterns
    cssguard validate --html ./public --config cssguard.json

    # Direct comparison (no training needed)
    cssguard direct --html ./public --css ./public/css

    # Find redundant CSS across multiple files
    cssguard redundancy --css ./main.css,./vendor/flowbite.min.css

NOTES:
    - If you add a new CSS pattern/utility that doesn't match the trained
      regex, it won't be checked. Re-run 'train' when adding new patterns.
    - For Tailwind/utility-first CSS, train against the PURGED output.

More info: https://github.com/JCorners68/cssguard`)
}

func trainCmd(args []string) {
	fs := flag.NewFlagSet("train", flag.ExitOnError)
	cssDir := fs.String("css", "", "CSS directory or file(s) to parse (comma-separated)")
	output := fs.String("output", "cssguard.json", "Output config file")
	verbose := fs.Bool("verbose", false, "Verbose output")
	fs.Parse(args)

	if *cssDir == "" {
		fmt.Fprintln(os.Stderr, "Error: --css is required")
		fs.Usage()
		os.Exit(1)
	}

	// Parse CSS files
	cssClasses := make(map[string]struct{})
	for _, path := range strings.Split(*cssDir, ",") {
		path = strings.TrimSpace(path)
		info, err := os.Stat(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: cannot stat %s: %v\n", path, err)
			continue
		}

		var classes map[string]struct{}
		if info.IsDir() {
			classes, err = parser.ParseFromDir(path)
		} else {
			classList, err2 := parser.ParseFromFile(path)
			if err2 != nil {
				err = err2
			} else {
				classes = make(map[string]struct{})
				for _, c := range classList {
					classes[c] = struct{}{}
				}
			}
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: error parsing %s: %v\n", path, err)
			continue
		}

		for c := range classes {
			cssClasses[c] = struct{}{}
		}
	}

	if len(cssClasses) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no CSS classes found")
		os.Exit(1)
	}

	if *verbose {
		fmt.Printf("Found %d unique CSS classes\n", len(cssClasses))
	}

	// Train
	t := trainer.New()
	t.AddClasses(cssClasses)
	config := t.Train()

	if *verbose {
		fmt.Printf("Generated %d patterns and %d literal classes\n",
			len(config.Patterns), len(config.LiteralClasses))
	}

	// Save
	if err := t.SaveConfig(*output); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Trained config saved to %s\n", *output)
	fmt.Printf("  Patterns: %d\n", len(config.Patterns))
	fmt.Printf("  Literals: %d\n", len(config.LiteralClasses))
}

// srcPathsFlag is a repeatable string flag for --src paths.
type srcPathsFlag []string

func (s *srcPathsFlag) String() string {
	return strings.Join(*s, ",")
}

func (s *srcPathsFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func validateCmd(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	htmlDir := fs.String("html", "", "HTML directory to scan")
	configPath := fs.String("config", "cssguard.json", "Trained config file")
	jsonOutput := fs.Bool("json", false, "Output JSON")
	failOnOrphans := fs.Bool("fail", true, "Exit with code 1 if orphans found")
	verbose := fs.Bool("verbose", false, "Show all orphan classes")

	// Source scanning flags
	var srcPaths srcPathsFlag
	fs.Var(&srcPaths, "src", "Source directory/file to scan for class tokens (repeatable)")
	srcExt := fs.String("src-ext", "", "Source file extensions (default: .js,.ts,.jsx,.tsx,.astro,.vue,.svelte,.md,.mdx)")
	srcExclude := fs.String("src-exclude", "", "Directories to exclude (default: node_modules,dist,.next,build,.git)")

	fs.Parse(args)

	if *htmlDir == "" {
		fmt.Fprintln(os.Stderr, "Error: --html is required")
		fs.Usage()
		os.Exit(1)
	}

	// Load config
	config, err := trainer.LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		fmt.Fprintln(os.Stderr, "Run 'cssguard train' first to generate config")
		os.Exit(1)
	}

	// Extract HTML classes
	htmlClasses, err := extractor.ExtractFromDir(*htmlDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting HTML classes: %v\n", err)
		os.Exit(1)
	}

	// Extract source classes if --src provided
	var srcClassCount int
	if len(srcPaths) > 0 {
		opts := srcscan.Options{
			Extensions: srcscan.ParseExtensions(*srcExt),
			Excludes:   srcscan.ParseExcludes(*srcExclude),
		}
		scanner := srcscan.New(opts)
		srcClasses, err := scanner.ScanPaths(srcPaths)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error scanning source files: %v\n", err)
			os.Exit(1)
		}
		srcClassCount = len(srcClasses)
		// Merge source classes into HTML classes
		for c := range srcClasses {
			htmlClasses[c] = struct{}{}
		}
	}

	// Validate
	v, err := validator.New(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating validator: %v\n", err)
		os.Exit(1)
	}

	result := v.ValidateAgainstPatterns(htmlClasses)

	// Output
	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result)
	} else {
		if srcClassCount > 0 {
			fmt.Printf("Source Classes: %d\n", srcClassCount)
		}
		fmt.Print(result.Summary())
		if *verbose && result.HasOrphans() {
			fmt.Println("\nOrphan classes:")
			for _, class := range result.Orphans {
				fmt.Printf("  - %s\n", class)
			}
		}
	}

	if *failOnOrphans && result.HasOrphans() {
		os.Exit(1)
	}
}

func directCmd(args []string) {
	fs := flag.NewFlagSet("direct", flag.ExitOnError)
	htmlDir := fs.String("html", "", "HTML directory to scan")
	cssDir := fs.String("css", "", "CSS directory or file(s) to parse")
	jsonOutput := fs.Bool("json", false, "Output JSON")
	failOnOrphans := fs.Bool("fail", true, "Exit with code 1 if orphans found")
	verbose := fs.Bool("verbose", false, "Show orphan and unused classes")
	showUnused := fs.Bool("unused", false, "Also report unused CSS classes")

	// Source scanning flags
	var srcPaths srcPathsFlag
	fs.Var(&srcPaths, "src", "Source directory/file to scan for class tokens (repeatable)")
	srcExt := fs.String("src-ext", "", "Source file extensions (default: .js,.ts,.jsx,.tsx,.astro,.vue,.svelte,.md,.mdx)")
	srcExclude := fs.String("src-exclude", "", "Directories to exclude (default: node_modules,dist,.next,build,.git)")

	fs.Parse(args)

	if *htmlDir == "" || *cssDir == "" {
		fmt.Fprintln(os.Stderr, "Error: --html and --css are required")
		fs.Usage()
		os.Exit(1)
	}

	// Extract HTML classes
	htmlClasses, err := extractor.ExtractFromDir(*htmlDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error extracting HTML classes: %v\n", err)
		os.Exit(1)
	}

	// Extract source classes if --src provided
	var srcClassCount int
	if len(srcPaths) > 0 {
		opts := srcscan.Options{
			Extensions: srcscan.ParseExtensions(*srcExt),
			Excludes:   srcscan.ParseExcludes(*srcExclude),
		}
		scanner := srcscan.New(opts)
		srcClasses, err := scanner.ScanPaths(srcPaths)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error scanning source files: %v\n", err)
			os.Exit(1)
		}
		srcClassCount = len(srcClasses)
		// Merge source classes into HTML classes
		for c := range srcClasses {
			htmlClasses[c] = struct{}{}
		}
	}

	// Parse CSS classes
	cssClasses := make(map[string]struct{})
	var parseErrors []string
	for _, path := range strings.Split(*cssDir, ",") {
		path = strings.TrimSpace(path)
		info, err := os.Stat(path)
		if err != nil {
			parseErrors = append(parseErrors, fmt.Sprintf("%s: %v", path, err))
			continue
		}

		var classes map[string]struct{}
		var parseErr error
		if info.IsDir() {
			classes, parseErr = parser.ParseFromDir(path)
		} else {
			classList, err := parser.ParseFromFile(path)
			if err != nil {
				parseErr = err
			} else {
				classes = make(map[string]struct{})
				for _, c := range classList {
					classes[c] = struct{}{}
				}
			}
		}

		if parseErr != nil {
			parseErrors = append(parseErrors, fmt.Sprintf("%s: %v", path, parseErr))
			continue
		}

		for c := range classes {
			cssClasses[c] = struct{}{}
		}
	}

	if len(parseErrors) > 0 {
		fmt.Fprintf(os.Stderr, "Warning: %d CSS path(s) had errors:\n", len(parseErrors))
		for _, e := range parseErrors {
			fmt.Fprintf(os.Stderr, "  - %s\n", e)
		}
	}

	// Validate directly
	result := validator.ValidateDirectly(htmlClasses, cssClasses)

	// Output
	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result)
	} else {
		if srcClassCount > 0 {
			fmt.Printf("Source Classes: %d\n", srcClassCount)
		}
		fmt.Print(result.Summary())
		if *verbose {
			if result.HasOrphans() {
				fmt.Println("\nOrphan classes (in HTML, not in CSS):")
				for _, class := range result.Orphans {
					fmt.Printf("  - %s\n", class)
				}
			}
			if *showUnused && result.HasUnused() {
				fmt.Println("\nUnused classes (in CSS, not in HTML):")
				max := 20
				for i, class := range result.Unused {
					if i >= max {
						fmt.Printf("  ... and %d more\n", len(result.Unused)-max)
						break
					}
					fmt.Printf("  - %s\n", class)
				}
			}
		}
	}

	if *failOnOrphans && result.HasOrphans() {
		os.Exit(1)
	}
}

func redundancyCmd(args []string) {
	fs := flag.NewFlagSet("redundancy", flag.ExitOnError)
	cssFiles := fs.String("css", "", "CSS files to compare (comma-separated)")
	jsonOutput := fs.Bool("json", false, "Output JSON")
	verbose := fs.Bool("verbose", false, "Show all redundant classes")
	threshold := fs.Float64("threshold", 80.0, "Coverage threshold to suggest removal (%)")
	fs.Parse(args)

	if *cssFiles == "" {
		fmt.Fprintln(os.Stderr, "Error: --css is required (comma-separated list of CSS files)")
		fs.Usage()
		os.Exit(1)
	}

	paths := strings.Split(*cssFiles, ",")
	if len(paths) < 2 {
		fmt.Fprintln(os.Stderr, "Error: need at least 2 CSS files to compare")
		os.Exit(1)
	}

	// Parse each CSS file separately, tracking which classes come from which file
	fileClasses := make(map[string]map[string]struct{}) // file -> set of classes
	allClasses := make(map[string][]string)             // class -> list of files

	for _, path := range paths {
		path = strings.TrimSpace(path)
		info, err := os.Stat(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: cannot stat %s: %v\n", path, err)
			continue
		}

		var classes map[string]struct{}
		if info.IsDir() {
			classes, err = parser.ParseFromDir(path)
		} else {
			classList, err2 := parser.ParseFromFile(path)
			if err2 != nil {
				err = err2
			} else {
				classes = make(map[string]struct{})
				for _, c := range classList {
					classes[c] = struct{}{}
				}
			}
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: error parsing %s: %v\n", path, err)
			continue
		}

		fileClasses[path] = classes
		for c := range classes {
			allClasses[c] = append(allClasses[c], path)
		}
	}

	// Find redundant classes (defined in multiple files)
	redundant := make(map[string][]string)
	for class, files := range allClasses {
		if len(files) > 1 {
			redundant[class] = files
		}
	}

	// Calculate coverage for each file pair
	type FilePair struct {
		File1    string  `json:"file1"`
		File2    string  `json:"file2"`
		Overlap  int     `json:"overlap"`
		File1Only int    `json:"file1_only"`
		File2Only int    `json:"file2_only"`
		Coverage float64 `json:"coverage_percent"` // % of smaller file covered by larger
	}

	var pairs []FilePair
	pathList := make([]string, 0, len(fileClasses))
	for p := range fileClasses {
		pathList = append(pathList, p)
	}

	for i := 0; i < len(pathList); i++ {
		for j := i + 1; j < len(pathList); j++ {
			f1, f2 := pathList[i], pathList[j]
			c1, c2 := fileClasses[f1], fileClasses[f2]

			overlap := 0
			for c := range c1 {
				if _, ok := c2[c]; ok {
					overlap++
				}
			}

			smaller := len(c1)
			if len(c2) < smaller {
				smaller = len(c2)
			}

			coverage := 0.0
			if smaller > 0 {
				coverage = float64(overlap) / float64(smaller) * 100
			}

			pairs = append(pairs, FilePair{
				File1:    f1,
				File2:    f2,
				Overlap:  overlap,
				File1Only: len(c1) - overlap,
				File2Only: len(c2) - overlap,
				Coverage: coverage,
			})
		}
	}

	// Output
	type RedundancyResult struct {
		TotalFiles      int                 `json:"total_files"`
		TotalClasses    int                 `json:"total_classes"`
		RedundantCount  int                 `json:"redundant_count"`
		Pairs           []FilePair          `json:"pairs"`
		Redundant       map[string][]string `json:"redundant,omitempty"`
		Removable       []string            `json:"removable,omitempty"`
	}

	// Find potentially removable files
	var removable []string
	for _, pair := range pairs {
		c1, c2 := fileClasses[pair.File1], fileClasses[pair.File2]

		// Check if file1 is fully covered by file2
		covered1 := 0
		for c := range c1 {
			if _, ok := c2[c]; ok {
				covered1++
			}
		}
		if len(c1) > 0 && float64(covered1)/float64(len(c1))*100 >= *threshold {
			removable = append(removable, fmt.Sprintf("%s (%.1f%% covered by %s)", pair.File1, float64(covered1)/float64(len(c1))*100, pair.File2))
		}

		// Check if file2 is fully covered by file1
		covered2 := 0
		for c := range c2 {
			if _, ok := c1[c]; ok {
				covered2++
			}
		}
		if len(c2) > 0 && float64(covered2)/float64(len(c2))*100 >= *threshold {
			removable = append(removable, fmt.Sprintf("%s (%.1f%% covered by %s)", pair.File2, float64(covered2)/float64(len(c2))*100, pair.File1))
		}
	}

	result := RedundancyResult{
		TotalFiles:     len(fileClasses),
		TotalClasses:   len(allClasses),
		RedundantCount: len(redundant),
		Pairs:          pairs,
		Removable:      removable,
	}

	if *verbose {
		result.Redundant = redundant
	}

	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(result)
	} else {
		fmt.Printf("Files analyzed: %d\n", result.TotalFiles)
		fmt.Printf("Total unique classes: %d\n", result.TotalClasses)
		fmt.Printf("Redundant classes: %d (defined in 2+ files)\n", result.RedundantCount)

		fmt.Println("\nFile comparisons:")
		for _, pair := range pairs {
			fmt.Printf("  %s vs %s\n", filepath.Base(pair.File1), filepath.Base(pair.File2))
			fmt.Printf("    Overlap: %d classes (%.1f%% coverage)\n", pair.Overlap, pair.Coverage)
			fmt.Printf("    Only in %s: %d\n", filepath.Base(pair.File1), pair.File1Only)
			fmt.Printf("    Only in %s: %d\n", filepath.Base(pair.File2), pair.File2Only)
		}

		if len(removable) > 0 {
			fmt.Printf("\nPotentially removable (>%.0f%% coverage):\n", *threshold)
			for _, r := range removable {
				fmt.Printf("  - %s\n", r)
			}
		}

		if *verbose && len(redundant) > 0 {
			fmt.Println("\nRedundant classes:")
			count := 0
			for class, files := range redundant {
				if count >= 20 {
					fmt.Printf("  ... and %d more\n", len(redundant)-20)
					break
				}
				fileNames := make([]string, len(files))
				for i, f := range files {
					fileNames[i] = filepath.Base(f)
				}
				fmt.Printf("  %s: %s\n", class, strings.Join(fileNames, ", "))
				count++
			}
		}
	}
}

// expandGlob expands a glob pattern to file paths.
func expandGlob(pattern string) []string {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil
	}
	return matches
}
