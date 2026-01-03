# CSSGuard

**Bidirectional CSS/HTML class validator** — Catch orphaned classes before they break production.

[![Go Report Card](https://goreportcard.com/badge/github.com/voxell-ai/cssguard)](https://goreportcard.com/report/github.com/voxell-ai/cssguard)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## The Problem

Tailwind CSS purges unused classes for smaller bundles. But if JavaScript adds classes at runtime (like Flowbite's drawer adding `translate-x-0`), they get purged too. Your HTML has the class. The browser applies it. **But no CSS exists**, so nothing happens. Silently.

Existing tools find unused CSS. CSSGuard finds the **opposite**: HTML classes with no CSS definition.

## Quick Start

```bash
# Install
go install github.com/voxell-ai/cssguard/cmd/cssguard@latest

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
        run: go install github.com/voxell-ai/cssguard/cmd/cssguard@latest

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
