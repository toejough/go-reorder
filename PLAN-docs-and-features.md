# Documentation and Features Improvement Plan

Goal: Make go-reorder maximally discoverable and useful for humans and AIs.

## Status Legend
- [ ] Not started
- [x] Complete
- [~] In progress

---

## Phase 1: Integration (Critical - Enables Adoption)

### 1.1 Pre-commit Hook Support
- [x] Create `.pre-commit-hooks.yaml` in repo root
- [x] Add pre-commit setup instructions to README
- [ ] Test with actual pre-commit installation

### 1.2 GitHub Actions Example
- [x] Create `.github/workflows/example-reorder-check.yml` (as example, not active)
- [x] Document in README under "CI Integration"

### 1.3 Makefile/Mage Target Examples
- [x] Add example targets to README

---

## Phase 2: README Overhaul

### 2.1 Quick Start Section
- [x] Add 3-command quick start at top of README
- [ ] Include both CLI and library quick starts

### 2.2 Before/After Examples
- [x] Add visual before/after code transformation
- [x] Show real-world messy file -> organized file

### 2.3 What This Doesn't Do
- [x] Document limitations clearly
- [x] Import ordering (handled by goimports)
- [x] Build constraints
- [x] cgo //export comments

### 2.4 Configuration Recipes
- [x] Standard library/package recipe
- [x] Web application recipe
- [x] CLI tool recipe
- [x] Monorepo guidance

### 2.5 Integration Section
- [x] Pre-commit hook setup
- [x] GitHub Actions setup
- [x] VSCode tasks.json example
- [x] Makefile example

### 2.6 Troubleshooting Section
- [x] Common config mistakes
- [x] How to debug categorization issues
- [x] Mode selection guidance (strict vs warn vs append)

### 2.7 Config Discovery Documentation
- [x] Explain the walk-up algorithm
- [x] Document stop conditions (.git, go.mod)
- [x] Monorepo scenarios

---

## Phase 3: CLI Improvements

### 3.1 --init Flag
- [x] Add `--init` flag to scaffold `.go-reorder.toml`
- [x] Include helpful comments in generated config
- [x] Add tests

### 3.2 --list-sections Flag
- [x] Add flag to list available section names
- [x] Useful for config authoring

### 3.3 Improved Error Messages
- [x] Add hints to strict mode errors
- [x] Suggest how to fix common issues

---

## Phase 4: API Documentation

### 4.1 Godoc Improvements
- [x] Document Config struct fields with examples
- [x] Document behavior mode semantics
- [x] Document section matching algorithm
- [x] Document constructor pattern matching

### 4.2 Library Usage Examples
- [x] Add example in README for programmatic use
- [x] Show custom config creation
- [x] Show AST-level usage with File()

---

## Phase 5: Polish

### 5.1 AI/Human Discoverability
- [x] ~~Create CLAUDE.md~~ (merged into README instead)
- [x] Include common usage patterns in README
- [x] Document API for all users

### 5.2 Examples Directory
- [x] Create examples/ with sample configs
- [x] Add example transformations

---

## Progress Log

### 2026-01-11
- Phase 3: Added tests for --init and --list-sections CLI flags
- Phase 4: Improved godoc for Config, SectionsConfig, TypesConfig, BehaviorConfig
- Phase 4: Added package-level documentation with examples
- Phase 5: Created CLAUDE.md for AI discoverability
- Phase 5: Created examples/ directory with webapp, cli-tool, minimal configs
- Phase 2.5: Added VSCode tasks.json example to README
- Phase 3.3: Implemented strict mode errors with helpful hints
- Phase 4.2: Added AST-level File() usage documentation

### 2026-01-10
- Created this plan document
- Phase 1: Created .pre-commit-hooks.yaml, GitHub Actions example
- Phase 2: Complete README overhaul with quick start, before/after, recipes
- Phase 3: Added --init and --list-sections CLI flags
