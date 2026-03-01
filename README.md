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
# Bootstrap a new local RepoSwarm installation
reposwarm new

# Or connect to an existing server
reposwarm config init
reposwarm doctor                  # Check everything is working
reposwarm repos list              # List tracked repos
reposwarm results list            # Browse investigation results
```

## Commands

### Setup & Diagnostics
| Command | Description |
|---------|-------------|
| `reposwarm new` | Bootstrap a new local installation (detects env, generates guides, optionally launches Claude Code/Codex) |
| `reposwarm doctor` | Diagnose installation health (config, API, Temporal, DynamoDB, worker, tools, network) |
| `reposwarm status` | Quick API health check with latency |
| `reposwarm upgrade` | Self-update to latest version |
| `reposwarm config init` | Interactive setup wizard |
| `reposwarm config show` | Display current config |
| `reposwarm config set <key> <value>` | Update config value |

### Repositories
| Command | Description |
|---------|-------------|
| `reposwarm repos list` | List all tracked repos (`--source`, `--filter`, `--enabled`) |
| `reposwarm repos show <name>` | Detailed single repo view |
| `reposwarm repos add <name>` | Add a repo (`--url`, `--source`) |
| `reposwarm repos remove <name>` | Remove a repo (`-y` skip confirm) |
| `reposwarm repos enable <name>` | Enable for investigation |
| `reposwarm repos disable <name>` | Disable from investigation |
| `reposwarm discover` | Auto-discover CodeCommit repos |

### Investigation & Workflows
| Command | Description |
|---------|-------------|
| `reposwarm investigate <repo>` | Investigate single repo (`--model`, `--chunk-size`) |
| `reposwarm investigate --all` | Investigate all enabled repos (`--parallel`) |
| `reposwarm workflows list` | List recent workflows (`--limit`) |
| `reposwarm workflows status <id>` | Workflow details |
| `reposwarm workflows terminate <id>` | Stop a workflow (`-y`, `--reason`) |
| `reposwarm watch [id]` | Watch workflows in real-time (`--interval`) |

### Results & Analysis
| Command | Description |
|---------|-------------|
| `reposwarm results list` | Repos with investigation results |
| `reposwarm results show <repo>` | List sections for a repo |
| `reposwarm results read <repo> [section]` | Read results (`--raw` for markdown) |
| `reposwarm results meta <repo> [section]` | Metadata only |
| `reposwarm results export <repo> -o file.md` | Export to file |
| `reposwarm results search <query>` | Search across all results |
| `reposwarm diff <repo1> <repo2> [section]` | Compare investigations |
| `reposwarm report [repos...] -o file.md` | Consolidated report (`--sections`) |

### Prompts
| Command | Description |
|---------|-------------|
| `reposwarm prompts list` | List prompts (`--type`, `--enabled`) |
| `reposwarm prompts show <name>` | Show template (`--raw`) |
| `reposwarm prompts create <name>` | Create (`--type`, `--template-file`) |
| `reposwarm prompts update <name>` | Update template/description |
| `reposwarm prompts delete <name>` | Delete |
| `reposwarm prompts toggle <name>` | Toggle enabled/disabled |
| `reposwarm prompts order <name> <n>` | Set execution order |
| `reposwarm prompts context <name> <text>` | Set context |
| `reposwarm prompts versions <name>` | Version history |
| `reposwarm prompts rollback <name> <ver>` | Rollback to version |
| `reposwarm prompts types` | List prompt types |
| `reposwarm prompts export -o file.json` | Export all |
| `reposwarm prompts import file.json` | Import |

### Server Configuration
| Command | Description |
|---------|-------------|
| `reposwarm server-config show` | View server-side config |
| `reposwarm server-config set <key> <value>` | Update server config |

## Global Flags

| Flag | Description |
|------|-------------|
| `--json` | JSON output (agent-friendly) |
| `--api-url <url>` | Override API URL |
| `--api-token <token>` | Override API token |
| `--no-color` | Disable colors |
| `--verbose` | Debug info |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `REPOSWARM_API_URL` | API server URL |
| `REPOSWARM_API_TOKEN` | Bearer token |

## Agent Usage

Every command supports `--json`:

```bash
reposwarm repos list --json | jq '.[].name'
reposwarm results read is-odd --json | jq '.content'
reposwarm new --json | jq '.environment'
reposwarm doctor --json | jq '.checks[] | select(.status=="fail")'
```

## Development

```bash
go test ./...           # 59 tests
go vet ./...            # Lint
go build ./cmd/reposwarm  # Build
```

## Architecture

```
cmd/reposwarm/          # Entrypoint
internal/
  api/                  # HTTP client + types
  bootstrap/            # Environment detection + guide generation
  commands/             # All cobra commands
  config/               # Config file management
  output/               # Table/JSON/color formatting
docs/                   # Agent-friendly docs
```

## CI/CD

CodePipeline (`reposwarm-cli-pipeline`):
GitHub → CodeBuild (Go 1.24, ARM64) → Tests → Cross-compile 4 targets → S3 + GitHub Release

## License

MIT
# Auto-trigger test 19:29
