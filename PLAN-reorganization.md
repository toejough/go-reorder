# Implementation Plan: Code Reorganization

## Progress

| Phase | Status | Commits |
|-------|--------|---------|
| 1. Extract internal/ast | Not started | |
| 2. Extract internal/categorize | Not started | |
| 3. Extract internal/emit | Not started | |
| 4. Extract internal/reassemble | Not started | |
| 5. Clean up root package | Not started | |
| 6. Split CLI into files | Not started | |
| 7. Clean up demos and docs | Not started | |

**Current**: Starting Phase 1

---

## Methodology

Each phase follows TDD:

1. **RED**: Write failing tests, commit: `test(scope): add tests for X`
2. **GREEN**: Implement to pass tests, commit: `feat(scope): implement X` or `refactor(scope): ...`
3. **REFACTOR**: Fix linter issues, commit: `refactor(scope): clean up X`
4. **REVIEW**: Stop for Joe's quick review

use conventional commits.
Trailer: `AI-Used: [claude]`

---

## Phase 1: Extract internal/ast utilities

**Goal**: Move AST helper functions to `internal/ast/util.go`

**Functions to extract from reorder.go**:

- `isExported(name string) bool`
- `extractTypeName(expr dst.Expr) string`
- `extractReceiverTypeName(recv *dst.FieldList) string`
- `containsIota(expr dst.Expr) bool`
- `isIotaBlock(decl *dst.GenDecl) bool`
- `extractEnumType(decl *dst.GenDecl) string`

**TDD Cycle**:

1. RED: Create `internal/ast/util_test.go` with tests for each function
2. GREEN: Create `internal/ast/util.go`, move functions, update imports in reorder.go
3. REFACTOR: Fix any linter issues
4. REVIEW

---

## Phase 2: Extract internal/categorize

**Goal**: Move categorization types and logic to `internal/categorize/`

**Types/functions to extract**:

- `categorizedDecls` struct
- `typeGroup` struct
- `enumGroup` struct
- `categorizeDeclarations(file *dst.File) *categorizedDecls`
- `sortCategorized(cat *categorizedDecls)`
- `collectUncategorized(cat *categorizedDecls, includedSections map[string]bool)`
- `identifySection(decl dst.Decl) string`

**TDD Cycle**:

1. RED: Create `internal/categorize/categorize_test.go` with unit tests
2. GREEN: Create package, move code, update imports
3. REFACTOR: Fix linter issues
4. REVIEW

---

## Phase 3: Extract internal/emit

**Goal**: Move emitters to `internal/emit/`

**To extract from emitter.go**:

- `sectionEmitter` type
- All `emit*` functions
- Emitter registry

**To extract from reorder.go**:

- `mergeConstSpecs`
- `mergeVarSpecs`

**TDD Cycle**:

1. RED: Create `internal/emit/emit_test.go` with comprehensive tests
2. GREEN: Move code, update imports
3. REFACTOR: Fix linter issues
4. REVIEW

---

## Phase 4: Extract internal/reassemble

**Goal**: Move reassembly logic to `internal/reassemble/`

**Functions to extract**:

- `reassembleDeclarations(cat *categorizedDecls) []dst.Decl`
- `reassembleDeclarationsWithConfig(cat *categorizedDecls, cfg *Config) []dst.Decl`

**TDD Cycle**:

1. RED: Create `internal/reassemble/reassemble_test.go`
2. GREEN: Move code, update imports
3. REFACTOR: Fix linter issues
4. REVIEW

---

## Phase 5: Clean up root package

**Goal**: Slim down reorder.go to public API only

**After phases 1-4, reorder.go should only contain**:

- `Source(src string) (string, error)`
- `SourceWithConfig(src string, cfg *Config) (string, error)`
- `File(file *dst.File) error`
- `FileWithConfig(file *dst.File, cfg *Config) error`
- `AnalyzeSectionOrder(src string) (*SectionOrder, error)`
- `Section` and `SectionOrder` types

**Also**:

- Delete `emitter.go` (moved to internal/emit)
- Merge `reorder_config_test.go` into `reorder_test.go`

**TDD Cycle**:

1. RED: N/A (no new tests, existing tests must pass)
2. GREEN: Clean up files, verify tests pass
3. REFACTOR: Fix linter issues
4. REVIEW

---

## Phase 6: Split CLI into files

**Goal**: Break up cmd/go-reorder/main.go

**New files**:

- `main.go` - Entry point only (~10 lines)
- `cli.go` - CLI struct, Run method, cliOptions
- `discover.go` - discoverFiles, isExcluded
- `process.go` - run, processFile, processStdin
- `testutil_test.go` - testContext, executeCLI (test-only)

**TDD Cycle**:

1. RED: N/A (no new tests, existing tests must pass)
2. GREEN: Split files, verify tests pass
3. REFACTOR: Fix linter issues
4. REVIEW

---

## Phase 7: Clean up demos and docs

**Goal**: Remove cruft, finalize

**Tasks**:

- Delete `cmd/demo/`
- Delete `cmd/reorder-demo/`
- Delete `PLAN-config-and-cli.md` (historical)
- Delete `PLAN-development.md` (historical)
- Update README if needed

**TDD Cycle**:

1. RED: N/A
2. GREEN: Delete files, verify build
3. REFACTOR: N/A
4. REVIEW (final)

---

## Checkpoints

| After Phase | What to Review                                 |
| ----------- | ---------------------------------------------- |
| 1           | internal/ast package works, reorder.go slimmer |
| 2           | Categorization isolated, types hidden          |
| 3           | Emitters isolated, better tested               |
| 4           | Reassembly isolated                            |
| 5           | reorder.go is ~100 lines of pure API           |
| 6           | CLI cleanly organized                          |
| 7           | No cruft, clean repo                           |

---

## Risk Mitigation

- **All existing tests must pass after each GREEN commit**
- If a phase breaks tests unexpectedly, revert and reassess
- Each phase is small enough to abandon if needed

---

## Estimated Commits

~21 commits total (3 per phase × 7 phases)

Ready to start Phase 1?
