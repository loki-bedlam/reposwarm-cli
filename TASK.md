# CLI Refactor Task

## Overview
Refactor the RepoSwarm CLI to be agent-first by default, with rich human output as opt-in.
Also restructure commands for better grouping. Do NOT add any new features or options — only change existing stuff.

Read CLI-REFACTOR-PLAN.md in the workspace root (infra/reposwarm/) for full context, but follow THIS file for what to actually do.

## Rules
- Run `go vet ./...` and `go test ./...` before finishing — everything must pass
- Do NOT add new commands or new flags (except `--human`)
- Do NOT rename `new` (keep it as `new`)
- Do NOT merge `doctor` into `status` (keep `doctor` as its own command)
- Keep backward-compat aliases for moved commands (print deprecation warning to stderr)
- Do NOT change any API calls or business logic — only output formatting and command tree structure

## Part 1: Output Architecture

### Goal
- **Default output** (no flags): plain text, no emojis, no color, no box-drawing. Markdown-compatible. Parseable by grep/awk/LLM.
- **`--human` flag**: moves current rich output here (emojis, colors, progress bars, fancy tables)
- **`--json` flag**: unchanged

### How
1. Create a `Formatter` interface in `internal/output/`:
```go
type Formatter interface {
    Table(headers []string, rows [][]string)
    Success(msg string)
    Error(msg string)
    Info(msg string)
    Section(title string)
    KeyValue(key, value string)
    List(items []string)
    Progress(completed, total int)  // only for human mode
}
```

2. Implement `AgentFormatter` — plain text, e.g.:
   - `Table`: aligned columns with `-`/`+` separators
   - `Success`: `OK: message`
   - `Error`: `ERROR: message`
   - `Info`: just the message
   - `Section`: `## Title` (markdown heading)
   - `KeyValue`: `Key:  value` (aligned)
   - `Progress`: `Progress: 6/36 (16%)`

3. Implement `HumanFormatter` — current style with emojis, colors, progress bars

4. Add `--human` global flag (alongside existing `--json`). Remove `--no-color`.

5. Set global `output.F` based on flags:
   - `--json` → JSON mode (each command handles this itself, unchanged)
   - `--human` → HumanFormatter
   - default → AgentFormatter

6. Migrate ALL commands to use `output.F.Table(...)`, `output.F.Success(...)` etc instead of direct `fmt.Printf` with emojis/colors.

### Agent output examples
```
# reposwarm repos list
Repositories (36 total)

Name                       Source       Enabled  Status
--------------------------+-----------+---------+--------
agentcore-agent            CodeCommit   yes      ok
agentcore-chat-agent       CodeCommit   yes      ok

# reposwarm wf progress
Daily Investigation Progress

Workflow:  investigate-daily-1772344932677
Started:   2026-03-01T06:02:12Z
Elapsed:   82m57s
Progress:  6/36 (16%)

Completed  6
Running    3
Failed     0
Pending    27

Completed:
  agentcore-agent            24m46s
  agentcore-chat-ui          24m32s

In Progress:
  bedlam-infra               24m59s

# reposwarm status
API:       ok (https://dkhtk1q9b2nii.cloudfront.net/v1)
Temporal:  connected (default)
DynamoDB:  connected
Worker:    1 active

# reposwarm repos add my-repo --url https://...
OK: Added repository my-repo

# reposwarm repos remove my-repo -y
OK: Removed repository my-repo
```

## Part 2: Command Restructuring

Move these commands under their logical parent. Keep the old top-level command as a deprecated alias (prints warning to stderr then runs the real command).

| Old Location | New Location | Notes |
|---|---|---|
| `reposwarm discover` | `reposwarm repos discover` | Keep alias |
| `reposwarm watch` | `reposwarm workflows watch` | Keep alias |
| `reposwarm diff` | `reposwarm results diff` | Keep alias |
| `reposwarm report` | `reposwarm results report` | Keep alias |
| `reposwarm server-config show` | `reposwarm config server` | Keep alias |
| `reposwarm server-config set` | `reposwarm config server-set` | Keep alias |
| `reposwarm results show` | `reposwarm results sections` | Keep `show` as alias |

### Deprecation alias pattern
```go
func deprecatedAlias(newCmd string, realCmd *cobra.Command) *cobra.Command {
    return &cobra.Command{
        Use:    realCmd.Use,
        Hidden: true,  // hide from help
        RunE: func(cmd *cobra.Command, args []string) error {
            fmt.Fprintf(os.Stderr, "Warning: '%s' is deprecated, use '%s' instead\n", cmd.CommandPath(), newCmd)
            return realCmd.RunE(cmd, args)
        },
    }
}
```

## Testing
After all changes:
1. `go vet ./...` must pass
2. `go test ./...` must pass
3. Verify these work:
   - `reposwarm repos list` (agent output)
   - `reposwarm repos list --human` (rich output)
   - `reposwarm repos list --json` (JSON)
   - `reposwarm discover` (deprecated alias, should warn + work)
   - `reposwarm repos discover` (new location)
   - `reposwarm wf progress` (agent output)
   - `reposwarm wf progress --human` (rich output)
