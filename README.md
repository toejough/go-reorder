# go-reorder

![go-reorder](assets/reorder.png)

A Go code reordering tool that organizes declarations according to configurable conventions.

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
go-reorder -w ./pkg/...

# Check if files need reordering (exit 1 if changes needed)
go-reorder -c .

# Show diff of what would change
go-reorder -d main.go

# Read from stdin, write to stdout
cat main.go | go-reorder -

# Use explicit config file
go-reorder --config=.go-reorder.toml -w .
```

### CLI Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--write` | `-w` | Write result to source file instead of stdout |
| `--check` | `-c` | Check if files are properly ordered (exit 1 if not) |
| `--diff` | `-d` | Display diff instead of reordered source |
| `--config` | | Path to config file |
| `--mode` | | Behavior mode: `strict`, `warn`, `append`, or `drop` |
| `--exclude` | | Exclude files matching pattern (can be repeated) |

### Behavior Modes

| Mode | Description |
|------|-------------|
| `strict` | Error if code has no matching section in config (default) |
| `warn` | Append unmatched code at end with warning |
| `append` | Silently append unmatched code at end |
| `drop` | Discard unmatched code (useful for splitting files) |

## Configuration

Create a `.go-reorder.toml` file in your project root. The CLI automatically discovers config files by walking up from the processed file's directory until it finds `.go-reorder.toml`, `.git`, or `go.mod`.

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

### Available Sections

- `imports` - Import declarations
- `main` - The main() function
- `init` - All init() functions (order preserved)
- `exported_consts` - Exported constant declarations
- `exported_enums` - Exported enum types with their iota blocks
- `exported_vars` - Exported variable declarations
- `exported_types` - Exported type definitions with constructors and methods
- `exported_funcs` - Exported standalone functions
- `unexported_consts`, `unexported_enums`, `unexported_vars`, `unexported_types`, `unexported_funcs` - Unexported equivalents
- `uncategorized` - Catch-all for anything not matching other sections

### Type/Enum Layout Elements

For `type_layout`:
- `typedef` - The type definition itself
- `constructors` - Constructor functions (NewXxx)
- `exported_methods` - Exported methods
- `unexported_methods` - Unexported methods

For `enum_layout`:
- `typedef` - The enum type definition
- `iota` - The associated iota const block
- `exported_methods` - Exported methods
- `unexported_methods` - Unexported methods

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
| `Source(src string)` | Reorder with default config |
| `SourceWithConfig(src string, cfg *Config)` | Reorder with custom config |
| `DefaultConfig()` | Get default configuration |
| `LoadConfig(path string)` | Load config from TOML file |
| `FindConfig(startDir string)` | Discover config file walking up directories |

## Default Ordering

Without a config file, declarations are ordered as:

1. Imports
2. main() function
3. init() functions
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

Within each section, declarations are sorted alphabetically. Types group their constructors and methods together.

## License

See LICENSE file for details.

## Credits

Originally developed as part of the [imptest](https://github.com/toejough/imptest) project.
