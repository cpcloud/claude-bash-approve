# Safe Repo Subdir `cd` Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Auto-approve `cd` into existing directories inside the current repo so otherwise-safe command chains do not fall through to OpenCode.

**Architecture:** Extend the existing `cd` validator instead of adding a new shell rule. Preserve the current same-repo boundary by approving either sibling worktree roots that share a git common dir or existing directories contained within the current repo root.

**Tech Stack:** Go, testify, existing `hooks/bash-approve` evaluator tests

---

### Task 1: Add failing coverage for repo-local `cd`

**Files:**
- Modify: `hooks/bash-approve/main_test.go`

- [ ] **Step 1: Write the failing test**

```go
t.Run("repo subdir cd is allowed in safe chain", func(t *testing.T) {
    repo := setupGitRepo(t)
    frontend := filepath.Join(repo, "frontend")
    require.NoError(t, os.Mkdir(frontend, 0o755))

    r := Evaluate(`go build ./... && cd frontend && bun run test:e2e`, Config{Enabled: []string{"all"}}, evalContext{cwd: repo})
    require.NotNil(t, r)
    assert.Equal(t, decisionAllow, r.decision)
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./hooks/bash-approve -run 'TestEvaluate.*repo subdir cd' -count=1`
Expected: FAIL because `cd frontend` falls back to no-opinion.

- [ ] **Step 3: Write minimal implementation**

```go
func isCurrentRepoWorktreeCD(args []*syntax.Word, ctx evalContext) bool {
    if repoLocalExistingDirectory(target, ctx.cwd) {
        return true
    }
    // existing worktree-root logic remains
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./hooks/bash-approve -run 'TestEvaluate.*repo subdir cd' -count=1`
Expected: PASS

### Task 2: Preserve current safety boundaries

**Files:**
- Modify: `hooks/bash-approve/main_test.go`
- Modify: `hooks/bash-approve/git.go`

- [ ] **Step 1: Add boundary tests**

```go
t.Run("cd outside repo remains no opinion", func(t *testing.T) {
    repo := setupGitRepo(t)
    outside := t.TempDir()

    r := Evaluate("cd "+outside, Config{Enabled: []string{"all"}}, evalContext{cwd: repo})
    require.NotNil(t, r)
    assert.Equal(t, "", r.decision)
})
```

- [ ] **Step 2: Implement minimal directory containment helper**

```go
func pathIsExistingDirWithinRepo(cwd, target string) bool
```

- [ ] **Step 3: Run targeted tests**

Run: `go test ./hooks/bash-approve -run 'TestEvaluate.*cd|TestEvaluateToolUse.*' -count=1`
Expected: PASS

- [ ] **Step 4: Run package tests**

Run: `go test ./hooks/bash-approve -count=1`
Expected: PASS
