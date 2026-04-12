# Safe Repo Subdir `cd` Design

**Goal:** Allow `cd` into an existing directory inside the current repository without triggering OpenCode permission prompts for otherwise-approved command chains.

**Context:** `make`, `go`, and `bun` commands are already approved by `bash-approve`, but command chains like `go build ... && cd frontend && bun run test:e2e ...` fall through to OpenCode because the `cd` validator only approves repo roots and sibling worktree roots.

## Desired Behavior

- `cd frontend` should be auto-approved when `frontend/` exists inside the current repo.
- Existing approval for switching to a sibling worktree root that shares the same git common dir should remain intact.
- `cd` to paths outside the current repo should still fall through as `no-opinion`.
- `cd` to nonexistent targets should still fall through as `no-opinion`.

## Approach

- Keep the current worktree-root check.
- Add a repo-local directory check for `cd` targets:
  - resolve the target from `cwd`
  - require the target to exist and be a directory
  - require the resolved target to be contained within the current repo root
- Reuse the existing `WithValidatorFallback("")` behavior so invalid `cd` targets continue to defer to OpenCode rather than being force-allowed.

## Testing

- Add a failing test for `cd frontend && bun run test:e2e`.
- Add a failing test for `go build ... && cd frontend && bun run test:e2e`.
- Verify existing worktree-root `cd` behavior still passes.
- Verify `cd ..` or another out-of-repo target still does not auto-approve.
