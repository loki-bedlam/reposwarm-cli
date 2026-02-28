# RepoSwarm CLI — Agent Usage Guide

## Overview

`reposwarm` is a Go CLI for interacting with the RepoSwarm platform.
It communicates with the RepoSwarm API server via HTTP + Bearer token auth.

## Setup

```bash
# Option 1: Config file
reposwarm config set apiUrl https://dkhtk1q9b2nii.cloudfront.net/v1
reposwarm config set apiToken YOUR_TOKEN

# Option 2: Environment variables
export REPOSWARM_API_URL=https://dkhtk1q9b2nii.cloudfront.net/v1
export REPOSWARM_API_TOKEN=YOUR_TOKEN

# Option 3: Inline flags
reposwarm --api-url URL --api-token TOKEN repos list
```

## Common Patterns

### List all repos as JSON
```bash
reposwarm repos list --json
```
Returns: `[{"name":"repo1","source":"CodeCommit","enabled":true,...},...]`

### Get full investigation for a repo
```bash
reposwarm results read my-repo --raw
```
Returns: All sections concatenated as raw markdown.

### Get single section
```bash
reposwarm results read my-repo hl_overview --json
```
Returns: `{"repo":"my-repo","section":"hl_overview","content":"...","createdAt":"..."}`

### Trigger investigation and get workflow ID
```bash
reposwarm investigate my-repo --json
```

### Check workflow status
```bash
reposwarm workflows status WORKFLOW_ID --json
```

### Export full report
```bash
reposwarm results export my-repo -o report.md
```

## Exit Codes
- `0` — success
- `1` — error (message on stderr)

## JSON Output

All commands with `--json` output valid JSON to stdout.
Error messages go to stderr. This means you can safely pipe:
```bash
reposwarm repos list --json 2>/dev/null | jq '.[] | .name'
```
