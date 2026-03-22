---
name: bash-approve-telemetry
description: Use when analyzing bash-approve hook decisions, finding commands that needed user approval, identifying candidates for auto-approval, or querying the telemetry database
---

# Bash-Approve Telemetry

Query the bash-approve hook's decision log to find approval candidates and analyze command patterns.

## Database Location

```
~/.claude/hooks/bash-approve/telemetry.db
```

The database is a SQLite file created next to the compiled `approve-bash` binary. The path is resolved via `os.Executable()` in `telemetry.go`.

## Schema

```sql
CREATE TABLE decisions (
    id       INTEGER PRIMARY KEY,
    ts       TEXT DEFAULT (datetime('now')),  -- UTC
    payload  TEXT,    -- full JSON input from Claude Code hook
    command  TEXT,    -- the bash command string
    decision TEXT,    -- allow | deny | ask | no-opinion
    reason   TEXT     -- human-readable label(s), pipe-separated
);
```

### Decision Values

| Decision | Meaning |
|----------|---------|
| `allow` | Auto-approved by a matching rule |
| `ask` | Fell through to Claude Code's permission prompt (intentional gate or no matching rule) |
| `no-opinion` | No matching pattern; hook abstained |
| `deny` | Explicitly blocked (e.g. `go mod vendor`) |

## Quick Reference Queries

All queries use `sqlite3 ~/.claude/hooks/bash-approve/telemetry.db`.

### Summary of recent decisions (past 7 days)

```sql
SELECT decision, count(*) FROM decisions
WHERE ts >= datetime('now', '-7 days')
GROUP BY decision ORDER BY count(*) DESC;
```

### Commands that triggered "ask" grouped by reason

```sql
SELECT reason, count(*) as cnt FROM decisions
WHERE ts >= datetime('now', '-7 days') AND decision = 'ask'
GROUP BY reason ORDER BY cnt DESC;
```

### Commands with "no-opinion" — candidates for new rules

```sql
SELECT command, count(*) as cnt FROM decisions
WHERE ts >= datetime('now', '-7 days') AND decision = 'no-opinion'
GROUP BY command ORDER BY cnt DESC LIMIT 30;
```

### Distinct unrecognized commands (no-opinion) with examples

```sql
SELECT command, reason, ts FROM decisions
WHERE ts >= datetime('now', '-7 days') AND decision = 'no-opinion'
ORDER BY ts DESC LIMIT 50;
```

### Most frequent commands overall

```sql
SELECT command, decision, count(*) as cnt FROM decisions
WHERE ts >= datetime('now', '-7 days')
GROUP BY command, decision ORDER BY cnt DESC LIMIT 30;
```

### Time range check

```sql
SELECT min(ts), max(ts), count(*) FROM decisions;
```

## Finding Auto-Approval Candidates

The best candidates for new auto-approve rules are commands that:
1. Show `no-opinion` (hook has no matching pattern at all)
2. Appear frequently (high count = high annoyance)
3. Are clearly safe (read-only, local, idempotent)

**Workflow:**
1. Run the "no-opinion grouped by command" query above
2. Identify repetitive safe commands (e.g., a CLI tool used often)
3. Add a new pattern in the bash-approve source repo's `rules.go` under `allCommandPatterns`
4. Or, if an existing pattern is in `disabled` in `categories.yaml`, move it to `enabled`

**Source code** is at `~/code/agent-skills/hooks/bash-approve/` — edits go here, not in `~/.claude/hooks/bash-approve/` (that's the deployed copy with compiled binary).

**Three decision types for new patterns:**
- `allow` (default) — auto-approve silently
- `WithDecision("")` — ask (fall through to Claude Code's permission prompt, e.g. `git push`, `gh pr create`, `go mod init`)
- `WithDecision("deny")` — block the command (e.g. `go mod vendor`)

Deny propagates through chains: if any segment is deny, the whole chain is deny. Ask propagates similarly but deny takes precedence.

For commands that show `ask` — these are **intentionally gated**. Only promote them to auto-approve if you're certain you want unattended execution.

## Configuration File

`~/.claude/hooks/bash-approve/categories.yaml` controls which rule categories are active:

```yaml
enabled:
  - all           # enable everything not explicitly disabled
disabled:
  - git push      # keep requiring confirmation
```

Fine-grained category names match the first tag in each pattern's `tags()` call in `rules.go`.

## Common Mistakes

- **Timestamps are UTC** — adjust when comparing to local time.
- **`no-opinion` vs `ask`** — `no-opinion` means no rule matched at all; `ask` means a rule matched but its decision is "" (intentional gate). New rules fix `no-opinion`; changing `categories.yaml` or pattern decisions fixes `ask`.
- **Database path** — it lives next to the compiled binary, not the source. If you recompile to a different location, a new empty db is created.
