# Feature Specification: Fix //go:embed Directive Preservation

**Feature Branch**: `001-fix-embed-directives`
**Created**: 2026-02-18
**Status**: Draft
**Input**: User description: "Fix go-reorder stripping //go:embed directives when reordering declarations (GitHub Issue #5)"

## Clarifications

### Session 2026-02-18

- Q: Should FR-005 (grouped var blocks) be removed or narrowed? → A: Keep FR-005 — extract directive-bearing vars from grouped blocks and emit them standalone.
- Q: What should the tool do when a blank line separates a directive from its declaration? → A: Normalize — remove the blank line and attach the directive to the declaration (auto-fix invalid input).
- Q: Should `//go:build` constraints be explicitly excluded from FR-002's scope? → A: Scope FR-002 to "declaration-bound directives" — file-level directives like `//go:build` are naturally excluded.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Embed Directives Stay Attached to Vars (Priority: P1)

A developer has a Go file containing `//go:embed` directives annotating `var` declarations. When they run `go-reorder` to sort their file's declarations, the `//go:embed` directives remain immediately above their associated `var` declarations. The reordered file compiles and the embedded variables contain the expected file contents at runtime.

**Why this priority**: This is the core bug reported in Issue #5. Without this fix, `go-reorder` silently breaks runtime behavior — embedded vars become nil, which compiles fine but fails at runtime. This is the most dangerous class of bug.

**Independent Test**: Can be fully tested by creating a Go file with an `//go:embed` directive on a var, reordering the file, and verifying the directive remains directly above its var in the output.

**Acceptance Scenarios**:

1. **Given** a Go file with `//go:embed testdata/hello.txt` immediately above `var helloTxt []byte`, **When** `go-reorder` processes the file and moves the var to a new position, **Then** the `//go:embed` directive moves with the var and appears immediately above it in the output.
2. **Given** a Go file with multiple `//go:embed` directives on different vars, **When** `go-reorder` processes the file, **Then** each directive remains attached to its respective var regardless of how vars are reordered.
3. **Given** a Go file with a `//go:embed` directive followed by a regular doc comment followed by a var, **When** `go-reorder` processes the file, **Then** the directive, doc comment, and var all move together as a unit.

---

### User Story 2 - Other Compiler Directives Preserved (Priority: P2)

A developer has a Go file containing other Go compiler directives (e.g., `//go:generate`, `//go:noinline`, `//go:nosplit`) attached to declarations. When they run `go-reorder`, these directives remain attached to their associated declarations.

**Why this priority**: Same root cause as the embed bug. Fixing the general directive-attachment problem prevents a class of issues, not just one symptom.

**Independent Test**: Can be tested by creating a Go file with `//go:generate` or `//go:noinline` directives on declarations, reordering, and verifying directives stay attached.

**Acceptance Scenarios**:

1. **Given** a Go file with `//go:generate` above a function declaration, **When** `go-reorder` processes the file, **Then** the directive remains immediately above its function.
2. **Given** a Go file with `//go:noinline` above a function, **When** `go-reorder` processes the file, **Then** the directive stays attached to the function.

---

### Edge Cases

- When a var has multiple `//go:embed` directives (embedding multiple patterns into one var), all directives move with the var as a unit (per FR-004).
- When a `//go:embed` directive is on an unexported var, it is preserved identically to exported vars — the directive moves with the var during reordering.
- When a blank line exists between the directive and the var in the input (invalid per Go spec), the tool normalizes by removing the blank line and attaching the directive directly to the declaration in the output.
- When a `//go:embed` var is inside a grouped `var ( ... )` block (technically invalid Go, but users may not realize this), the tool extracts it as a standalone `var` declaration with its directive preserved, producing valid output. Remaining non-directive vars stay in the grouped block.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The tool MUST treat `//go:embed` directives as semantically bound to the immediately following `var` declaration, moving them together as a unit during reordering.
- **FR-002**: The tool MUST preserve all declaration-bound Go compiler directives (comments matching `//go:` prefix that are attached to a specific var or func declaration) during reordering. File-level directives like `//go:build` are not in scope. Type declarations already preserve GenDecl decorations via existing code. Const-bound directives are deferred (no practical use case identified).
- **FR-003**: The tool MUST preserve the relative ordering between a directive and its declaration (directive immediately above, no inserted blank lines).
- **FR-004**: The tool MUST handle vars with multiple `//go:embed` directives, keeping all of them attached.
- **FR-005**: The tool MUST handle `//go:embed` vars that appear inside grouped `var ( ... )` blocks (technically invalid Go) by extracting them as standalone declarations with their directives preserved, producing valid output that helps users correct the mistake.
- **FR-006**: The tool MUST normalize blank lines between a `//go:` directive and its declaration, removing the gap so the directive appears immediately above the declaration in the output.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All existing tests continue to pass after the fix (no regressions).
- **SC-002**: A Go file with `//go:embed` directives produces correct output after reordering — directives remain directly above their vars.
- **SC-003**: A Go file with other `//go:` compiler directives (generate, noinline, nosplit) produces correct output after reordering.
- **SC-004**: The fix handles the exact reproduction case from Issue #5 correctly.
