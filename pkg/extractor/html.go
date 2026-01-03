// Package extractor provides HTML class extraction utilities.
package extractor

import (
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// ExtractFromFile extracts all CSS class names from an HTML file.
func ExtractFromFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ExtractFromReader(f)
}

// ExtractFromReader extracts all CSS class names from an HTML reader.
func ExtractFromReader(r io.Reader) ([]string, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}

	classes := make(map[string]struct{})
	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode {
			for _, attr := range n.Attr {
				if attr.Key == "class" {
					for _, class := range strings.Fields(attr.Val) {
						class = strings.TrimSpace(class)
						if class != "" {
							classes[class] = struct{}{}
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}
	extract(doc)

	result := make([]string, 0, len(classes))
	for class := range classes {
		result = append(result, class)
	}
	return result, nil
}

// ExtractFromDir recursively extracts classes from all HTML files in a directory.
func ExtractFromDir(dir string) (map[string]struct{}, error) {
	classes := make(map[string]struct{})

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(path), ".html") {
			return nil
		}

		fileClasses, err := ExtractFromFile(path)
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

// ExtractFromGlob extracts classes from files matching a glob pattern.
func ExtractFromGlob(pattern string) (map[string]struct{}, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	classes := make(map[string]struct{})
	for _, path := range matches {
		fileClasses, err := ExtractFromFile(path)
		if err != nil {
			continue // Skip files that can't be parsed
		}
		for _, class := range fileClasses {
			classes[class] = struct{}{}
		}
	}
	return classes, nil
}

// classInStyleRegex matches class names in style attributes (for inline detection)
var classInStyleRegex = regexp.MustCompile(`\.([a-zA-Z_-][a-zA-Z0-9_-]*)`)

// ExtractFromInlineStyles extracts class references from <style> tags.
func ExtractFromInlineStyles(r io.Reader) ([]string, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}

	classes := make(map[string]struct{})
	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "style" {
			if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
				matches := classInStyleRegex.FindAllStringSubmatch(n.FirstChild.Data, -1)
				for _, match := range matches {
					if len(match) > 1 {
						classes[match[1]] = struct{}{}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}
	extract(doc)

	result := make([]string, 0, len(classes))
	for class := range classes {
		result = append(result, class)
	}
	return result, nil
}
