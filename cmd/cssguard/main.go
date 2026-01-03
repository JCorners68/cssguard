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

	"github.com/voxell-ai/cssguard/pkg/extractor"
	"github.com/voxell-ai/cssguard/pkg/parser"
	"github.com/voxell-ai/cssguard/pkg/trainer"
	"github.com/voxell-ai/cssguard/pkg/validator"
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
    train      Train regex patterns from your CSS (run once after build)
    validate   Validate HTML classes against trained patterns (fast, for CI)
    direct     Direct comparison without patterns (slower but no training)
    version    Print version
    help       Print this help

EXAMPLES:
    # One-time training after CSS purge
    cssguard train --css ./public/css --output cssguard.json

    # Fast CI validation using trained patterns
    cssguard validate --html ./public --config cssguard.json

    # Direct comparison (no training needed)
    cssguard direct --html ./public --css ./public/css

NOTES:
    - If you add a new CSS pattern/utility that doesn't match the trained
      regex, it won't be checked. Re-run 'train' when adding new patterns.
    - For Tailwind/utility-first CSS, train against the PURGED output.

More info: https://github.com/voxell-ai/cssguard`)
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

func validateCmd(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	htmlDir := fs.String("html", "", "HTML directory to scan")
	configPath := fs.String("config", "cssguard.json", "Trained config file")
	jsonOutput := fs.Bool("json", false, "Output JSON")
	failOnOrphans := fs.Bool("fail", true, "Exit with code 1 if orphans found")
	verbose := fs.Bool("verbose", false, "Show all orphan classes")
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

	// Parse CSS classes
	cssClasses := make(map[string]struct{})
	for _, path := range strings.Split(*cssDir, ",") {
		path = strings.TrimSpace(path)
		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		var classes map[string]struct{}
		if info.IsDir() {
			classes, _ = parser.ParseFromDir(path)
		} else {
			classList, _ := parser.ParseFromFile(path)
			classes = make(map[string]struct{})
			for _, c := range classList {
				classes[c] = struct{}{}
			}
		}

		for c := range classes {
			cssClasses[c] = struct{}{}
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

// expandGlob expands a glob pattern to file paths.
func expandGlob(pattern string) []string {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil
	}
	return matches
}
