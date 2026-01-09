# Issue Tracker

A simple md issue tracker.

## Statuses

- backlog (to choose from)
- selected (to work on next)
- in progress (currently being worked on)
- review (ready for review/testing)
- done (completed)
- cancelled (not going to be done, for whatever reason, should have a reason)
- blocked (waiting on something else)

---

## Issue Template

Standard issue structure organized by category:

```markdown
### [number]. [Issue Title]

#### Universal

**Status**
[backlog/selected/in progress/review/done/cancelled/blocked/migrated]

**Description** (optional - recommended)
[What needs to be done - clear, concise explanation of the issue or feature]

#### Planning (for backlog/selected issues)

**Rationale** (optional)
[Why this is needed - business/technical justification]

**Acceptance** (optional - recommended)
[What defines completion - specific, measurable criteria]

**Effort** (optional)
[Trivial/Small/Medium/Large with optional time estimate like "2-4 hours"]

**Priority** (optional)
[Low/Medium/High/Critical]

**Dependencies** (optional)
[What this issue depends on - other issues, external factors, etc.]

#### Work Tracking (for in-progress/done issues)

**Started**
[YYYY-MM-DD HH:MM TZ]

**Completed** (for done)
[YYYY-MM-DD HH:MM TZ or just YYYY-MM-DD]

**Commit** (for done)
[commit hash or multiple hashes for multi-commit work]

**Timeline**

- YYYY-MM-DD HH:MM TZ - [Phase/milestone]: [Activity description]
- YYYY-MM-DD HH:MM TZ - [Phase/milestone]: [Activity description]
- ...

#### Documentation (for done issues)

**Solution** (optional)
[How the issue was solved - implementation approach, key decisions]

**Files Modified** (optional)
[List of key files changed, especially useful for understanding impact]

#### Bug Details (for bug fixes)

**Discovered** (optional)
[When/how the bug was found - context about discovery]

**Root Cause** (optional)
[Technical explanation of what caused the issue]

**Current Behavior** (optional)
[What happens now - the problematic behavior]

**Expected Behavior** (optional)
[What should happen - the desired behavior]
```

---

## Backlog

Issues to choose from for future work.

### 7. Add --verbose flag to CLI

#### Universal

**Status**
backlog

**Description**
No way to see what config is being applied or get detailed output about processing.

#### Planning

**Rationale**
Debugging config issues is difficult without visibility into what config was loaded and applied.

**Acceptance**
- Add `--verbose` / `-v` flag
- When enabled, print: config file path (or "using defaults"), effective config values, files being processed
- Output goes to stderr to not interfere with stdout

**Effort**
Small

**Priority**
Low

---

## Selected

Issues selected for upcoming work.

_No selected issues_

---

## In Progress

Issues currently being worked on.

_No issues in progress_

---

## Review

Issues ready for review/testing.

_No issues in review_

---

## Done

Completed issues.

### 1. Fix deletion of receiver methods in method-only files

#### Universal

**Status**
done

**Description**
When `reorder.Source()` processes a Go file that contains only receiver methods (no type definitions, constants, or package-level functions), it deletes all the method implementations, leaving only the package declaration and imports.

#### Planning

**Rationale**
This prevents using go-reorder on codebases that organize receiver methods across multiple files, which is a common Go pattern for test helpers, interface implementations split by concern, and large types with methods organized by category.

**Acceptance**
`reorder.Source()` should reorder methods according to standard section ordering while preserving all method implementations, even in files that contain only receiver methods.

**Effort**
Medium

**Priority**
Critical - causes data loss

#### Work Tracking

**Started**
2026-01-01 23:30 EST

**Completed**
2026-01-02 09:59 EST

**Commit**
57a36e1

**Timeline**

- 2026-01-02 09:59 EST - Complete: Verified fix working in glowsync after magefile dependency update
- 2026-01-01 23:56 EST - Complete: Committed fix (57a36e1), all tests passing
- 2026-01-01 23:42 EST - COMMIT: Routing to git-workflow to commit fix
- 2026-01-01 23:42 EST - REFACTOR: Auditor PASS - 0 linter issues, all 16 tests passing
- 2026-01-01 23:40 EST - REFACTOR: Routing to auditor for code quality review
- 2026-01-01 23:40 EST - GREEN: Implementer added third pass for method-only typeGroups - all 5 tests passing
- 2026-01-01 23:37 EST - GREEN: Routing to implementer to fix categorizeDeclarations()
- 2026-01-01 23:37 EST - RED: failure-debugger created 5 regression tests, identified root cause in categorizeDeclarations()
- 2026-01-01 23:33 EST - INVESTIGATE: Routing to failure-debugger to reproduce and analyze bug
- 2026-01-01 23:30 EST - Started: Issue created from bug report

#### Bug Details

**Discovered**
While attempting to enable go-reorder in CI pipeline for copy-files repository. File `internal/syncengine/sync_test_helpers.go` had all methods deleted during reordering.

**Current Behavior**
`reorder.Source()` returns a string containing only package declaration and imports. All receiver method implementations are deleted.

**Expected Behavior**
`reorder.Source()` should preserve all receiver methods and reorder them according to standard section ordering.

**Root Cause**
In `categorizeDeclarations()` at lines 276-290, when a method declaration is encountered, a `typeGroup` is created/updated for the receiver type and the method is added to the typeGroup's method lists. However, if the type isn't declared in this file (typeDecl remains nil), the typeGroup is never added to `cat.exportedTypes` or `cat.unexportedTypes`, so `reassembleDeclarations()` never includes them in the output.

#### Documentation

**Solution**
Added a third pass in `categorizeDeclarations()` (lines 378-387) that identifies typeGroups with methods but no type declaration (typeDecl == nil) and adds them to the appropriate categorized list based on whether the type name is exported.

**Files Modified**
- reorder.go: Added third pass to categorize method-only typeGroups
- reorder_test.go: Added 5 comprehensive regression tests + real-world glowsync test case
- issues.md: Created issue tracker for the project

---

### 2. Add defensive check for untyped iota blocks

#### Universal

**Status**
done

**Description**
Untyped iota blocks were incorrectly categorized as unexported enums with empty type name instead of regular constants.

#### Planning

**Effort**
Small

**Priority**
High

#### Work Tracking

**Completed**
2026-01-08

**Commit**
cd232a0

#### Documentation

**Solution**
Modified `CategorizeDeclarations()` and `IdentifySection()` in `internal/categorize/categorize.go` to check if `typeName` is empty after calling `ExtractEnumType()`. If empty, treat as regular constants, not enum.

**Files Modified**
- internal/categorize/categorize.go: Check for empty typeName before treating as enum
- internal/categorize/categorize_test.go: Added test cases for untyped iota blocks

---

### 3. Document four-pass categorization algorithm

#### Universal

**Status**
done

**Description**
Added comprehensive documentation to `CategorizeDeclarations()` explaining the four-pass algorithm and constructor matching.

#### Work Tracking

**Completed**
2026-01-08

**Commit**
696cb5c

---

### 4. Add test for ambiguous constructor matching

#### Universal

**Status**
done

**Description**
Constructor matching uses longest-suffix match (`NewFooBar` matches `FooBar` over `Foo`), but this behavior was undocumented and untested.

#### Work Tracking

**Completed**
2026-01-08

**Commit**
9756e34

#### Documentation

**Solution**
Added `TestConstructorMatching_LongestMatchWins` test that verifies:
- `NewFoo` matches type `Foo` (not `FooBar`)
- `NewFooBar` matches type `FooBar` (longest match)
- `NewFooBarBaz` matches type `FooBar` (contains longest match)

**Files Modified**
- internal/categorize/categorize_test.go: Added dedicated constructor matching test

---

### 5. Include source position in parse error messages

#### Universal

**Status**
done

**Description**
Parse errors from `Source()` and `SourceWithConfig()` need to include line/column information.

#### Work Tracking

**Completed**
2026-01-08

**Commit**
07a2c87

#### Documentation

**Solution**
Investigated and found that parse errors already include line:column position from the Go parser. Error format is "failed to parse source: 3:6: expected 'IDENT', found 123". Added test to document this behavior.

**Files Modified**
- tests/reorder_test.go: Added TestSource_ParseErrorIncludesPosition

---

### 6. Add benchmarks for performance baseline

#### Universal

**Status**
done

**Description**
No benchmarks existed. Performance characteristics on large files were unknown.

#### Work Tracking

**Completed**
2026-01-08

**Commit**
9d5c3e2

#### Documentation

**Solution**
Added benchmarks for:
- `Source()` with small (~100 line), medium (~1k line), large (~10k line) files
- `SourceWithConfig()` with small/medium/large files
- `AnalyzeSectionOrder()`
- `CategorizeDeclarations()`

**Files Modified**
- tests/benchmark_test.go: New file with Source/SourceWithConfig/AnalyzeSectionOrder benchmarks
- internal/categorize/categorize_test.go: Added CategorizeDeclarations benchmark

---

## Cancelled

Issues that will not be completed.

_No cancelled issues_

---

## Blocked

Issues waiting on dependencies.

_No blocked issues_
