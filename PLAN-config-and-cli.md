# Plan: TOML Config + CLI for go-reorder

## Current State

Ordering is hardcoded in two places:
- `expectedPositions` map (line 41-54) - defines section order for analysis
- `reassembleDeclarations` (line 614-763) - assembles declarations in fixed order

Current hardcoded order:
1. imports
2. main()
3. exported constants
4. exported enums (type + iota block + methods)
5. exported variables
6. exported types (type + constructors + methods)
7. exported functions
8. unexported constants
9. unexported enums
10. unexported variables
11. unexported types
12. unexported functions

## Goal

1. **TOML config** - Make ordering configurable
2. **CLI wrapper** - `go-reorder` command for users who don't want library usage

---

## Part 1: TOML Config

### Config Schema

```toml
# .go-reorder.toml

[sections]
# Order of major sections. Each section can be exported, unexported, or both.
# "both" means exported first, then unexported of that category.
order = [
  "imports",
  "main",
  "const:exported",
  "enum:exported",
  "var:exported",
  "init",
  "type:exported",
  "func:exported",
  "const:unexported",
  "enum:unexported",
  "var:unexported",
  "type:unexported",
  "func:unexported",
  "uncategorized",
]

[grouping]
# Group type definitions with their methods
types_with_methods = true

# Group constructors (NewFoo) with their types
types_with_constructors = true

# Merge scattered const/var declarations into single blocks
merge_const_blocks = true
merge_var_blocks = true

# Keep enum type and iota const block together
enums_with_iota = true

[constructors]
# Patterns that identify constructor functions
# "New" matches NewFoo, NewFooWithOptions, etc.
prefixes = ["New"]

[sorting]
# How to sort within sections: "alphabetical" or "preserve"
within_sections = "alphabetical"

[types]
# Layout of type groups. Components:
#   typedef              - the type definition itself
#   constructors:exported/unexported/both
#   methods:exported/unexported/both
# ":both" expands to exported then unexported
layout = ["typedef", "constructors:both", "methods:both"]

[enums]
# Layout of enum groups (type + iota block)
layout = ["typedef", "iota", "methods:both"]
```

### Go Types

```go
// config.go

type Config struct {
    Sections     SectionConfig     `toml:"sections"`
    Grouping     GroupingConfig    `toml:"grouping"`
    Constructors ConstructorConfig `toml:"constructors"`
    Sorting      SortingConfig     `toml:"sorting"`
    Types        TypeLayoutConfig  `toml:"types"`
    Enums        EnumLayoutConfig  `toml:"enums"`
}

type SectionConfig struct {
    Order []string `toml:"order"`
}

type GroupingConfig struct {
    TypesWithMethods       bool `toml:"types_with_methods"`
    TypesWithConstructors  bool `toml:"types_with_constructors"`
    MergeConstBlocks       bool `toml:"merge_const_blocks"`
    MergeVarBlocks         bool `toml:"merge_var_blocks"`
    EnumsWithIota          bool `toml:"enums_with_iota"`
}

type ConstructorConfig struct {
    Prefixes []string `toml:"prefixes"`
}

type SortingConfig struct {
    WithinSections string `toml:"within_sections"` // "alphabetical" | "preserve"
}

type TypeLayoutConfig struct {
    Layout []string `toml:"layout"` // e.g., ["typedef", "constructors:both", "methods:both"]
}

type EnumLayoutConfig struct {
    Layout []string `toml:"layout"` // e.g., ["typedef", "iota", "methods:both"]
}
```

### Default Config

```go
func DefaultConfig() *Config {
    return &Config{
        Sections: SectionConfig{
            Order: []string{
                "imports",
                "main",
                "const:exported",
                "enum:exported",
                "var:exported",
                "init",
                "type:exported",
                "func:exported",
                "const:unexported",
                "enum:unexported",
                "var:unexported",
                "type:unexported",
                "func:unexported",
                "uncategorized",
            },
        },
        Grouping: GroupingConfig{
            TypesWithMethods:      true,
            TypesWithConstructors: true,
            MergeConstBlocks:      true,
            MergeVarBlocks:        true,
            EnumsWithIota:         true,
        },
        Constructors: ConstructorConfig{
            Prefixes: []string{"New"},
        },
        Sorting: SortingConfig{
            WithinSections: "alphabetical",
        },
        Types: TypeLayoutConfig{
            Layout: []string{"typedef", "constructors:both", "methods:both"},
        },
        Enums: EnumLayoutConfig{
            Layout: []string{"typedef", "iota", "methods:both"},
        },
    }
}
```

---

## Part 2: Config-Driven Reassembly

### Current Problem

`reassembleDeclarations` is procedural - it has 12 hardcoded blocks of code, one per section:

```go
// Current: hardcoded sequence
decls = append(decls, cat.imports...)
if cat.main != nil { decls = append(decls, cat.main) }
if len(cat.exportedConsts) > 0 { ... }
for _, enumGrp := range cat.exportedEnums { ... }
// ... repeat for each section
```

### Solution: Section Emitters

Create a map of section emitters that can be invoked in config order:

```go
// emitter.go

type sectionEmitter func(cat *categorizedDecls, cfg *Config) []dst.Decl

var emitters = map[string]sectionEmitter{
    "imports":          emitImports,
    "main":             emitMain,
    "const:exported":   emitExportedConsts,
    "const:unexported": emitUnexportedConsts,
    "enum:exported":    emitExportedEnums,
    "enum:unexported":  emitUnexportedEnums,
    "var:exported":     emitExportedVars,
    "var:unexported":   emitUnexportedVars,
    "type:exported":    emitExportedTypes,
    "type:unexported":  emitUnexportedTypes,
    "func:exported":    emitExportedFuncs,
    "func:unexported":  emitUnexportedFuncs,
}

func emitImports(cat *categorizedDecls, cfg *Config) []dst.Decl {
    return cat.imports
}

func emitMain(cat *categorizedDecls, cfg *Config) []dst.Decl {
    if cat.main == nil {
        return nil
    }
    return []dst.Decl{cat.main}
}

func emitExportedConsts(cat *categorizedDecls, cfg *Config) []dst.Decl {
    if len(cat.exportedConsts) == 0 {
        return nil
    }
    if cfg.Grouping.MergeConstBlocks {
        return []dst.Decl{mergeConstSpecs(cat.exportedConsts, "Exported constants.")}
    }
    // Return as separate declarations
    // ...
}

func emitExportedTypes(cat *categorizedDecls, cfg *Config) []dst.Decl {
    var decls []dst.Decl
    for _, typeGrp := range cat.exportedTypes {
        decls = append(decls, emitTypeGroup(typeGrp, cfg.Types.Layout)...)
    }
    return decls
}

// emitTypeGroup emits a single type according to layout config
func emitTypeGroup(tg *typeGroup, layout []string) []dst.Decl {
    var decls []dst.Decl
    for _, part := range layout {
        switch part {
        case "typedef":
            if tg.typeDecl != nil {
                decls = append(decls, tg.typeDecl)
            }
        case "constructors:exported":
            decls = append(decls, funcsToDecls(filterExported(tg.constructors, true))...)
        case "constructors:unexported":
            decls = append(decls, funcsToDecls(filterExported(tg.constructors, false))...)
        case "constructors:both":
            decls = append(decls, funcsToDecls(filterExported(tg.constructors, true))...)
            decls = append(decls, funcsToDecls(filterExported(tg.constructors, false))...)
        case "methods:exported":
            decls = append(decls, funcsToDecls(tg.exportedMethods)...)
        case "methods:unexported":
            decls = append(decls, funcsToDecls(tg.unexportedMethods)...)
        case "methods:both":
            decls = append(decls, funcsToDecls(tg.exportedMethods)...)
            decls = append(decls, funcsToDecls(tg.unexportedMethods)...)
        }
    }
    return decls
}

// Similar pattern for emitEnumGroup with "typedef", "iota", "methods:*"
```

### New reassembleDeclarations

```go
func reassembleDeclarations(cat *categorizedDecls, cfg *Config) []dst.Decl {
    var decls []dst.Decl

    for _, section := range cfg.Sections.Order {
        emitter, ok := emitters[section]
        if !ok {
            continue // skip unknown sections
        }
        sectionDecls := emitter(cat, cfg)
        decls = append(decls, sectionDecls...)
    }

    return decls
}
```

### API Changes

```go
// Keep backward compatibility with optional config

func Source(src string) (string, error) {
    return SourceWithConfig(src, DefaultConfig())
}

func SourceWithConfig(src string, cfg *Config) (string, error) {
    // ... parse, reorder with config, print
}

func File(file *dst.File) error {
    return FileWithConfig(file, DefaultConfig())
}

func FileWithConfig(file *dst.File, cfg *Config) error {
    // ... reorder with config
}
```

---

## Part 3: CLI

### Structure

```
cmd/
  go-reorder/
    main.go
```

### Usage

```
go-reorder [flags] [paths...]

Flags:
  -config string    Path to config file (default: .go-reorder.toml in current or parent dirs)
  -check            Check mode - exit 1 if files need reordering, don't modify
  -write            Write changes back to files (default: print to stdout)
  -diff             Show diff of changes
  -list             List files that would be changed

Examples:
  go-reorder .                     # Check all .go files recursively
  go-reorder -write ./pkg/...      # Fix all files in pkg/
  go-reorder -diff main.go         # Show what would change
```

### Implementation Sketch

```go
// cmd/go-reorder/main.go

func main() {
    cfg := parseFlags()
    config := loadConfig(cfg.configPath)

    files := discoverFiles(cfg.paths)

    var exitCode int
    for _, file := range files {
        result := processFile(file, config, cfg.mode)
        if result.changed {
            exitCode = 1
            switch cfg.mode {
            case modeCheck:
                fmt.Printf("%s needs reordering\n", file)
            case modeDiff:
                printDiff(file, result.original, result.reordered)
            case modeWrite:
                writeFile(file, result.reordered)
                fmt.Printf("fixed %s\n", file)
            }
        }
    }

    os.Exit(exitCode)
}

func discoverFiles(paths []string) []string {
    // Walk directories, find *.go files
    // Skip vendor/, testdata/, _*, .*
    // Respect .gitignore if present
}
```

---

## Implementation Order

1. **Config types + loading** - `config.go` with types, defaults, TOML loading
2. **Refactor to emitters** - Break `reassembleDeclarations` into emitter functions
3. **Wire config through** - Add `*WithConfig` variants, update internals
4. **Tests for config** - Verify different orderings work
5. **CLI skeleton** - Basic flag parsing and file discovery
6. **CLI modes** - Implement check/write/diff/list
7. **CLI tests** - Integration tests with temp directories

---

## Safety: No Deletion Guarantee

### Problem

Code that doesn't match our patterns could be silently dropped:
- Standalone comments between declarations
- Build constraints (`//go:build`)
- CGo preambles (`import "C"` with comment block)
- `init()` functions (special - shouldn't be alphabetized)
- Future Go syntax we don't recognize
- Methods for types defined in other files

### Solution: Catch-All + Validation

```go
type categorizedDecls struct {
    // ... existing fields ...

    // Catch-all for anything we don't recognize
    uncategorized []dst.Decl

    // Track what we've seen for validation
    inputCount int
}
```

**Categorization changes:**

```go
func categorizeDeclarations(file *dst.File, cfg *Config) *categorizedDecls {
    cat := &categorizedDecls{
        inputCount: len(file.Decls),
    }

    for _, decl := range file.Decls {
        categorized := false

        switch d := decl.(type) {
        case *dst.GenDecl:
            // ... existing logic, set categorized = true when matched ...
        case *dst.FuncDecl:
            // ... existing logic, set categorized = true when matched ...
        }

        if !categorized {
            cat.uncategorized = append(cat.uncategorized, decl)
        }
    }

    return cat
}
```

**Emit uncategorized at the end:**

```go
var emitters = map[string]sectionEmitter{
    // ... existing emitters ...
    "uncategorized": emitUncategorized,
}

// Default config always ends with uncategorized
Order: []string{
    // ... normal sections ...
    "uncategorized",  // Always last - catches anything we missed
}
```

**Validation before returning:**

```go
func reassembleDeclarations(cat *categorizedDecls, cfg *Config) ([]dst.Decl, error) {
    var decls []dst.Decl

    for _, section := range cfg.Sections.Order {
        // ... emit sections ...
    }

    // CRITICAL: Validate we didn't lose anything
    if len(decls) != cat.inputCount {
        return nil, fmt.Errorf(
            "declaration count mismatch: input %d, output %d (would lose code)",
            cat.inputCount, len(decls),
        )
    }

    return decls, nil
}
```

### Special Cases

**Build constraints:** Must stay at top of file, before package declaration. The `dst` library handles this - they're attached to the file, not declarations. Verify we preserve `file.Decs`.

**CGo:** `import "C"` must stay with its preceding comment block. Detect and keep as unit:
```go
if isImportC(genDecl) {
    cat.cgoImport = genDecl  // Special field, emitted right after regular imports
}
```

**`init()` functions:** Just another section - user places it in the order where they want it:
```toml
order = [
  "imports",
  "main",
  "const:exported",
  "var:exported",
  "init",              # placed after vars, before types
  "type:exported",
  # ...
]
```

Track separately (preserve original order since there can be multiple):
```go
type categorizedDecls struct {
    // ...
    initFuncs []*dst.FuncDecl  // init() functions, order preserved (not sorted)
}

var emitters = map[string]sectionEmitter{
    // ...
    "init": emitInitFuncs,
}
```

### Testing the Guarantee

```go
func TestNoCodeLoss(t *testing.T) {
    // Fuzz test: generate random valid Go files
    // Reorder them
    // Parse both original and result
    // Compare: same declarations (by identity/content), possibly different order
}

func TestRoundTrip(t *testing.T) {
    // Reorder twice - second pass should be no-op
    result1 := reorder(input)
    result2 := reorder(result1)
    assert.Equal(t, result1, result2)
}
```

---

## Incomplete Config Handling

### Problem

User's config might not cover all code in their files:
- Config only lists `func:exported` but file has unexported functions
- Config omits `uncategorized` entirely
- Config omits `init` but file has init functions

### Solution: Explicit Mode Flag

**`--mode` flag with four options:**

| Mode | Behavior |
|------|----------|
| `strict` | Error if code has no home (default) |
| `warn` | Append at end + warning |
| `append` | Silently append at end |
| `drop` | Intentionally discard unmatched code |

**Examples:**

```bash
# Default - error on incomplete config
$ go-reorder .
error: main.go has unexported functions but config has no "func:unexported" section
hint: use --mode=append to add unmatched code at end
      or --mode=drop to intentionally discard it

# Append mode - keep everything, put strays at end
$ go-reorder --mode=warn .
warning: main.go: appending 3 unexported functions (no matching section in config)

# Drop mode - intentionally split a file
$ go-reorder --mode=drop --config=exported-only.toml src.go > exported.go
$ go-reorder --mode=drop --config=unexported-only.toml src.go > unexported.go
```

**Config default:**

```toml
[behavior]
# "strict" (default) | "warn" | "append" | "drop"
unmatched = "strict"
```

CLI `--mode` flag overrides config setting.

### Implementation

```go
func reassembleDeclarations(cat *categorizedDecls, cfg *Config) ([]dst.Decl, error) {
    var decls []dst.Decl
    emitted := make(map[string]bool)

    // Emit sections in config order
    for _, section := range cfg.Sections.Order {
        emitter, ok := emitters[section]
        if !ok {
            continue
        }
        emitted[section] = true
        decls = append(decls, emitter(cat, cfg)...)
    }

    // Check for unemitted code
    unemitted := findUnemittedCode(cat, emitted)
    if len(unemitted) > 0 {
        switch cfg.Behavior.Mode {
        case "strict":
            return nil, &UnmatchedCodeError{Sections: unemitted}
        case "warn":
            log.Printf("warning: appending unmatched sections: %v", unemitted)
            fallthrough
        case "append":
            for _, section := range unemitted {
                decls = append(decls, emitters[section](cat, cfg)...)
            }
        case "drop":
            // Intentionally discard - user wants to split/filter
            // Skip the count validation below
            return decls, nil
        }
    }

    // Final validation - count must match (unless drop mode)
    if len(decls) != cat.inputCount {
        return nil, fmt.Errorf("declaration count mismatch: input %d, output %d",
            cat.inputCount, len(decls))
    }

    return decls, nil
}

func findUnemittedCode(cat *categorizedDecls, emitted map[string]bool) []string {
    var missing []string

    if len(cat.unexportedFuncs) > 0 && !emitted["func:unexported"] {
        missing = append(missing, "func:unexported")
    }
    if len(cat.initFuncs) > 0 && !emitted["init"] {
        missing = append(missing, "init")
    }
    // ... check all sections

    return missing
}
```

### Error Messages

Clear, actionable errors:

```
error: src/parser.go has code with no matching config section:
  - 2 init functions (add "init" to sections.order)
  - 5 unexported variables (add "var:unexported" to sections.order)

hint: use --mode=append to add unmatched code at end
      use --mode=drop to intentionally discard
      or add missing sections to your .go-reorder.toml
```

---

## Config Discovery

### Default Behavior

Walk up from current directory to project root (`.git`, `go.mod`, or filesystem root), looking for `.go-reorder.toml`.

```bash
# Uses first .go-reorder.toml found walking up from cwd
$ go-reorder .

# Explicit config path overrides discovery
$ go-reorder --config=/path/to/custom.toml .
```

### Per-Directory Configs

Configs in subdirectories override parent configs for that subtree:

```
project/
├── .go-reorder.toml          # applies to project root
├── pkg/
│   ├── .go-reorder.toml      # overrides for pkg/ and below
│   └── parser/
│       └── parser.go         # uses pkg/.go-reorder.toml
├── cmd/
│   └── main.go               # uses project/.go-reorder.toml
└── internal/
    ├── .go-reorder.toml      # overrides for internal/
    └── util/
        ├── .go-reorder.toml  # overrides for util/ only
        └── helpers.go        # uses internal/util/.go-reorder.toml
```

**CLI output is loud about which config applies:**

```
$ go-reorder .
using config: .go-reorder.toml
  cmd/main.go ... ok
  pkg/parser/parser.go ... ok
using config: pkg/.go-reorder.toml
  pkg/lexer/lexer.go ... ok
using config: internal/.go-reorder.toml
  internal/core.go ... ok
using config: internal/util/.go-reorder.toml
  internal/util/helpers.go ... ok
```

---

## File Patterns

Include and exclude patterns using fish glob syntax. Later patterns override earlier ones.

### Config

```toml
[files]
# Patterns evaluated in order, later overrides earlier
patterns = [
  "**/*.go",           # include all Go files
  "!vendor/**",        # exclude vendor
  "!**/*_test.go",     # exclude tests
  "vendor/kept/**",    # but include this specific vendor path
]
```

### CLI Flags

```bash
# Override config patterns
$ go-reorder --include="**/*.go" --exclude="vendor/**" .

# Multiple patterns, order matters
$ go-reorder --include="**/*.go" --exclude="**/*_test.go" --include="critical_test.go" .
```

### Pattern Syntax (Fish Globs)

| Pattern | Matches |
|---------|---------|
| `*.go` | Go files in current dir |
| `**/*.go` | Go files recursively |
| `!vendor/**` | Exclude vendor tree |
| `pkg/*/` | Directories directly under pkg |
| `{a,b}/*.go` | Go files in a/ or b/ |

---

## Stdin/Stdout Support

```bash
# Pipe mode - read stdin, write stdout
$ cat input.go | go-reorder > output.go

# Explicit stdin
$ go-reorder - < input.go > output.go

# With config (required for stdin since no directory to discover from)
$ go-reorder --config=my.toml - < input.go
```

When reading from stdin:
- Config must be explicit (`--config`) or use defaults
- Mode defaults apply (`--mode=strict` unless overridden)
- Output always goes to stdout
