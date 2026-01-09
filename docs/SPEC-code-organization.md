# Code Organization Spec: go-reorder

## Current State

### File Structure

```
go-reorder/
├── cmd/
│   ├── demo/                    # Demo CLI (unclear purpose)
│   │   ├── examples/            # Example input files
│   │   └── main.go              # 40 lines
│   ├── go-reorder/              # Main CLI
│   │   ├── main.go              # 349 lines (CLI + file discovery + processing)
│   │   └── main_test.go         # 561 lines
│   └── reorder-demo/            # Another demo (duplicate?)
│       └── main.go              # 41 lines
├── config.go                    # 262 lines (Config types + loading + discovery)
├── config_test.go               # 295 lines
├── emitter.go                   # 205 lines (Section emitters)
├── emitter_test.go              # 39 lines (minimal coverage)
├── reorder.go                   # 1033 lines (EVERYTHING ELSE)
├── reorder_test.go              # 944 lines (core reorder tests)
├── reorder_config_test.go       # 279 lines (config-aware reorder tests)
└── README.md, PLAN-*.md, etc.
```

### Problems

#### 1. **reorder.go is a 1000-line monolith**

Contains unrelated concerns:
- Public API (`Source`, `SourceWithConfig`, `File`, `AnalyzeSectionOrder`)
- AST categorization (`categorizedDecls`, `typeGroup`, `enumGroup`, `categorizeDeclarations`)
- Reassembly logic (`reassembleDeclarations`, `reassembleDeclarationsWithConfig`)
- AST utilities (`isExported`, `extractTypeName`, `containsIota`, `isIotaBlock`)
- Const/var merging (`mergeConstSpecs`, `mergeVarSpecs`)
- Section identification (`identifySection`)
- Sorting (`sortCategorized`)

#### 2. **cmd/go-reorder/main.go does too much**

Contains:
- CLI struct and flag handling
- Test infrastructure (`testContext`, `testCtx`, `executeCLI`)
- File discovery (`discoverFiles`, `isExcluded`)
- File processing (`processFile`, `processStdin`)
- Core run logic (`run`)

The test infrastructure is interleaved with production code.

#### 3. **Test file organization is inconsistent**

- `reorder_test.go` - Tests core reordering
- `reorder_config_test.go` - Tests config-aware reordering (why separate?)
- `config_test.go` - Tests config loading
- `emitter_test.go` - Minimal emitter tests (39 lines for 205 lines of code)
- `cmd/go-reorder/main_test.go` - CLI integration tests

The split between `reorder_test.go` and `reorder_config_test.go` is artificial.

#### 4. **Duplicate demo commands**

- `cmd/demo/` - Loads and displays config
- `cmd/reorder-demo/` - Similar purpose?

Both are ~40 lines and unclear if either is needed.

#### 5. **emitter.go is under-tested**

39 lines of tests for 205 lines of code. The emitters are mostly tested indirectly through reorder tests, but edge cases aren't covered.

#### 6. **No clear separation between library and internals**

All types are in package `reorder`:
- Public API: `Source`, `SourceWithConfig`, `Config`, `LoadConfig`, etc.
- Internal types: `categorizedDecls`, `typeGroup`, `enumGroup`, `sectionEmitter`

Users importing the library see internal implementation details.

---

## Proposed State

### New File Structure

```
go-reorder/
├── cmd/
│   └── go-reorder/
│       ├── main.go              # CLI entry point only (~30 lines)
│       ├── cli.go               # CLI struct, flags, Run method
│       ├── cli_test.go          # CLI integration tests
│       ├── discover.go          # File discovery logic
│       ├── discover_test.go     # File discovery tests
│       └── process.go           # File/stdin processing
│
├── reorder.go                   # Public API only (~50 lines)
│                                # Source, SourceWithConfig, File, FileWithConfig
│                                # AnalyzeSectionOrder
│
├── config.go                    # Config types + DefaultConfig + Validate
├── config_load.go               # LoadConfig, FindConfig (TOML + discovery)
├── config_test.go               # All config tests
│
├── internal/
│   ├── categorize/
│   │   ├── categorize.go        # categorizedDecls, typeGroup, enumGroup
│   │   ├── categorize_test.go   # Categorization unit tests
│   │   └── sort.go              # sortCategorized
│   │
│   ├── emit/
│   │   ├── emit.go              # sectionEmitter, emitter registry
│   │   ├── emit_test.go         # Emitter unit tests
│   │   ├── types.go             # emitTypeGroup, emitEnumGroup
│   │   └── merge.go             # mergeConstSpecs, mergeVarSpecs
│   │
│   ├── ast/
│   │   └── util.go              # isExported, extractTypeName, containsIota, etc.
│   │
│   └── reassemble/
│       ├── reassemble.go        # reassembleDeclarations, reassembleWithConfig
│       └── reassemble_test.go   # Reassembly unit tests
│
├── reorder_test.go              # Integration tests (Source, SourceWithConfig)
│
└── testdata/                    # Test fixtures
    ├── basic.go
    ├── enums.go
    ├── methods_only.go
    └── ...
```

### Key Changes

#### 1. **Split reorder.go into focused modules**

| Current | Proposed Location |
|---------|-------------------|
| `Source`, `SourceWithConfig` | `reorder.go` (public API) |
| `categorizedDecls`, `typeGroup` | `internal/categorize/categorize.go` |
| `categorizeDeclarations` | `internal/categorize/categorize.go` |
| `sortCategorized` | `internal/categorize/sort.go` |
| `reassembleDeclarations` | `internal/reassemble/reassemble.go` |
| `sectionEmitter`, emitters | `internal/emit/emit.go` |
| `emitTypeGroup`, `emitEnumGroup` | `internal/emit/types.go` |
| `mergeConstSpecs`, `mergeVarSpecs` | `internal/emit/merge.go` |
| `isExported`, `extractTypeName` | `internal/ast/util.go` |

#### 2. **Clean up cmd/go-reorder**

| Current | Proposed |
|---------|----------|
| `main.go` (349 lines) | `main.go` (~30 lines, entry point only) |
| CLI + flags | `cli.go` |
| `discoverFiles`, `isExcluded` | `discover.go` |
| `processFile`, `processStdin`, `run` | `process.go` |
| Test infrastructure | `cli_test.go` (with `testContext` internal) |

#### 3. **Consolidate tests**

- Merge `reorder_test.go` and `reorder_config_test.go` → single `reorder_test.go`
- Move inline test fixtures to `testdata/` directory
- Add proper unit tests for `internal/` packages

#### 4. **Remove duplicate demos**

Delete `cmd/demo/` and `cmd/reorder-demo/`. If a demo is needed, add examples to README or a single `cmd/example/` with clear purpose.

#### 5. **Use internal/ for implementation details**

Anything not part of the public API goes in `internal/`:
- `categorizedDecls`, `typeGroup`, `enumGroup`
- `sectionEmitter`, emitter functions
- AST utilities

Users see only:
```go
import "github.com/toejough/go-reorder"

reorder.Source(src)
reorder.SourceWithConfig(src, cfg)
reorder.LoadConfig(path)
reorder.DefaultConfig()
reorder.FindConfig(dir)
```

---

## Migration Path

### Phase 1: Extract internal packages (no behavior change)

1. Create `internal/ast/util.go` with AST helpers
2. Create `internal/categorize/` with categorization logic
3. Create `internal/emit/` with emitters
4. Create `internal/reassemble/` with reassembly logic
5. Update `reorder.go` to import from `internal/`
6. All existing tests must pass

### Phase 2: Clean up CLI

1. Split `cmd/go-reorder/main.go` into `cli.go`, `discover.go`, `process.go`
2. Move test infrastructure into test file
3. Remove duplicate demo commands

### Phase 3: Consolidate tests

1. Merge `reorder_config_test.go` into `reorder_test.go`
2. Add unit tests for `internal/` packages
3. Move large test fixtures to `testdata/`

### Phase 4: Documentation

1. Update README to reflect new structure
2. Add godoc comments to public API
3. Remove PLAN-*.md files (historical, no longer needed)

---

## Benefits

| Problem | Solution | Benefit |
|---------|----------|---------|
| 1000-line monolith | Split into focused packages | Easier to understand, test, modify |
| Mixed public/internal | Use `internal/` | Clear API boundary |
| CLI does too much | Split into files by concern | Easier CLI maintenance |
| Inconsistent test organization | Consolidate, add unit tests | Better coverage, clearer intent |
| Duplicate demos | Remove | Less confusion |

---

## Open Questions

1. **Should config discovery be in the library or CLI only?**
   - Currently: Library (`FindConfig`)
   - Alternative: CLI-only, library just loads explicit paths

2. **Should we use a testdata/ directory?**
   - Pro: Cleaner tests, reusable fixtures
   - Con: More files to maintain

3. **How much unit testing for internal packages?**
   - Current: Mostly integration tests
   - Alternative: Unit tests for each internal package
