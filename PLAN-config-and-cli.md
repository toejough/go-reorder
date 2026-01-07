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

## Open Questions

1. **Config discovery** - Walk up directories looking for `.go-reorder.toml`? Or require explicit path?
2. **Per-directory configs** - Allow different configs in subdirectories?
3. **Ignore patterns** - Config option for files/dirs to skip? Or rely on .gitignore?
4. **Stdin/stdout** - Support `go-reorder < input.go > output.go`?
