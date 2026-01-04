# CSSGuard: Bidirectional CSS/HTML Class Validation

**Version 1.0 | January 2026**
**Author: Jonathan Corners, Voxell AI**

## Abstract

Modern CSS toolchains like Tailwind CSS purge unused classes to minimize bundle size. This creates a new class of bug: HTML references classes that were purged from CSS, causing silent styling failures. Existing tools detect unused CSS (bloat) but not orphaned HTML classes (breakage).

CSSGuard solves this with bidirectional validation: detecting both classes in HTML that have no CSS definition (critical errors) and classes in CSS never used in HTML (optimization opportunities).

## The Problem

### The Traditional Direction: CSS → HTML

For decades, the CSS problem was bloat. Developers wrote CSS, forgot to delete rules, and shipped megabytes of unused styles. Tools emerged to solve this:

- **PurifyCSS** (2015): Scans HTML/JS, removes unused CSS
- **PurgeCSS** (2018): Powers Tailwind's production builds
- **UnCSS**: Chrome DevTools Coverage panel

These tools ask: *"Which CSS selectors have no matching HTML elements?"*

### The New Direction: HTML → CSS

Utility-first CSS frameworks changed the equation. With Tailwind:

1. You write `class="translate-x-0"` in HTML
2. Tailwind generates `.translate-x-0 { transform: translateX(0); }` on-demand
3. PurgeCSS removes classes not found in your source files
4. **But**: If JavaScript adds classes at runtime (e.g., Flowbite drawer), they get purged

The failure mode is silent. The HTML has the class. The browser applies it. But no CSS rule exists, so nothing happens.

**Real-world example**: Flowbite's drawer component adds `translate-x-0` via JavaScript. If this class isn't in your HTML source files, Tailwind purges it. The drawer stops animating. No error is thrown.

### Why Existing Tools Miss This

| Tool | Direction | Catches Orphans? |
|------|-----------|------------------|
| PurgeCSS | CSS → HTML | No (it's the cause) |
| Chrome Coverage | CSS → HTML | No |
| html-inspector | Both | Yes, but requires PhantomJS (deprecated) |
| stylelint | CSS only | No |

## The Solution: Bidirectional Validation

CSSGuard performs set operations on two collections:

```
HTML_CLASSES = { classes extracted from all HTML files }
CSS_CLASSES  = { selectors parsed from all CSS files }

ORPHANS = HTML_CLASSES - CSS_CLASSES  // Critical: will break styling
UNUSED  = CSS_CLASSES - HTML_CLASSES  // Optimization: bloat
HEALTHY = HTML_CLASSES ∩ CSS_CLASSES  // Working correctly
```

### The Regex Training Approach

Direct comparison works but doesn't scale for utility-first CSS. Tailwind generates thousands of utilities (`text-gray-50` through `text-gray-950`). Comparing literally is wasteful.

CSSGuard uses a train-once, validate-fast approach:

**Phase 1: Training (run once after CSS build)**
1. Parse purged CSS to extract all class selectors
2. Identify patterns (e.g., `text-{color}-{shade}`)
3. Generate regex matchers
4. Save patterns to `cssguard.json`

**Phase 2: Validation (run in CI)**
1. Extract classes from HTML
2. Match against compiled regex patterns
3. Report unmatched classes as potential orphans

This reduces validation from O(n×m) comparisons to O(n×p) regex matches, where p << m (patterns << literal classes).

### Pattern Learning

CSSGuard learns patterns from your actual CSS:

```json
{
  "patterns": [
    {
      "name": "text-color",
      "regex": "^text-(gray|red|blue|green)-\\d+$",
      "description": "Text color utilities",
      "examples": ["text-gray-500", "text-red-600"],
      "count": 147
    }
  ],
  "literal_classes": ["antialiased", "container", "prose"]
}
```

New utilities that don't match existing patterns require re-training. This is intentional: it forces explicit acknowledgment of new patterns.

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      cssguard CLI                        │
├─────────────────────────────────────────────────────────┤
│  train          │  validate           │  direct          │
│  ─────          │  ────────           │  ──────          │
│  CSS → Patterns │  HTML vs Patterns   │  HTML vs CSS     │
└────────┬────────┴─────────┬───────────┴────────┬────────┘
         │                  │                    │
    ┌────▼────┐        ┌────▼────┐         ┌────▼────┐
    │ parser  │        │extractor│         │validator│
    │ (CSS)   │        │ (HTML)  │         │ (sets)  │
    └─────────┘        └─────────┘         └─────────┘
         │                  │                    │
    ┌────▼────┐        ┌────▼────┐         ┌────▼────┐
    │ trainer │───────▶│ config  │◀────────│ compare │
    │ (regex) │        │ (JSON)  │         │ (diff)  │
    └─────────┘        └─────────┘         └─────────┘
```

## Usage

### Installation

```bash
go install github.com/JCorners68/cssguard/cmd/cssguard@latest
```

### Training (One-Time)

After your CSS build (post-Tailwind purge):

```bash
cssguard train --css ./public/css/main.css --output cssguard.json
```

### Validation (CI)

```bash
cssguard validate --html ./public --config cssguard.json --fail
```

Exit code 1 if orphans found.

### Direct Mode (No Training)

For quick checks without training:

```bash
cssguard direct --html ./public --css ./public/css --verbose
```

### Redundancy Detection

When multiple CSS files are provided, CSSGuard automatically detects redundant CSS—files where one largely duplicates another:

```bash
cssguard direct --html ./public --css "./main.css,./vendor/flowbite.min.css"
```

This addresses a common pattern: developers add a component library (Flowbite, DaisyUI) alongside their Tailwind build, unaware that both define the same utility classes. The result is bloated CSS with duplicate definitions.

CSSGuard calculates overlap between CSS files:

```
Files analyzed: 2
Total unique classes: 1831
Redundant classes: 303 (defined in 2+ files)

File comparisons:
  main.css vs flowbite.min.css
    Overlap: 303 classes (52.3% coverage)
    Only in main.css: 276
    Only in flowbite.min.css: 1252

⚠ Redundant CSS (>80% covered):
  - flowbite.min.css (85.2% covered by main.css)
```

When one file is >80% covered by another, CSSGuard flags it as removable. This threshold is configurable via `--redundancy-threshold`.

**CI Integration**: The `direct` command can be used in CI to fail builds when redundant CSS is detected:

```yaml
- name: Check for CSS redundancy
  run: |
    CSS_FILES=$(find ./static/css -name "*.css" | tr '\n' ',')
    cssguard direct --html ./public --css "$CSS_FILES"
```

## CI Integration

### GitHub Actions

```yaml
- name: Build site
  run: npm run build

- name: Validate CSS classes
  run: |
    cssguard validate --html ./public --config cssguard.json --fail
```

### Pre-commit Hook

```bash
#!/bin/sh
npm run build
cssguard validate --html ./public --config cssguard.json
```

## Limitations

1. **New patterns require re-training**: If you introduce a new utility pattern (e.g., custom plugin), re-run `train`. This is by design—explicit acknowledgment of new patterns.

2. **JavaScript-only classes**: Classes added purely via JS that never appear in HTML won't be in the HTML set. Use Tailwind's `safelist` for these.

3. **Dynamic class composition**: Template literals like `` `text-${color}-500` `` won't be detected. Use full class names in source.

4. **Source-only classes**: For classes present only in JS/TS source (but not emitted HTML), CSSGuard can optionally scan sources via --src; truly runtime-generated classes may still require a Tailwind safelist.

## Comparison with Alternatives

| Feature | CSSGuard | html-inspector | PurgeCSS |
|---------|----------|----------------|----------|
| Orphan detection | ✓ | ✓ | ✗ |
| Unused detection | ✓ | ✓ | ✓ |
| No browser required | ✓ | ✗ (PhantomJS) | ✓ |
| Pattern learning | ✓ | ✗ | ✗ |
| Tailwind-aware | ✓ | ✗ | ✓ |
| Active maintenance | ✓ | ✗ (2017) | ✓ |

## Performance

Tested on a Hugo site with 27 pages and Tailwind CSS:

| Mode | Time | Memory |
|------|------|--------|
| train | 45ms | 8MB |
| validate | 12ms | 4MB |
| direct | 38ms | 6MB |

## Conclusion

The shift to utility-first CSS created a gap in tooling. We optimized for removing unused CSS but created a new failure mode: HTML referencing purged classes.

CSSGuard closes this gap with bidirectional validation. Train once against your purged CSS, then validate in CI. Catch orphaned classes before they break production.

---

**License**: MIT
**Repository**: https://github.com/JCorners68/cssguard
**Author**: Jonathan Corners ([@JCorners68](https://github.com/JCorners68))
