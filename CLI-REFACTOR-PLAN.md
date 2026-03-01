# RepoSwarm CLI Refactor Plan

## 1. Output Architecture: Agent-First Design

### Current Problems
- Default output uses emojis, color codes, fancy tables, progress bars
- `--json` flag exists but is a second-class citizen bolted onto each command
- `--no-color` exists but output is still human-oriented (icons, decorative lines)
- Agents have to parse visual formatting or always remember `--json`

### New Design

**Default: Agent-friendly plain text (markdown-compatible)**
- No emojis, no color codes, no box-drawing characters
- Clean columns, dash separators, readable by any LLM or script
- Structured but plain — parseable by `grep`, `awk`, `cut`
- Markdown-compatible (headers, bullet lists, tables)

**`--human` flag: Rich terminal experience**
- Color, emojis, progress bars, fancy tables
- The current output style moves here

**`--json` flag: Machine-parseable (unchanged)**
- Structured JSON for programmatic use

### Implementation

```
internal/output/
  output.go       → routing logic (agent/human/json)
  agent.go        → plain text formatters (default)
  human.go        → rich terminal formatters (--human)
  json.go         → JSON output (--json, unchanged)
```

**Core pattern:**
```go
// Global mode set by flag parsing
var Mode = ModeAgent  // default

const (
    ModeAgent Mode = iota  // plain text, no decoration
    ModeHuman              // colors, emojis, progress bars
    ModeJSON               // structured JSON
)

// Each formatter implements the same interface
type Formatter interface {
    Table(headers []string, rows [][]string)
    Success(msg string)
    Error(msg string)
    Info(msg string)
    Progress(completed, total int, label string)
    KeyValue(pairs []KV)
    Section(title string)
    List(items []string)
}
```

**Agent output examples:**
```
# repos list
Repositories (36 total)

Name                      Source       Enabled  Status
-------------------------+-----------+---------+--------
agentcore-agent           CodeCommit   yes      ok
agentcore-chat-agent      CodeCommit   yes      ok
...

# workflows progress
Daily Investigation Progress

Workflow:  investigate-daily-1772344932677
Started:   2026-03-01T06:02:12Z
Elapsed:   82m57s
Progress:  6/36 (16%)

Status     Count
---------+------
Completed  6
Running    3
Failed     0
Pending    27

Completed:
  agentcore-agent            24m46s
  agentcore-chat-ui          24m32s
  ...

In Progress:
  bedlam-infra               24m59s
  bedlam-next                24m46s
  cloudbubbles               24m40s
```

**Human output: same as current** (emojis, colors, progress bars)

### Migration Strategy
1. Create `Formatter` interface + `AgentFormatter` + `HumanFormatter`
2. Replace all direct `fmt.Printf` with `output.F.Table(...)`, `output.F.Success(...)` etc.
3. Add `--human` global flag, remove `--no-color` (agent mode is already no-color)
4. Keep `--json` as-is

---

## 2. Command Audit & Restructuring

### Current Command Tree (22 commands)
```
reposwarm
├── config          (CLI config)
│   ├── init
│   ├── set
│   └── show
├── server-config   (server config)
│   ├── set
│   └── show
├── new             (bootstrap install)
├── upgrade         (self-update)
├── status          (health check)
├── doctor          (deep diagnostics)
├── discover        (find CodeCommit repos)
├── repos
│   ├── list
│   ├── show
│   ├── add
│   ├── remove
│   ├── enable
│   └── disable
├── investigate     (trigger investigation)
├── workflows
│   ├── list
│   ├── status
│   ├── progress
│   └── terminate
├── results
│   ├── list
│   ├── show
│   ├── read
│   ├── meta
│   ├── search
│   └── export
├── prompts
│   ├── list
│   ├── show
│   ├── create
│   ├── delete
│   ├── toggle
│   ├── order
│   ├── context
│   ├── rollback
│   ├── types
│   ├── export
│   └── import
├── report          (generate markdown report)
├── diff            (compare results)
└── watch           (live poll)
```

### Issues Found

| Issue | Problem | Fix |
|-------|---------|-----|
| `discover` is a top-level orphan | Does the same thing as a repos subcommand | Move under `repos` |
| `server-config` vs `config` | Confusing split, awkward name | Merge: `config` for CLI, `config server` for server |
| `status` vs `doctor` | Overlapping — both check health | Merge: `status` for quick check, `status --deep` for doctor |
| `watch` is top-level | It's a workflow operation | Move under `workflows watch` |
| `investigate` is top-level | It triggers a workflow | Keep top-level (high-frequency), but alias `repos investigate` |
| `repos enable/disable` | Two commands for a toggle | Keep both (explicit is better) |
| `results show` vs `results read` | `show` lists sections, `read` gets content — naming is confusing | Rename: `results sections` and `results read` |
| `diff` is top-level | It compares results | Move under `results diff` |
| `report` is top-level | It generates from results | Move under `results report` |
| `new` naming | "new" is vague | Rename to `init` (standard convention) |

### Proposed Command Tree

```
reposwarm
├── init                    (was: new — bootstrap installation)
├── upgrade                 (self-update)
├── status                  (quick health: API + Temporal + Worker)
│   └── --deep              (was: doctor — full diagnostics)
├── config
│   ├── show                (CLI config)
│   ├── set                 (CLI config)
│   ├── init                (interactive wizard)
│   ├── server              (was: server-config show)
│   └── server set          (was: server-config set)
├── repos
│   ├── list
│   ├── show <name>
│   ├── add <name>
│   ├── remove <name>
│   ├── enable <name>
│   ├── disable <name>
│   └── discover            (was: top-level discover)
├── investigate [repo]      (top-level for quick access)
│   └── --all / --daily
├── workflows               (alias: wf)
│   ├── list
│   ├── status <id>
│   ├── progress            (daily investigation summary)
│   ├── watch [id]          (was: top-level watch)
│   └── terminate <id>
├── results                 (alias: res)
│   ├── list                (repos with results)
│   ├── sections <repo>     (was: show — list sections for a repo)
│   ├── read <repo> [sect]  (read content)
│   ├── meta <repo>         (metadata only)
│   ├── search <query>      (cross-repo search)
│   ├── diff <r1> <r2>      (was: top-level diff)
│   ├── report [repos...]   (was: top-level report)
│   └── export <repo>       (full markdown export)
└── prompts
    ├── list
    ├── show <name>
    ├── create
    ├── delete <name>
    ├── toggle <name>
    ├── order <name>
    ├── context <name>
    ├── rollback <name>
    ├── types
    ├── export
    └── import <file>
```

### Key Changes Summary

| Change | Reason |
|--------|--------|
| `new` → `init` | Standard CLI convention (git init, npm init) |
| `doctor` → `status --deep` | Eliminate confusion; one command, two depths |
| `discover` → `repos discover` | It's a repo operation, not a top-level action |
| `server-config` → `config server` | Natural nesting, fewer top-level commands |
| `watch` → `workflows watch` | It watches workflows — belongs there |
| `diff` → `results diff` | Compares results — belongs with results |
| `report` → `results report` | Generates from results — belongs with results |
| `results show` → `results sections` | Clearer: "show" is too generic |
| Keep `investigate` top-level | High-frequency command, quick access matters |
| Deprecation aliases for old names | `discover`, `watch`, `diff`, `report`, `new` still work with deprecation warning |

### New Options to Add

| Command | New Option | Purpose |
|---------|-----------|---------|
| `repos list` | `--format table\|csv\|names` | Agent-friendly: `--format names` for just repo names |
| `repos list` | `--count` | Just print the count |
| `investigate` | `--watch` | Auto-attach `workflows watch` after triggering |
| `workflows progress` | `--wait` | Block until daily workflow completes |
| `workflows progress` | `--notify` | Send notification when done (webhook URL) |
| `results read` | `--raw` | No markdown headers, just content |
| `results search` | `--repo <name>` | Scope search to one repo |
| `prompts list` | `--enabled-only` | Filter to active prompts |
| Global | `--quiet` / `-q` | Suppress all output except errors and data |

---

## 3. Implementation Order

### Phase 1: Output Architecture (foundation)
1. Create `Formatter` interface
2. Implement `AgentFormatter` (plain text, default)
3. Implement `HumanFormatter` (current style, `--human`)
4. Add `--human` global flag, wire up mode selection
5. Migrate all commands to use `output.F.*` instead of direct fmt
6. Remove `--no-color` (agent mode handles this)

### Phase 2: Command Restructuring
1. Move `discover` → `repos discover` (keep alias)
2. Move `watch` → `workflows watch` (keep alias)
3. Move `diff` → `results diff` (keep alias)
4. Move `report` → `results report` (keep alias)
5. Merge `server-config` → `config server`
6. Merge `doctor` → `status --deep`
7. Rename `new` → `init` (keep alias)
8. Rename `results show` → `results sections`
9. Add deprecation warnings on old paths

### Phase 3: New Options
1. Add format/count options to list commands
2. Add `--watch` to investigate
3. Add `--wait`/`--notify` to progress
4. Add `--quiet` global flag

### Phase 4: Polish
1. Update all `--help` text
2. Update README with new command tree
3. Add shell completion for new structure
4. Write migration guide for existing users
