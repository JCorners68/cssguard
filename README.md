# CSSGuard

**Bidirectional CSS/HTML class validator** — Catch orphaned classes before they break production.

[![Go Report Card](https://goreportcard.com/badge/github.com/JCorners68/cssguard)](https://goreportcard.com/report/github.com/JCorners68/cssguard)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## The Problem

Tailwind CSS purges unused classes for smaller bundles. But if JavaScript adds classes at runtime (like Flowbite's drawer adding `translate-x-0`), they get purged too. Your HTML has the class. The browser applies it. **But no CSS exists**, so nothing happens. Silently.

Existing tools find unused CSS. CSSGuard finds the **opposite**: HTML classes with no CSS definition.

## Quick Start

```bash
# Install
go install github.com/JCorners68/cssguard/cmd/cssguard@latest

# Train (once, after CSS build)
cssguard train --css ./public/css/main.css --output cssguard.json

# Validate (in CI)
cssguard validate --html ./public --config cssguard.json --fail
```

## How It Works

```
HTML classes:  { a, b, c, d }
CSS classes:   { b, c, d, e, f }

ORPHANS (breaks site):  { a }      ← CSSGuard catches this
UNUSED (bloat):         { e, f }   ← Also reported
```

### Train Once, Validate Fast

1. **Train**: Parse your purged CSS, learn patterns (`text-{color}-{shade}`), save to JSON
2. **Validate**: Match HTML classes against patterns in milliseconds

## Commands

### `train` — Learn patterns from CSS

```bash
cssguard train --css ./public/css --output cssguard.json
```

Options:
- `--css` — CSS file or directory (required)
- `--output` — Config output path (default: `cssguard.json`)
- `--verbose` — Show pattern statistics

### `validate` — Check HTML against patterns

```bash
cssguard validate --html ./public --config cssguard.json --fail
```

Options:
- `--html` — HTML directory (required)
- `--config` — Trained config file (default: `cssguard.json`)
- `--fail` — Exit code 1 if orphans found (default: true)
- `--json` — JSON output
- `--verbose` — List orphan classes

### `direct` — Compare without training

```bash
cssguard direct --html ./public --css ./public/css --verbose
```

Slower but needs no training. Good for one-off checks.

**Redundancy Detection**: When multiple CSS files are provided, `direct` automatically detects redundant CSS (files with >80% class overlap):

```bash
cssguard direct --html ./public --css "./main.css,./vendor.css"
```

Output includes warnings if one CSS file is mostly covered by another:

```
⚠ Redundant CSS (>80% covered):
  - flowbite.min.css (85.2% covered by main.css)
```

Use `--redundancy-threshold` to adjust sensitivity (default: 80%).

### `redundancy` — Dedicated redundancy analysis

```bash
cssguard redundancy --css "./main.css,./vendor.css" --verbose
```

Compare CSS files to find duplicate class definitions. Useful for identifying vendor libraries that overlap with your Tailwind output.

```
Files analyzed: 2
Total unique classes: 1831
Redundant classes: 303 (defined in 2+ files)

File comparisons:
  main.css vs flowbite.min.css
    Overlap: 303 classes (52.3% coverage)
```

## Optional Source Scan (`--src`)

When classes are only defined in JavaScript/TypeScript (not in emitted HTML), they appear as false "orphans". The `--src` flag scans source files to extract class tokens:

```bash
# Scan source files to reduce false positives
cssguard validate --html ./public --config cssguard.json --src ./src

# Multiple source directories
cssguard direct --html ./public --css ./public/css --src ./src --src ./components
```

**How it works:**

This is **string-literal harvesting**, not framework parsing. CSSGuard extracts class tokens from:

- `class="..."` and `className="..."` attributes
- Helper functions: `clsx("...")`, `classnames("...")`, `twMerge("...")`, `cva("...")`

Only string literal arguments are extracted — template literals with `${}` and conditional expressions are skipped to avoid false positives.

**Options:**

- `--src` — Source directory/file to scan (repeatable)
- `--src-ext` — File extensions (default: `.js,.ts,.jsx,.tsx,.astro,.vue,.svelte,.md,.mdx`)
- `--src-exclude` — Directories to exclude (default: `node_modules,dist,.next,build,.git`)

**Output:**

```
Source Classes: 42
HTML Classes:   463
CSS Classes:    579
Matched:        505 (100.0%)
Orphans:        0
```

## CI Integration

### GitHub Actions

```yaml
jobs:
  build:
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Build site
        run: npm run build

      - name: Install cssguard
        run: go install github.com/JCorners68/cssguard/cmd/cssguard@latest

      - name: Validate CSS classes
        run: cssguard validate --html ./public --config cssguard.json --fail
```

### Pre-commit

```bash
#!/bin/sh
npm run build && cssguard validate --html ./public --config cssguard.json
```

## Output

```
$ cssguard validate --html ./public --verbose

HTML Classes: 847
Matched:      842 (99.4%)
Orphans:      5 (HTML classes with no CSS)

Orphan classes:
  - animate-fade-in-up-custom
  - bg-gradient-radial
  - text-balance
  - translate-x-0
  - translate-x-full
```

## Important Notes

> **If you add a new CSS pattern that doesn't match the trained regex, it won't be checked.**
>
> Re-run `cssguard train` when adding new utility patterns. This is intentional — explicit acknowledgment of new patterns prevents silent gaps.

## Performance

| Command | Time | Memory |
|---------|------|--------|
| train | ~45ms | ~8MB |
| validate | ~12ms | ~4MB |
| direct | ~38ms | ~6MB |

Tested on a 27-page Hugo site with Tailwind CSS.

## Why Not Just Use...

| Tool | Finds Orphans? | Notes |
|------|---------------|-------|
| PurgeCSS | No | It's the cause of orphans |
| Chrome Coverage | No | Only finds unused CSS |
| html-inspector | Yes | Requires PhantomJS (deprecated) |
| stylelint | No | CSS-only |
| **CSSGuard** | **Yes** | Bidirectional, modern, fast |

## Whitepaper

See [docs/WHITEPAPER.md](docs/WHITEPAPER.md) for the full technical approach.

## Contributing

PRs welcome. Please include tests.

```bash
go test ./...
```

## License

MIT — [Voxell AI](https://voxell.ai)

## Author

Jonathan Corners ([@JCorners68](https://github.com/JCorners68)) — Founder, Voxell AI
