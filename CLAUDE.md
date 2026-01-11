# go-reorder

A Go declaration reordering tool that organizes code by configurable conventions.

## What This Tool Does

Reorders declarations within Go source files according to a configurable section order:
1. Imports
2. main() function
3. init() functions (order preserved)
4. Exported consts, enums, vars, types, funcs
5. Unexported consts, enums, vars, types, funcs

Types are grouped with their constructors (functions matching `New*TypeName`) and methods.

## Quick Reference

```bash
# Check if files need reordering (CI-friendly)
go-reorder -c ./...

# Fix files in place
go-reorder -w ./...

# Preview changes without modifying
go-reorder -d main.go

# Create config file
go-reorder --init

# List available sections for config
go-reorder --list-sections
```

## CLI Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--write` | `-w` | Write result to source file |
| `--check` | `-c` | Exit 1 if files need reordering |
| `--diff` | `-d` | Show diff of changes |
| `--verbose` | `-v` | Show config and processing details |
| `--config` | | Explicit config file path |
| `--mode` | | Behavior: `strict`/`warn`/`append`/`drop` |
| `--exclude` | | Glob pattern to exclude (repeatable) |
| `--init` | | Create default `.go-reorder.toml` |
| `--list-sections` | | List available section names |

## Configuration

Config file: `.go-reorder.toml` (auto-discovered walking up from file location)

```toml
[sections]
order = [
  "imports",
  "main",
  "init",
  "exported_consts",
  "exported_enums",
  "exported_vars",
  "exported_types",
  "exported_funcs",
  "unexported_consts",
  "unexported_enums",
  "unexported_vars",
  "unexported_types",
  "unexported_funcs",
  "uncategorized",
]

[types]
type_layout = ["typedef", "constructors", "exported_methods", "unexported_methods"]
enum_layout = ["typedef", "iota", "exported_methods", "unexported_methods"]

[behavior]
mode = "strict"  # strict | warn | append | drop
```

### Behavior Modes

- `strict`: Error if code has no matching section (default)
- `warn`: Append unmatched code with stderr warning
- `append`: Silently append unmatched code
- `drop`: Discard unmatched code (use for file splitting)

## Library API

```go
import "github.com/toejough/go-reorder"

// Reorder with defaults
result, err := reorder.Source(srcCode)

// Reorder with custom config
cfg, _ := reorder.LoadConfig(".go-reorder.toml")
result, err := reorder.SourceWithConfig(srcCode, cfg)

// Or modify default config
cfg := reorder.DefaultConfig()
cfg.Behavior.Mode = "append"
result, err := reorder.SourceWithConfig(srcCode, cfg)

// Analyze current section order (returns section names in order found)
analysis := reorder.AnalyzeSectionOrder(srcCode)
```

## Integration Examples

### Pre-commit Hook

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/toejough/go-reorder
    rev: main
    hooks:
      - id: go-reorder        # Auto-fix
      # OR
      - id: go-reorder-check  # Check only
```

### GitHub Actions

```yaml
- run: go install github.com/toejough/go-reorder/cmd/go-reorder@latest
- run: go-reorder -c ./...
```

### Makefile

```makefile
reorder:
	go-reorder -w ./...

reorder-check:
	go-reorder -c ./...
```

## Common Tasks

### Add go-reorder to a project

```bash
go-reorder --init              # Create config
go-reorder -d ./...            # Preview changes
go-reorder -w ./...            # Apply changes
```

### Check ordering in CI

```bash
go-reorder -c ./...            # Exit 1 if changes needed
```

### Customize ordering for web app (handlers last)

```toml
[sections]
order = [
  "imports",
  "exported_types",      # Models first
  "unexported_types",
  "unexported_funcs",
  "exported_funcs",      # Handlers last
  "uncategorized",
]
```

## What This Tool Does NOT Do

- Import ordering (use `goimports` or `gci`)
- Code formatting (use `gofmt`)
- Cross-file analysis (each file independent)
- Special handling for build constraints or `//export` comments

## Key Files

- `reorder.go` - Public API functions
- `config.go` - Configuration types and loading
- `internal/categorize/` - Declaration categorization
- `internal/reassemble/` - Output generation
- `cmd/go-reorder/` - CLI implementation
