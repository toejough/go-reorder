# Patterns and Lessons Learned

## CLI Test Infrastructure

**Lesson:** When adding new code paths in a CLI with test infrastructure, follow existing patterns for exit handling.

**Context:** Added `runInit()` that called `os.Exit(1)` directly, which broke tests because the test harness uses a `testCtx` to capture exit codes.

**Pattern:** Helper functions should return exit codes, not call `os.Exit()`. Let the caller (e.g., `Run()`) check for test context and handle exit appropriately:
```go
// Good: return exit code
func (c *CLI) runInit(...) int {
    if err != nil {
        return 1
    }
    return 0
}

// Caller handles test context
exitCode := c.runInit(...)
if testCtx != nil {
    testCtx.exitCode = exitCode
    return nil
}
if exitCode != 0 {
    os.Exit(exitCode)
}
```

**General principle:** When adding new functionality, look at how similar existing functionality handles edge cases (testing, error handling, cleanup).

---

## Documentation Consolidation

**Lesson:** Before creating a new documentation file, check if the content belongs in an existing file.

**Context:** Created CLAUDE.md for "AI discoverability" but it was largely redundant with README.md. Ended up merging and deleting it.

**Pattern:**
- Don't assume different audiences need separate docs - good documentation works for everyone
- A single comprehensive README is usually better than fragmented docs
- If you find yourself duplicating content across files, consolidate

**Exception:** Separate files make sense when content is truly distinct (e.g., CONTRIBUTING.md for contributor-specific workflows, CHANGELOG.md for version history).
