# Research: Fix //go:embed Directive Preservation

**Date**: 2026-02-18 | **Branch**: `001-fix-embed-directives`

## Root Cause Analysis

### Bug Mechanism (3-stage loss)

**Stage 1 — Extraction** (`categorize.go:170-183`): When categorizing var declarations, individual `ValueSpec` nodes are extracted from their wrapping `GenDecl`. The `GenDecl.Decs` (which contains `//go:embed` directives in `.Decs.Start`) is discarded — only the bare `ValueSpec` is stored.

**Stage 2 — Storage**: `CategorizedDecls` stores vars as `[]*dst.ValueSpec` with no reference to original `GenDecl` decorations. The directive information is permanently lost at this point.

**Stage 3 — Merging** (`categorize.go:720-737`): `MergeVarSpecs` creates a new `GenDecl` with `decl.Decs.Start = nil` and overwrites every spec's `Decs.Before = dst.NewLine`. Even if directives survived extraction, they'd be overwritten here.

### Why Types Don't Have This Bug

Type declarations (`categorize.go:198-210`) already have a fix: for standalone GenDecls (`len(genDecl.Specs) == 1`), the entire `GenDecl.Decs` is preserved on a new GenDecl wrapper. This was added in commit `3c8f179` to fix doc comment loss. Vars never received this treatment.

### DST Decoration Model

The `dave/dst` library stores `//go:embed` directives in `GenDecl.Decs.Start` — the comment decorations before the `var` keyword. The individual `ValueSpec.Decs` only contains per-spec decorations (within a grouped `var()` block). For standalone var declarations, all leading comments (including directives) are on the `GenDecl`.

## Design Decision

**Decision**: Store directive-bearing vars/consts as whole `GenDecl` nodes in a separate field, emit them individually (not merged into blocks).

**Rationale**: Follows the principle of least disruption — existing merge logic stays unchanged for non-directive vars. Directive vars get the same treatment types already have (preserved GenDecl with decorations).

**Alternatives considered**:

1. **Transfer decorations to ValueSpec.Decs.Before during extraction** — Fragile: requires modifying MergeVarSpecs to detect and skip directive-bearing specs, changing its return type from `*dst.GenDecl` to `[]dst.Decl`.

2. **Change all var storage to `[]*dst.GenDecl`** — Correct but invasive refactor of CategorizedDecls, MergeVarSpecs, MergeConstSpecs, all emit functions, and all tests referencing these fields.

3. **Separate field for directive-bearing GenDecls** (chosen) — Minimal changes: new field in CategorizedDecls, detection during extraction, individual emit in correct section position. No changes to existing merge path.

## Go Spec Constraints

- `//go:embed` must immediately precede a standalone `var` declaration (not inside `var()` group)
- No blank lines allowed between directive and declaration
- Multiple `//go:embed` lines can precede one var (multiple patterns)
- Other `//go:` directives (`noinline`, `nosplit`, `generate`) follow similar attachment rules to their declarations

## Scope of Detection

Any comment matching `//go:` prefix in `GenDecl.Decs.Start` should trigger preservation. This covers `//go:embed`, `//go:generate`, `//go:noinline`, `//go:nosplit`, `//go:linkname`, etc. File-level directives like `//go:build` are not affected — they attach to the package clause, not to individual declarations.
