# Implementation Plan: Fix //go:embed Directive Preservation

**Branch**: `001-fix-embed-directives` | **Date**: 2026-02-18 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/001-fix-embed-directives/spec.md`

## Summary

Fix go-reorder stripping `//go:embed` and other `//go:` compiler directives when reordering declarations. The root cause is a 3-stage loss: var extraction discards GenDecl decorations, storage has no decoration metadata, and merging overwrites remaining decorations. The fix stores directive-bearing vars/consts as whole GenDecls (not extracted ValueSpecs) and emits them individually, following the pattern already used for type declarations.

## Technical Context

**Language/Version**: Go 1.21+
**Primary Dependencies**: `dave/dst` (Decorated Syntax Tree), `dave/dst/decorator`
**Storage**: N/A (source code transformation tool)
**Testing**: `go test -tags sqlite_fts5` via `targ` build system
**Target Platform**: Cross-platform CLI tool
**Project Type**: Single Go module library + CLI
**Performance Goals**: N/A (bug fix)
**Constraints**: Must not break existing reorder behavior for non-directive declarations
**Scale/Scope**: ~4 files modified, ~150 lines changed

## Constitution Check

*GATE: Constitution is template (not project-specific). No gates to evaluate.*

No violations. Proceeding.

## Project Structure

### Documentation (this feature)

```text
specs/001-fix-embed-directives/
├── plan.md              # This file
├── research.md          # Root cause analysis and design decision
├── spec.md              # Feature specification
└── checklists/
    └── requirements.md  # Spec quality checklist
```

### Source Code (repository root)

```text
reorder.go                          # Public API (no changes needed)
internal/
├── categorize/
│   └── categorize.go               # MODIFY: directive detection + separate storage
├── emit/
│   └── emit.go                     # MODIFY: emit directive-bearing decls individually
├── reassemble/
│   └── reassemble.go               # MODIFY: pass directive decls through
└── ast/
    └── util.go                     # MODIFY: add HasDirective helper
tests/
    └── reorder_test.go             # MODIFY: add directive preservation tests
```

**Structure Decision**: Existing single-project structure. Changes are localized to the categorize/emit pipeline with a new utility function.

## Implementation Approach

### Change 1: Add directive detection utility

**File**: `internal/ast/util.go`

Add a `HasGoDirective(decs dst.GenDeclDecorations) bool` function that checks if any decoration in `Decs.Start` matches the `//go:` prefix. This centralizes detection logic.

### Change 2: Separate directive-bearing vars/consts during categorization

**File**: `internal/categorize/categorize.go`

Add new fields to `CategorizedDecls`:
```go
ExportedVarDecls     []*dst.GenDecl  // vars with //go: directives
UnexportedVarDecls   []*dst.GenDecl
```

In the var extraction path (lines 170-183), before extracting ValueSpecs:
1. Check if the GenDecl has `//go:` directives via `HasGoDirective`
2. If yes: store the whole GenDecl in the new `*VarDecls` field (preserving decorations)
3. If no: extract ValueSpecs as before (existing behavior unchanged)

Const-bound directives are deferred (no practical use case identified). Type declarations already preserve GenDecl decorations via existing code (commit 3c8f179).

### Change 3: Emit directive-bearing decls individually

**File**: `internal/emit/emit.go`

In `emitExportedVars` / `emitUnexportedVars`: emit directive-bearing GenDecls as individual declarations BEFORE the merged block. They sort into the same section but are not merged.

### Change 4: Pass new fields through reassembly

**File**: `internal/reassemble/reassemble.go`

Ensure `Declarations()` and `DeclarationsWithOrder()` include the new `*VarDecls` and `*ConstDecls` fields in the correct section positions. The emit functions already handle the details.

### Change 5: Extract directive-bearing vars from grouped blocks (FR-005)

**File**: `internal/categorize/categorize.go`

When a `var ( ... )` block contains a spec with a `//go:embed` directive (technically invalid Go, but users may not realize), extract that spec as a standalone `var` GenDecl with its directive preserved. Remaining non-directive specs stay in the grouped block. This produces valid Go output and helps users correct the mistake.

### Change 6: Normalize blank lines between directives and declarations (FR-006)

**File**: `internal/categorize/categorize.go`

When storing a directive-bearing GenDecl, ensure there is no `EmptyLine` spacing between the directive comment and the declaration keyword. If the original source had a blank line (invalid per Go spec), normalize it by setting appropriate `Decs.Before` spacing so the directive appears immediately above the declaration in output.

### Change 7: Handle function directives

**File**: `internal/categorize/categorize.go`

Verify that `FuncDecl` directives (`//go:noinline`, `//go:nosplit`) are already preserved. Since functions are stored as whole `*dst.FuncDecl` nodes (not extracted from GenDecls), their decorations should already be intact. Add test coverage to confirm.

## TDD Plan

### Red Phase (failing tests)

1. **Test: embed directive on exported var** — Input has `//go:embed` above `var`, expect directive present in output
2. **Test: embed directive on unexported var** — Same for unexported var
3. **Test: Issue #5 reproduction** — Exact case from the bug report
4. **Test: multiple embed directives on different vars** — Both preserved independently
5. **Test: embed directive + doc comment** — Both move together
6. **Test: multiple `//go:embed` lines on one var** — FR-004: var with two embed patterns, both kept
7. **Test: embed var inside grouped `var()` block** — FR-005: extracted as standalone
8. **Test: blank line between directive and var** — FR-006: normalized in output
9. **Test: `//go:noinline` on function** — Directive stays attached after reorder
10. **Test: `//go:generate` on function** — Same pattern
11. **Test: directive on type declaration** — Verify existing preservation (expect pass)

### Green Phase (make tests pass)

Implement Changes 1-6 above in order.

### Refactor Phase

Evaluate whether `ExportedVars`/`ExportedVarDecls` duality can be simplified. Consider if the merge functions should be aware of directives rather than having two parallel storage paths. Keep only if it simplifies the code without over-engineering.
