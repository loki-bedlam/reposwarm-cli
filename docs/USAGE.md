# RepoSwarm CLI — Agent Usage Guide

## Overview

`reposwarm` is a Go CLI for the RepoSwarm platform. Single binary, 4ms startup, 
`--json` flag on every command. Designed for both humans and AI agents.

## Setup

```bash
# Option 1: Config file (interactive)
reposwarm config init

# Option 2: Environment variables
export REPOSWARM_API_URL=https://dkhtk1q9b2nii.cloudfront.net/v1
export REPOSWARM_API_TOKEN=YOUR_TOKEN

# Option 3: Inline flags (per-command)
reposwarm --api-url URL --api-token TOKEN repos list
```

## Command Reference (Agent-Optimized)

### Check connection
```bash
reposwarm status --json
# {"connected":true,"status":"healthy","version":"1.0.0","latency":309,...}
```

### List repositories
```bash
reposwarm repos list --json
# [{"name":"repo1","source":"CodeCommit","enabled":true,"hasDocs":true},...]

# Filter
reposwarm repos list --source GitHub --json
reposwarm repos list --filter "mesh" --json
reposwarm repos list --enabled --json
```

### Discover CodeCommit repos
```bash
reposwarm discover --json
# {"success":true,"discovered":36,"added":5,"skipped":31,"total":36}
```

### Trigger investigation
```bash
reposwarm investigate my-repo --json
# {"workflowId":"investigate-single-my-repo","success":true}

reposwarm investigate --all --json
```

### Monitor workflows
```bash
reposwarm workflows list --json --limit 10
# [{"workflowId":"wf-1","status":"Running","type":"Investigate",...},...]

reposwarm workflows status WORKFLOW_ID --json
```

### Read results
```bash
# List repos with results
reposwarm results list --json

# List sections
reposwarm results show is-odd --json

# Read single section
reposwarm results read is-odd hl_overview --json
# {"repo":"is-odd","section":"hl_overview","content":"...","createdAt":"..."}

# Read ALL sections (raw markdown)
reposwarm results read is-odd --raw

# Metadata only
reposwarm results meta is-odd hl_overview --json

# Search across all results
reposwarm results search "authentication" --json
# [{"repo":"is-odd","section":"authentication","line":"..."},...]

# Export
reposwarm results export is-odd -o report.md
```

### Compare repos
```bash
reposwarm diff repo1 repo2 --json
# {"repo1":"repo1","repo2":"repo2","shared":["overview","apis"],"only1":["security"],"only2":[]}
```

### Manage prompts
```bash
reposwarm prompts list --json
reposwarm prompts show overview --json
reposwarm prompts create my-prompt --type base --template-file prompt.md
reposwarm prompts toggle my-prompt
reposwarm prompts versions my-prompt --json
reposwarm prompts rollback my-prompt 2
reposwarm prompts export -o backup.json
reposwarm prompts import backup.json
```

### Server configuration
```bash
reposwarm server-config show --json
# {"defaultModel":"us.anthropic.claude-sonnet-4-6","chunkSize":10,...}
```

## Exit Codes
- `0` — success
- `1` — error (message on stderr)

## Output Behavior
- `--json` → valid JSON to stdout; errors to stderr
- Default → colored pretty output to stdout
- `--raw` (on read/show) → raw content only, no headers
- `--no-color` → strip ANSI codes

## Error Handling
- 401 → "run 'reposwarm config init' to update your token"
- 404 → "not found: /path"
- Connection failure → descriptive network error
- Missing config → "run 'reposwarm config init'"
