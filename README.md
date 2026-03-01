# reposwarm-cli

CLI for [RepoSwarm](https://github.com/loki-bedlam/reposwarm-ui) — AI-powered multi-repo architecture discovery.

Written in Go. Single 9MB binary, zero runtime dependencies, 4ms startup.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/loki-bedlam/reposwarm-cli/main/install.sh | sh
```

Or build from source:
```bash
git clone https://github.com/loki-bedlam/reposwarm-cli.git
cd reposwarm-cli
go build -o reposwarm ./cmd/reposwarm
```

## Quick Start

```bash
reposwarm config init             # Connect to a RepoSwarm server
reposwarm status                  # Check connection
reposwarm repos list              # List tracked repos
reposwarm investigate <repo>      # Run an investigation
reposwarm results sections <repo> # Browse results
```

## Commands

### Setup & Diagnostics
| Command | Description |
|---------|-------------|
| `reposwarm config init` | Interactive setup wizard |
| `reposwarm config show` | Display current config |
| `reposwarm config set <key> <value>` | Update config value |
| `reposwarm config server` | View server-side config |
| `reposwarm config server-set <key> <value>` | Update server config |
| `reposwarm status` | Quick API health check with latency |
| `reposwarm doctor` | Deep diagnosis (config, API, Temporal, DynamoDB, worker, network) |
| `reposwarm new` | Bootstrap a new local installation |
| `reposwarm version` | Print version (`-v` / `--version` also work) |
| `reposwarm upgrade` | Self-update to latest version (`--force` to reinstall) |

### Repositories
| Command | Description |
|---------|-------------|
| `reposwarm repos list` | List all tracked repos (`--source`, `--filter`, `--enabled`) |
| `reposwarm repos show <name>` | Detailed single repo view |
| `reposwarm repos add <name>` | Add a repo (`--url`, `--source`) |
| `reposwarm repos remove <name>` | Remove a repo (`-y` skip confirm) |
| `reposwarm repos enable <name>` | Enable for investigation |
| `reposwarm repos disable <name>` | Disable from investigation |
| `reposwarm repos discover` | Auto-discover CodeCommit repos |

### Investigation & Workflows
| Command | Description |
|---------|-------------|
| `reposwarm investigate <repo>` | Investigate single repo (`--model`, `--chunk-size`) |
| `reposwarm investigate --all` | Investigate all enabled repos (`--parallel`) |
| `reposwarm workflows list` | List recent workflows (`--limit`) |
| `reposwarm workflows status <id>` | Workflow details |
| `reposwarm workflows progress` | Show investigation progress across repos |
| `reposwarm workflows watch [id]` | Watch workflows in real-time (`--interval`) |
| `reposwarm workflows terminate <id>` | Stop a workflow (`-y`, `--reason`) |

### Results & Analysis
| Command | Description |
|---------|-------------|
| `reposwarm results list` | Repos with investigation results |
| `reposwarm results sections <repo>` | List sections for a repo |
| `reposwarm results read <repo> [section]` | Read results (`--raw` for markdown) |
| `reposwarm results meta <repo> [section]` | Metadata only |
| `reposwarm results export <repo> -o file.md` | Export to file |
| `reposwarm results export --all -d ./docs` | Export all repos to directory |
| `reposwarm results search <query>` | Search results (`--repo`, `--section`, `--max`) |
| `reposwarm results audit` | Validate all repos have complete sections |
| `reposwarm results diff <repo1> <repo2>` | Compare investigations |
| `reposwarm results report [repos...] -o f.md` | Consolidated report (`--sections`) |

### Prompts
| Command | Description |
|---------|-------------|
| `reposwarm prompts list` | List prompts (derives from results if API returns empty) |
| `reposwarm prompts show <name>` | Show template (`--raw`) |
| `reposwarm prompts create <name>` | Create (`--type`, `--template-file`) |
| `reposwarm prompts update <name>` | Update template/description |
| `reposwarm prompts delete <name>` | Delete |
| `reposwarm prompts toggle <name>` | Toggle enabled/disabled |

## Global Flags

| Flag | Description |
|------|-------------|
| `--json` | JSON output (agent/script-friendly) |
| `--for-agent` | Plain text output for agents/scripts |
| `--api-url <url>` | Override API URL |
| `--api-token <token>` | Override API token |
| `--verbose` | Debug info |
| `-v` / `--version` | Print version |

Default output is human-friendly (colors, tables). Use `--for-agent` for plain text or `--json` for structured output.

## Environment Variables

| Variable | Description |
|----------|-------------|
| `REPOSWARM_API_URL` | API server URL |
| `REPOSWARM_API_TOKEN` | Bearer token |

## Agent Usage

Every command supports `--json`:

```bash
reposwarm repos list --json | jq '.[].name'
reposwarm results read my-repo --json | jq '.content'
reposwarm doctor --json | jq '.checks[] | select(.status=="fail")'
```

## Development

```bash
go test ./...              # Tests
go vet ./...               # Lint
go build ./cmd/reposwarm   # Build
```

## CI/CD

CodePipeline (`reposwarm-cli-pipeline`):
GitHub push → CodeBuild (Go 1.24, ARM64) → Tests → Cross-compile 4 targets → GitHub Release

## License

MIT
