# Development Plan: go-reorder Config + CLI

## Methodology

### TDD Workflow

Every feature follows Red-Green-Refactor:

1. **Red**: Write failing test(s) that define expected behavior
2. **Green**: Write minimal code to make tests pass
3. **Refactor**: Clean up while keeping tests green

Tests are written first, committed separately from implementation when substantial.

### UAT Checkpoints

User acceptance checkpoints where Joe provides feedback before proceeding:

| Checkpoint | After Phase | What to Review |
|------------|-------------|----------------|
| UAT-1 | Phase 2 | Config loading + defaults work correctly |
| UAT-2 | Phase 4 | Emitter refactor produces identical output to current |
| UAT-3 | Phase 6 | Type/enum layouts configurable, safety guarantees work |
| UAT-4 | Phase 8 | CLI works end-to-end with all modes |
| UAT-5 | Phase 10 | Config discovery + file patterns work |

At each checkpoint:
- Demo the feature
- Joe tests with real files
- Gather feedback before next phase

### Libraries

| Purpose | Library | Rationale |
|---------|---------|-----------|
| Testing | `github.com/toejough/imptest` | Imperative test style, Joe's preferred |
| TOML parsing | `github.com/BurntSushi/toml` | Standard, well-maintained, struct tags |
| CLI | `github.com/toejough/targ` | Joe's arg parsing library |
| Glob matching | `github.com/bmatcuk/doublestar/v4` | `**` support, closest to fish globs |
| Diff output | `github.com/pmezard/go-difflib` | Python difflib port, unified diff format |

### Commit Strategy

Three commits per TDD cycle:

1. **Red**: Commit failing tests
   - `test(scope): add tests for X`
   - Tests compile but fail

2. **Green**: Commit implementation
   - `feat(scope): implement X` or `fix(scope): ...`
   - Tests pass

3. **Refactor**: Commit cleanup
   - `refactor(scope): clean up X`
   - Tests still pass + `golangci-lint run` passes

Skip refactor commit if no cleanup needed.

### Quality Gates

- `golangci-lint run` must pass before refactor commits
- `golangci-lint run` must pass before any phase is complete
- Especially critical during Phases 3 & 4 (structural refactoring)

---

## Build Plan

### Phase 1: Config Types + Defaults

**Goal**: Define config structures and default values.

**Tests** (imptest style):
```go
func TestDefaultConfig(t *testing.T) {
    test := imptest.New(t)
    cfg := DefaultConfig()

    test.Assert("has expected section count", len(cfg.Sections.Order) == 14)
    test.Assert("starts with imports", cfg.Sections.Order[0] == "imports")
    test.Assert("ends with uncategorized", cfg.Sections.Order[13] == "uncategorized")
    test.Assert("mode defaults to strict", cfg.Behavior.Mode == "strict")
    // ... etc
}

func TestConfigValidation(t *testing.T) {
    test := imptest.New(t)

    test.Run("valid config passes", func(test *imptest.T) {
        cfg := DefaultConfig()
        err := cfg.Validate()
        test.Assert("no error", err == nil)
    })

    test.Run("unknown section errors", func(test *imptest.T) {
        cfg := DefaultConfig()
        cfg.Sections.Order = []string{"bogus"}
        err := cfg.Validate()
        test.Assert("returns error", err != nil)
    })
}
```

**Implementation**:
- `config.go`: Config struct, DefaultConfig(), Validate()
- No TOML loading yet - just the types

**Files**: `config.go`, `config_test.go`

---

### Phase 2: TOML Loading

**Goal**: Load config from TOML files, merge with defaults.

**Tests**:
```go
func TestLoadConfig(t *testing.T)
  - Loads valid TOML file
  - Missing file returns defaults
  - Partial config merges with defaults
  - Invalid TOML returns error
  - Invalid values return error with line number

func TestConfigMerge(t *testing.T)
  - Explicit values override defaults
  - Unset values keep defaults
  - Empty arrays don't override (vs explicit empty)
```

**Implementation**:
- `config.go`: LoadConfig(path string) (*Config, error)
- Merge logic for partial configs

**UAT-1**: Demo config loading, test with sample TOML files.

---

### Phase 3: Emitter Infrastructure

**Goal**: Create emitter map and infrastructure without changing behavior.

**Tests**:
```go
func TestEmitterRegistry(t *testing.T)
  - All expected section names have emitters
  - Unknown section names return nil emitter

func TestEmitterSignatures(t *testing.T)
  - Each emitter returns []dst.Decl
  - Emitters handle empty input gracefully
```

**Implementation**:
- `emitter.go`: sectionEmitter type, emitters map
- Extract existing logic into individual emitter functions
- No behavior change yet - same hardcoded order

**Quality**: Run `golangci-lint run` after each extraction to catch issues early.

**Files**: `emitter.go`, `emitter_test.go`

---

### Phase 4: Data-Driven Reassembly

**Goal**: Replace hardcoded reassembleDeclarations with config-driven loop.

**Tests**:
```go
func TestReassembleWithDefaultConfig(t *testing.T)
  - Output identical to current behavior
  - All existing tests still pass

func TestReassembleWithCustomOrder(t *testing.T)
  - Sections appear in config order
  - Reversed order works
  - Subset of sections works (with mode=drop)

func TestReassembleRoundTrip(t *testing.T)
  - reorder(reorder(x)) == reorder(x)
```

**Implementation**:
- Modify `reassembleDeclarations` to iterate `cfg.Sections.Order`
- Add `*WithConfig` API variants
- Keep existing API as wrapper with DefaultConfig

**Quality**: Run `golangci-lint run` continuously during refactor. This is the highest-risk phase.

**Critical**: All existing tests must pass unchanged.

**UAT-2**: Verify output identical to before refactor.

---

### Phase 5: Type/Enum Layout Config

**Goal**: Make type and enum internal ordering configurable.

**Tests**:
```go
func TestTypeLayout(t *testing.T)
  - Default layout: typedef, constructors:both, methods:both
  - Custom layout: methods before constructors
  - Partial layout: only exported methods
  - Empty typedef (methods-only files)

func TestEnumLayout(t *testing.T)
  - Default layout: typedef, iota, methods:both
  - Custom layout: iota before typedef
  - Methods ordering follows layout
```

**Implementation**:
- `emitTypeGroup(tg *typeGroup, layout []string) []dst.Decl`
- `emitEnumGroup(eg *enumGroup, layout []string) []dst.Decl`
- Wire layout config into emitters

**Files**: Modify `emitter.go`

---

### Phase 6: Safety Guarantees

**Goal**: Catch-all bucket, count validation, mode handling.

**Tests**:
```go
func TestUncategorizedCatchAll(t *testing.T)
  - Unknown declarations go to uncategorized
  - Uncategorized emitted when in config order

func TestCountValidation(t *testing.T)
  - Input count == output count (non-drop modes)
  - Mismatch returns error

func TestModeStrict(t *testing.T)
  - Errors on unmatched code
  - Error message lists missing sections
  - Error message suggests fixes

func TestModeWarn(t *testing.T)
  - Appends unmatched code
  - Logs warning

func TestModeAppend(t *testing.T)
  - Appends unmatched code silently

func TestModeDrop(t *testing.T)
  - Discards unmatched code
  - No count validation error
  - Useful for file splitting

func TestNoCodeLoss(t *testing.T)
  - Fuzz test with random valid Go files
  - Parse original and result
  - Same declarations exist (different order ok)
```

**Implementation**:
- Add `uncategorized` field to categorizedDecls
- Add `inputCount` tracking
- Implement mode switch in reassembleDeclarations
- `findUnemittedCode()` helper

**UAT-3**: Demo safety with intentionally bad configs, test drop mode for splitting.

---

### Phase 7: CLI Skeleton

**Goal**: Basic CLI that reads files and applies reordering.

**Tests**:
```go
func TestCLIHelp(t *testing.T)
  - --help shows usage
  - All flags documented

func TestCLIVersion(t *testing.T)
  - --version shows version

func TestCLIBasicRun(t *testing.T)
  - Processes single file
  - Processes directory recursively
  - Respects --write flag
  - Respects --check flag (exit code)
  - Respects --diff flag (shows diff)
```

**Implementation**:
- `cmd/go-reorder/main.go`
- Flag parsing with targ
- Basic file discovery (all .go files)
- Wire to library

**Files**: `cmd/go-reorder/main.go`, `cmd/go-reorder/main_test.go`

---

### Phase 8: CLI Modes + Config Flag

**Goal**: Full mode support and explicit config path.

**Tests**:
```go
func TestCLIModeFlag(t *testing.T)
  - --mode=strict errors on unmatched
  - --mode=warn warns and appends
  - --mode=append silently appends
  - --mode=drop discards

func TestCLIConfigFlag(t *testing.T)
  - --config loads specified file
  - Missing config file errors
  - Config values applied correctly

func TestCLIOutput(t *testing.T)
  - Shows file being processed
  - Shows config being used
  - Shows warnings/errors clearly
```

**Implementation**:
- Add --mode flag
- Add --config flag
- Wire flags to library options

**UAT-4**: End-to-end CLI testing with various modes and configs.

---

### Phase 9: Config Discovery

**Goal**: Walk up directories to find config, per-directory overrides.

**Tests**:
```go
func TestConfigDiscoveryWalkUp(t *testing.T)
  - Finds config in current dir
  - Finds config in parent dir
  - Stops at .git boundary
  - Stops at go.mod boundary
  - Stops at filesystem root

func TestConfigDiscoveryPerDirectory(t *testing.T)
  - Subdir config overrides parent
  - Nested overrides work
  - Files use nearest ancestor config

func TestConfigDiscoveryOutput(t *testing.T)
  - CLI shows which config applies to each file
  - Config changes logged when switching
```

**Implementation**:
- `findConfig(startDir string) (string, error)`
- `configForFile(filePath string, rootConfig string) (*Config, error)`
- Integrate into CLI file processing loop

**Files**: `config.go` (discovery), `cmd/go-reorder/main.go`

---

### Phase 10: File Patterns

**Goal**: Include/exclude patterns with fish glob syntax.

**Tests**:
```go
func TestFilePatterns(t *testing.T)
  - "**/*.go" matches recursively
  - "!vendor/**" excludes vendor
  - Later patterns override earlier
  - "!**/*_test.go" then "important_test.go" works

func TestFilePatternsConfig(t *testing.T)
  - [files].patterns from config used
  - Empty patterns = default (all .go)

func TestFilePatternsFlags(t *testing.T)
  - --include adds include pattern
  - --exclude adds exclude pattern
  - Flag patterns override config
```

**Implementation**:
- `matchesPatterns(path string, patterns []string) bool`
- Integrate doublestar library
- Add --include/--exclude flags
- Wire into file discovery

**UAT-5**: Test with complex directory structures, verify patterns work.

---

### Phase 11: Stdin/Stdout

**Goal**: Support piping for single-file processing.

**Tests**:
```go
func TestStdinStdout(t *testing.T)
  - "-" argument reads stdin
  - Output goes to stdout
  - --config required (no discovery)
  - --config optional (uses defaults)
  - --mode works with stdin

func TestStdinErrors(t *testing.T)
  - Invalid Go code errors clearly
  - Mode=strict errors go to stderr
```

**Implementation**:
- Detect "-" as stdin sentinel
- Read from os.Stdin, write to os.Stdout
- Skip file discovery for stdin mode

---

### Phase 12: Polish + Documentation

**Goal**: README, examples, edge cases.

**Tasks**:
- Update README with CLI usage
- Add example configs for common setups
- Document all flags and config options
- Add CHANGELOG entry
- Test on real codebases (imptest, this repo)

**Final UAT**: Full walkthrough with Joe on real projects.

---

## Dependency Graph

```
Phase 1 (Config Types)
    ↓
Phase 2 (TOML Loading) → UAT-1
    ↓
Phase 3 (Emitter Infrastructure)
    ↓
Phase 4 (Data-Driven Reassembly) → UAT-2
    ↓
Phase 5 (Type/Enum Layout)
    ↓
Phase 6 (Safety Guarantees) → UAT-3
    ↓
Phase 7 (CLI Skeleton)
    ↓
Phase 8 (CLI Modes) → UAT-4
    ↓
Phase 9 (Config Discovery)
    ↓
Phase 10 (File Patterns) → UAT-5
    ↓
Phase 11 (Stdin/Stdout)
    ↓
Phase 12 (Polish) → Final UAT
```

---

## Risk Areas

| Risk | Mitigation |
|------|------------|
| Emitter refactor breaks existing behavior | Phase 4 requires all existing tests pass |
| Complex glob patterns slow | Benchmark, consider caching |
| Per-directory config confusing | Loud CLI output, clear docs |
| Drop mode loses code accidentally | Require explicit flag, warn in docs |
| Count validation false positives | Comprehensive fuzz testing in Phase 6 |

---

## Definition of Done

Each phase complete when:
1. **Red commit**: Failing tests committed
2. **Green commit**: Implementation committed, tests pass
3. **Refactor commit**: Cleanup committed (if needed), `golangci-lint run` passes
4. UAT passed (at checkpoints)

Project complete when:
1. All phases done
2. README updated
3. Works on real codebases
4. Joe approves final UAT
