# Eliminate Global Pattern State Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace mutable package-level `allCommandPatterns`/`allWrapperPatterns` vars with functions, and pass an evaluator function to validators so future recursive validators (find, xargs) don't need `init()`.

**Architecture:** Add an `evaluatorFunc` type, change `argsValidator` to accept it as a second parameter, convert the two global `var` slices into functions returning fresh slices. The evaluator closure is constructed in `matchAndBuild` from the already-in-scope pattern slices.

**Tech Stack:** Go, mvdan.cc/sh/v3 (bash AST parser)

**Spec:** `docs/superpowers/specs/2026-04-04-eliminate-global-patterns-design.md`

**Test command:** `cd hooks/bash-approve && go test ./...`
**Lint command:** `cd hooks/bash-approve && golangci-lint run ./...`

---

### Task 1: Add `evaluatorFunc` type and update `argsValidator` signature

**Files:**
- Modify: `hooks/bash-approve/rules.go:1-60` (types and `WithValidator`)
- Modify: `hooks/bash-approve/curl.go:127` (`isCurlReadOnly` signature)
- Modify: `hooks/bash-approve/main.go:483` (validator call site in `matchAndBuild`)

This task changes the type system. Tests will fail until all call sites are updated, so all three files change together.

- [ ] **Step 1: Add `evaluatorFunc` type and update `argsValidator` in `rules.go`**

In `rules.go`, replace lines 9-16 (the comment block and `argsValidator` type):

```go
// pattern defines a command or wrapper that can be matched.
// tags controls both the label (first tag) and config matching (all tags).
// decision is the hook permission decision: "allow" (default) or "" (no opinion, ask user).
// denyReason, if set, is shown to Claude when the command is denied, explaining why.
// argsValidator is called after a regex match to refine the decision using
// the parsed AST arguments. Return true to keep the matched decision, false
// to downgrade to "ask" (no opinion).
type argsValidator func(args []*syntax.Word) bool
```

with:

```go
// pattern defines a command or wrapper that can be matched.
// tags controls both the label (first tag) and config matching (all tags).
// decision is the hook permission decision: "allow" (default) or "" (no opinion, ask user).
// denyReason, if set, is shown to Claude when the command is denied, explaining why.

// evaluatorFunc evaluates a command string against pattern lists.
// Used to break circular dependencies between validators and pattern definitions.
type evaluatorFunc func(cmd string) *result

// argsValidator is called after a regex match to refine the decision using
// the parsed AST arguments (including command name at [0]).
// Return true to keep the matched decision, false to downgrade to "ask".
type argsValidator func(args []*syntax.Word, eval evaluatorFunc) bool
```

- [ ] **Step 2: Update `isCurlReadOnly` signature in `curl.go`**

Change line 127 from:

```go
func isCurlReadOnly(args []*syntax.Word) bool {
```

to:

```go
func isCurlReadOnly(args []*syntax.Word, _ evaluatorFunc) bool {
```

- [ ] **Step 3: Update validator call site in `matchAndBuild` in `main.go`**

Change lines 482-484 from:

```go
	// Run post-match validator if present; false downgrades to "ask".
	if matched.validate != nil && !matched.validate(astArgs) {
		return &result{reason: matched.label(), decision: decisionAsk}
	}
```

to:

```go
	// Run post-match validator if present; false downgrades to "ask".
	if matched.validate != nil {
		eval := func(cmd string) *result {
			return evaluate(cmd, wrapperPats, commandPats)
		}
		if !matched.validate(astArgs, eval) {
			return &result{reason: matched.label(), decision: decisionAsk}
		}
	}
```

- [ ] **Step 4: Run tests to verify nothing broke**

Run: `cd hooks/bash-approve && go test ./...`
Expected: All tests PASS (the type change is internal; behavior is identical).

- [ ] **Step 5: Run linter**

Run: `cd hooks/bash-approve && golangci-lint run ./...`
Expected: No warnings.

- [ ] **Step 6: Commit**

```bash
cd hooks/bash-approve
git add rules.go curl.go main.go
git commit -m "refactor: add evaluatorFunc type and pass to validators"
```

---

### Task 2: Convert global vars to functions

**Files:**
- Modify: `hooks/bash-approve/rules.go` (convert `var allWrapperPatterns` and `var allCommandPatterns` to functions)
- Modify: `hooks/bash-approve/main.go` (`buildActivePatterns` — 4 references to old globals)
- Modify: `hooks/bash-approve/main_test.go` (6 references to old globals)

- [ ] **Step 1: Convert `allWrapperPatterns` to `wrapperPatterns()` in `rules.go`**

Find `var allWrapperPatterns = []pattern{` and change it to:

```go
func wrapperPatterns() []pattern {
	return []pattern{
```

Add a closing `}` after the existing `}` that closes the slice literal. The result:

```go
func wrapperPatterns() []pattern {
	return []pattern{
		NewPattern(`^timeout\s+\d+\s+`, tags("timeout", "wrapper")),
		// ... existing patterns unchanged ...
		NewPattern(`^/[^\s]+/`, tags("absolute path", "wrapper")),
	}
}
```

- [ ] **Step 2: Convert `allCommandPatterns` to `commandPatterns()` in `rules.go`**

Same treatment. Find `var allCommandPatterns = []pattern{` and change it to:

```go
func commandPatterns() []pattern {
	return []pattern{
```

Add a closing `}` after the existing `}` that closes the slice literal. The result:

```go
func commandPatterns() []pattern {
	return []pattern{
		// git
		NewPattern(`^git\s+...`, tags("git read op", "git")),
		// ... existing patterns unchanged ...
		NewPattern(`^standardrb\b`, tags("standardrb", "ruby")),
	}
}
```

- [ ] **Step 3: Update `buildActivePatterns` in `main.go`**

Replace the entire `buildActivePatterns` function body (4 references to the old globals):

```go
// buildActivePatterns filters pattern lists based on the config.
func buildActivePatterns(cfg Config) (wrappers []pattern, commands []pattern) {
	// Fast path: if all enabled and nothing disabled, return all patterns.
	if len(cfg.Disabled) == 0 && len(cfg.Enabled) == 1 && cfg.Enabled[0] == "all" {
		return wrapperPatterns(), commandPatterns()
	}

	enabled := toSet(cfg.Enabled)
	disabled := toSet(cfg.Disabled)
	for _, p := range wrapperPatterns() {
		if isEnabled(p.tags, enabled, disabled) {
			wrappers = append(wrappers, p)
		}
	}
	for _, p := range commandPatterns() {
		if isEnabled(p.tags, enabled, disabled) {
			commands = append(commands, p)
		}
	}
	return
}
```

- [ ] **Step 4: Update all 6 test references in `main_test.go`**

Line 16 — `evaluateAll`:
```go
func evaluateAll(cmd string) *result {
	return evaluate(cmd, wrapperPatterns(), commandPatterns())
}
```

Line 607 — `stripWrappers` test:
```go
			core, wrappers := stripWrappers(tt.cmd, wrapperPatterns())
```

Lines 764-765 — `TestBuildActivePatterns`:
```go
		assert.Len(t, wrappers, len(wrapperPatterns()))
		assert.Len(t, commands, len(commandPatterns()))
```

Line 1190 — `TestPatternTagsNonEmpty`:
```go
	allPatterns := append(wrapperPatterns(), commandPatterns()...)
```

Line 1217 — `TestNoOverlappingPatterns`:
```go
		for _, sc := range commandPatterns() {
```

- [ ] **Step 5: Run tests**

Run: `cd hooks/bash-approve && go test ./...`
Expected: All tests PASS.

- [ ] **Step 6: Run linter**

Run: `cd hooks/bash-approve && golangci-lint run ./...`
Expected: No warnings.

- [ ] **Step 7: Commit**

```bash
cd hooks/bash-approve
git add rules.go main.go main_test.go
git commit -m "refactor: replace global pattern vars with functions

Eliminates mutable package-level state. Pattern slices are now
constructed by wrapperPatterns() and commandPatterns(), returning
fresh slices on each call. This enables future recursive validators
(find -exec, xargs) to use WithValidator() at definition time
instead of init()."
```
