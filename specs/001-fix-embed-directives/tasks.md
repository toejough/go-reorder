# Tasks: Fix //go:embed Directive Preservation

**Input**: Design documents from `/specs/001-fix-embed-directives/`
**Prerequisites**: plan.md, spec.md, research.md

**Tests**: Required — TDD red/green/refactor cycle per project conventions.

**Organization**: Tasks grouped by user story. US1 is the MVP.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story (US1, US2)
- Exact file paths included in descriptions

---

## Phase 1: Foundational (Blocking Prerequisites)

**Purpose**: Utility functions and data structures needed by all user stories

- [x] T001 [P] Add `HasGoDirective` utility function that checks for `//go:` prefix in GenDecl decorations in `internal/ast/util.go`
- [x] T002 [P] Add directive-bearing GenDecl fields (`ExportedVarDecls`, `UnexportedVarDecls`) to `CategorizedDecls` struct in `internal/categorize/categorize.go`

**Checkpoint**: Foundation ready — user story implementation can begin

---

## Phase 2: User Story 1 - Embed Directives Stay Attached to Vars (Priority: P1) MVP

**Goal**: `//go:embed` directives remain attached to their var declarations after reordering, including edge cases (grouped blocks, blank lines, multiple directives).

**Independent Test**: Create Go files with `//go:embed` directives on vars, reorder, verify directives appear directly above their vars in output.

**Covers**: FR-001, FR-003, FR-004, FR-005, FR-006, SC-002, SC-004

### Tests for User Story 1 (Red Phase)

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [x] T003 [US1] Write failing test for basic `//go:embed` directive on exported var — input has directive above var, other declarations force reorder, expect directive in output directly above var — in `tests/reorder_test.go`
- [x] T004 [US1] Write failing test for `//go:embed` directive on unexported var — same pattern with unexported var among other unexported declarations — in `tests/reorder_test.go`
- [x] T005 [US1] Write failing test for Issue #5 exact reproduction case (embed + functions that trigger reorder) — in `tests/reorder_test.go`
- [x] T006 [US1] Write failing test for multiple `//go:embed` directives on different vars — two vars with different embeds, both preserved — in `tests/reorder_test.go`
- [x] T007 [US1] Write failing test for `//go:embed` directive + doc comment on same var — directive, comment, and var all move together — in `tests/reorder_test.go`
- [x] T008 [US1] Write failing test for multiple `//go:embed` lines on one var (FR-004) — var with two embed patterns, both kept — in `tests/reorder_test.go`
- [x] T009 [US1] Write failing test for `//go:embed` var inside grouped `var()` block (FR-005) — directive-bearing var extracted as standalone, remaining vars stay grouped — in `tests/reorder_test.go`
- [x] T010 [US1] Write failing test for blank line between `//go:embed` and var (FR-006) — blank line removed in output, directive attached directly — in `tests/reorder_test.go`

### Implementation for User Story 1 (Green Phase)

- [x] T011 [US1] Implement directive detection in var extraction path — when GenDecl has `//go:` directive, store whole GenDecl in `ExportedVarDecls`/`UnexportedVarDecls` instead of extracting ValueSpec — in `internal/categorize/categorize.go`
- [x] T012 [US1] Implement grouped var block extraction (FR-005) — when a `var()` block contains a directive-bearing spec, extract it as standalone GenDecl with directive preserved, keep remaining specs in grouped block — in `internal/categorize/categorize.go`
- [x] T013 [US1] Implement blank line normalization (FR-006) — ensure directive-bearing GenDecls have no `EmptyLine` spacing between directive and declaration keyword — in `internal/categorize/categorize.go`
- [x] T014 [US1] Implement individual emit for directive-bearing vars — emit `*VarDecls` as standalone declarations before the merged block in same section — in `internal/emit/emit.go`
- [x] T015 [US1] Wire directive-bearing decl fields through reassembly — ensure `Declarations()` and `DeclarationsWithOrder()` include new fields in correct section positions — in `internal/reassemble/reassemble.go`

**Checkpoint**: US1 complete — all `//go:embed` tests pass, existing tests still pass (SC-001)

---

## Phase 3: User Story 2 - Other Compiler Directives Preserved (Priority: P2)

**Goal**: `//go:generate`, `//go:noinline`, `//go:nosplit` and other declaration-bound `//go:` directives stay attached to their declarations after reordering.

**Independent Test**: Create Go files with `//go:noinline` or `//go:generate` on functions, reorder, verify directives remain attached.

**Covers**: FR-002, SC-003

### Tests for User Story 2 (Red Phase)

- [x] T016 [US2] Write failing test for `//go:noinline` on function — directive stays above function after reorder — in `tests/reorder_test.go`
- [x] T017 [US2] Write failing test for `//go:generate` on function — directive stays above function after reorder — in `tests/reorder_test.go`
- [x] T018 [US2] Write verification test for directive on type declaration — confirm type directives already preserved by existing GenDecl.Decs handling (expect pass) — in `tests/reorder_test.go`

### Implementation for User Story 2 (Green Phase)

- [x] T019 [US2] Verify function directive preservation — `FuncDecl` nodes already store whole declarations with decorations; confirm tests pass without changes or implement fix if needed — in `internal/categorize/categorize.go`

**Checkpoint**: US2 complete — all directive tests pass, all existing tests still pass

---

## Phase 4: Polish & Cross-Cutting Concerns

**Purpose**: Refactoring and final validation

- [x] T020 Evaluate refactoring opportunities — assessed: duality justified (different types serve different purposes), no simplification needed
- [x] T021 Run full regression test suite and verify all success criteria (SC-001 through SC-004) — all pass
- [x] T022 Update `FindExcludedSections` in `internal/categorize/categorize.go` to account for new directive-bearing decl fields if needed for strict/warn/drop modes — done by categorize-impl
- [ ] T023 Close GitHub Issue #5 with commit referencing `Closes #5`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Foundational (Phase 1)**: No dependencies — start immediately
- **US1 (Phase 2)**: Depends on Phase 1 completion — this is the MVP
- **US2 (Phase 3)**: Depends on Phase 1; can start in parallel with US1 since it targets different declaration types (functions vs vars)
- **Polish (Phase 4)**: Depends on Phase 2 and Phase 3 completion

### Within Each User Story

- Tests MUST be written and FAIL before implementation (TDD Red)
- Implementation makes tests pass (TDD Green)
- Refactoring evaluated in Polish phase (TDD Refactor)

### Parallel Opportunities

- **Phase 1**: T001 and T002 can run in parallel (different files)
- **Phase 2 vs Phase 3**: US2 tests (T016-T018) can be written in parallel with US1 implementation — they target functions/types, not vars
- **Phase 2 tests**: T003-T010 are all in same file — sequential within phase, but can be written as a batch

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Foundational (T001-T002)
2. Complete Phase 2: US1 Red tests (T003-T010)
3. Complete Phase 2: US1 Green implementation (T011-T015)
4. **STOP and VALIDATE**: All embed tests pass, existing tests still pass
5. This alone closes Issue #5

### Incremental Delivery

1. Foundational → ready
2. US1 (embed on vars) → Test independently → closes Issue #5 (MVP!)
3. US2 (other directives on funcs) → Test independently → hardens the fix
4. Polish → Refactor + regression validation

---

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks
- [Story] label maps task to specific user story for traceability
- All tests are table-driven in `tests/reorder_test.go` following existing patterns
- Commit after each logical group (red tests, green implementation)
- US2 may be a no-op verification if function directives are already preserved by DST
