# go-reorder

![go-reorder](assets/reorder.png)

A Go code reordering tool that organizes declarations according to configurable conventions.

## Quick Start

```bash
# Install
go install github.com/toejough/go-reorder/cmd/go-reorder@latest

# Check which files need reordering
go-reorder -c ./...

# Fix all files in place
go-reorder -w ./...
```

## Before/After Example

**Before** - declarations scattered throughout the file:
```go
package user

func validateEmail(email string) bool { return email != "" }

type User struct { ID int; Name string }

const MaxUsers = 100

func NewUser(name string) *User { return &User{Name: name} }

var defaultUser = &User{Name: "guest"}

func (u *User) String() string { return u.Name }
```

**After** - organized by convention:
```go
package user

// Exported constants.
const (
    MaxUsers = 100
)

// Exported variables.
var (
    defaultUser = &User{Name: "guest"}
)

type User struct { ID int; Name string }

func NewUser(name string) *User { return &User{Name: name} }

func (u *User) String() string { return u.Name }

func validateEmail(email string) bool { return email != "" }
```

## Features

- **Configurable ordering** via TOML config files
- **CLI tool** for processing files and directories
- **Library API** for programmatic use
- Preserves all comments and documentation
- Groups types with their constructors and methods
- Handles enum types (iota blocks paired with their type definitions)
- Merges scattered const/var declarations into organized blocks
- Safety modes to prevent accidental code loss

## Installation

### CLI Tool

```bash
go install github.com/toejough/go-reorder/cmd/go-reorder@latest
```

### Library

```bash
go get github.com/toejough/go-reorder
```

## CLI Usage

```bash
# Process a single file (output to stdout)
go-reorder main.go

# Process and write back to file
go-reorder -w main.go

# Process all Go files in a directory recursively
go-reorder -w ./...

# Check if files need reordering (exit 1 if changes needed)
go-reorder -c ./...

# Show diff of what would change
go-reorder -d main.go

# Read from stdin, write to stdout
cat main.go | go-reorder -

# Use explicit config file
go-reorder --config=.go-reorder.toml -w .

# Verbose output (shows config file, mode, file count)
go-reorder -v -w ./...
```

### CLI Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--write` | `-w` | Write result to source file instead of stdout |
| `--check` | `-c` | Check if files are properly ordered (exit 1 if not) |
| `--diff` | `-d` | Display diff instead of reordered source |
| `--verbose` | `-v` | Show config and processing details |
| `--config` | | Path to config file |
| `--mode` | | Behavior mode: `strict`, `warn`, `append`, or `drop` |
| `--exclude` | | Exclude files matching pattern (can be repeated) |

### Check Mode Output

When `--check` finds files that need reordering, it shows details:

```
config: .go-reorder.toml

pkg/server/handler.go
  found:    Imports -> unexported functions -> Exported Types
  expected: Imports -> Exported Types -> unexported functions

pkg/server/utils.go
  sections: Imports -> Exported Functions
  issue:    within-section reordering needed (e.g., alphabetizing, type grouping)
```

## Integration

### Pre-commit Hook

Add to your `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/toejough/go-reorder
    rev: main  # or specific version
    hooks:
      - id: go-reorder        # Auto-fix files
      # OR
      - id: go-reorder-check  # Check only (CI-friendly)
```

### GitHub Actions

```yaml
# .github/workflows/lint.yml
name: Lint
on: [push, pull_request]

jobs:
  reorder-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'
      - run: go install github.com/toejough/go-reorder/cmd/go-reorder@latest
      - run: go-reorder -c ./...
```

### Makefile

```makefile
.PHONY: reorder reorder-check

reorder:
	go-reorder -w ./...

reorder-check:
	go-reorder -c ./...
```

### Mage

```go
// Reorder fixes declaration ordering in all Go files.
func Reorder() error {
    return sh.Run("go-reorder", "-w", "./...")
}

// ReorderCheck verifies declaration ordering without modifying files.
func ReorderCheck() error {
    return sh.Run("go-reorder", "-c", "./...")
}
```

## Configuration

### Config Discovery

The CLI automatically discovers config files by walking up from the processed file's directory. It stops when it finds:
1. `.go-reorder.toml` - uses this config
2. `.git` directory - stops searching, uses defaults
3. `go.mod` file - stops searching, uses defaults

This means you can have different configs for different modules in a monorepo.

### Example Config

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

| Mode | Description |
|------|-------------|
| `strict` | Error if code has no matching section in config (default) |
| `warn` | Append unmatched code at end with warning to stderr |
| `append` | Silently append unmatched code at end |
| `drop` | Discard unmatched code (dangerous - use for splitting files) |

### Available Sections

| Section | Description |
|---------|-------------|
| `imports` | Import declarations |
| `main` | The main() function |
| `init` | All init() functions (original order preserved) |
| `exported_consts` | Exported constant declarations |
| `exported_enums` | Exported enum types with their iota blocks |
| `exported_vars` | Exported variable declarations |
| `exported_types` | Exported type definitions with constructors and methods |
| `exported_funcs` | Exported standalone functions |
| `unexported_*` | Unexported equivalents of the above |
| `uncategorized` | Catch-all for anything not matching other sections |

### Type/Enum Layout Elements

For `type_layout`:
- `typedef` - The type definition itself
- `constructors` - Constructor functions (functions named `NewTypeName` or `NewXxxTypeName`)
- `exported_methods` - Exported methods on the type
- `unexported_methods` - Unexported methods on the type

For `enum_layout`:
- `typedef` - The enum type definition (e.g., `type Status int`)
- `iota` - The associated iota const block
- `exported_methods` / `unexported_methods` - Methods on the enum type

## Configuration Recipes

### Standard Library/Package

Default config works well. Types grouped with constructors and methods.

### Web Application (handlers last)

```toml
[sections]
order = [
  "imports",
  "exported_consts",
  "exported_vars",
  "exported_types",      # Models, services
  "unexported_types",    # Internal helpers
  "unexported_funcs",    # Internal helpers
  "exported_funcs",      # Handlers last - easy to find
  "uncategorized",
]
```

### CLI Tool (main first)

```toml
[sections]
order = [
  "imports",
  "main",                # Entry point at top
  "init",
  "exported_types",      # Command structs
  "exported_funcs",      # Command implementations
  "unexported_funcs",
  "uncategorized",
]
```

### Minimal (just organize, don't be strict)

```toml
[behavior]
mode = "append"  # Don't error on uncategorized code
```

## Library Usage

### Basic Example

```go
package main

import (
    "os"
    "github.com/toejough/go-reorder"
)

func main() {
    content, _ := os.ReadFile("example.go")

    // Reorder with default config
    reordered, err := reorder.Source(string(content))
    if err != nil {
        panic(err)
    }

    os.WriteFile("example.go", []byte(reordered), 0644)
}
```

### With Custom Config

```go
// Load config from file
cfg, err := reorder.LoadConfig(".go-reorder.toml")
if err != nil {
    panic(err)
}

// Or use default and modify
cfg := reorder.DefaultConfig()
cfg.Behavior.Mode = "append"

// Reorder with config
reordered, err := reorder.SourceWithConfig(string(content), cfg)
```

### API Functions

| Function | Description |
|----------|-------------|
| `Source(src string)` | Reorder source code with default config |
| `SourceWithConfig(src string, cfg *Config)` | Reorder with custom config |
| `File(file *dst.File)` | Reorder a parsed DST file in place |
| `FileWithConfig(file *dst.File, cfg *Config)` | Reorder parsed file with config |
| `DefaultConfig()` | Get default configuration |
| `LoadConfig(path string)` | Load config from TOML file |
| `FindConfig(startDir string)` | Discover config file walking up directories |
| `AnalyzeSectionOrder(src string)` | Analyze current section order without modifying |

## Default Ordering

Without a config file, declarations are ordered as:

1. Imports
2. main() function
3. init() functions (original order preserved)
4. Exported constants
5. Exported enums (type + iota block + methods)
6. Exported variables
7. Exported types (type + constructors + methods)
8. Exported functions
9. Unexported constants
10. Unexported enums
11. Unexported variables
12. Unexported types
13. Unexported functions
14. Uncategorized (catch-all)

Within each section, declarations are sorted alphabetically. Types group their constructors (functions matching `New*TypeName`) and methods together.

## What This Tool Doesn't Do

- **Import ordering** - Use `goimports` or `gci` for that
- **Code formatting** - Use `gofmt` or `gofumpt`
- **Linting** - Use `golangci-lint`
- **Build constraint handling** - Files with `//go:build` are processed normally
- **cgo export comments** - `//export` comments are preserved but not specially handled
- **Cross-file analysis** - Each file is processed independently

## Troubleshooting

### "strict mode: code has no matching section"

Your config doesn't include a section for some code. Options:
1. Add the missing section to your config's `order` array
2. Add `uncategorized` to catch everything else
3. Use `mode = "append"` to be lenient

### Files not being discovered

The `./...` pattern uses Go's package discovery. For non-Go directories, specify paths explicitly:
```bash
go-reorder -w ./scripts/*.go ./tools/*.go
```

### Config not being found

Use `-v` (verbose) to see which config is loaded:
```bash
go-reorder -v -c ./...
# Output: config: /path/to/.go-reorder.toml
# Or:     config: using defaults
```

### Wrong ordering after reorder

1. Check your config file syntax (TOML)
2. Verify section names are spelled correctly
3. Use `-d` to see what would change before `-w`

### Constructor not grouping with type

Constructors must match the pattern `New*TypeName` where `TypeName` is exact. Examples:
- `NewUser` matches `User`
- `NewMockUser` matches `User`
- `CreateUser` does NOT match (no `New` prefix)

## License

MIT - See LICENSE file for details.

## Credits

Built with [dave/dst](https://github.com/dave/dst) for AST manipulation that preserves comments.

Originally developed as part of the [imptest](https://github.com/toejough/imptest) project.
