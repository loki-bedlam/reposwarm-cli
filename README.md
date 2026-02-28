# reposwarm-cli

CLI for [RepoSwarm](https://github.com/loki-bedlam/reposwarm-ui) — AI-powered multi-repo architecture discovery.

Written in Go. Single 8.9MB binary, zero runtime dependencies, 4ms startup.

## Install

```bash
# Build from source (requires Go 1.24+)
git clone https://github.com/loki-bedlam/reposwarm-cli.git
cd reposwarm-cli
go build -o reposwarm ./cmd/reposwarm
sudo mv reposwarm /usr/local/bin/

# Cross-compile
GOOS=linux GOARCH=arm64 go build -o reposwarm-linux-arm64 ./cmd/reposwarm
GOOS=darwin GOARCH=arm64 go build -o reposwarm-darwin-arm64 ./cmd/reposwarm
GOOS=linux GOARCH=amd64 go build -o reposwarm-linux-amd64 ./cmd/reposwarm
```

## Quick Start

```bash
# 1. Configure API connection
reposwarm config init

# 2. Check connection
reposwarm status

# 3. List tracked repos
reposwarm repos list

# 4. Browse investigation results
reposwarm results list
reposwarm results read is-odd
reposwarm results export is-odd -o report.md

# 5. Trigger new investigation
reposwarm investigate my-repo
reposwarm watch investigate-single-my-repo
```

## Commands

### Core
| Command | Description |
|---------|-------------|
| `reposwarm status` | Check API health and service status |
| `reposwarm config init` | Interactive setup wizard |
| `reposwarm config show` | Display current config |
| `reposwarm config set <key> <value>` | Update a config value |

### Repositories
| Command | Description |
|---------|-------------|
| `reposwarm repos list` | List all tracked repos (`--source`, `--filter`, `--enabled`) |
| `reposwarm repos add <name>` | Add a repo (`--url`, `--source`) |
| `reposwarm repos remove <name>` | Remove a repo (`-y` to skip confirm) |
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

### Results
| Command | Description |
|---------|-------------|
| `reposwarm results list` | Repos with investigation results |
| `reposwarm results show <repo>` | List sections for a repo |
| `reposwarm results read <repo>` | Read ALL sections concatenated |
| `reposwarm results read <repo> <section>` | Read single section (`--raw`) |
| `reposwarm results meta <repo> [section]` | Metadata only (no content) |
| `reposwarm results export <repo> -o file.md` | Export to markdown file |
| `reposwarm results search <query>` | Search across all results |
| `reposwarm diff <repo1> <repo2> [section]` | Compare investigations |

### Prompts
| Command | Description |
|---------|-------------|
| `reposwarm prompts list` | List all prompts (`--type`, `--enabled`) |
| `reposwarm prompts show <name>` | Show prompt details + template (`--raw`) |
| `reposwarm prompts create <name>` | Create prompt (`--type`, `--template-file`) |
| `reposwarm prompts update <name>` | Update template/description |
| `reposwarm prompts delete <name>` | Delete a prompt |
| `reposwarm prompts toggle <name>` | Toggle enabled/disabled |
| `reposwarm prompts order <name> <n>` | Set execution order |
| `reposwarm prompts context <name> <text>` | Set prompt context |
| `reposwarm prompts versions <name>` | Version history |
| `reposwarm prompts rollback <name> <ver>` | Rollback to version |
| `reposwarm prompts types` | List prompt types |
| `reposwarm prompts export -o file.json` | Export all prompts |
| `reposwarm prompts import file.json` | Import prompts |

### Server Configuration
| Command | Description |
|---------|-------------|
| `reposwarm server-config show` | View server-side config |
| `reposwarm server-config set <key> <value>` | Update server config |

## Global Flags

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON (agent-friendly) |
| `--api-url <url>` | Override API URL |
| `--api-token <token>` | Override API token |
| `--no-color` | Disable colored output |
| `--verbose` | Show debug info |
| `-v, --version` | Show version |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `REPOSWARM_API_URL` | API server URL (overrides config file) |
| `REPOSWARM_API_TOKEN` | Bearer token (overrides config file) |

## Agent Usage

All commands support `--json` for machine-readable output:

```bash
# Pipe to jq
reposwarm repos list --json | jq '.[].name'
reposwarm results read is-odd --json | jq '.content'

# Use in scripts
WF_ID=$(reposwarm investigate my-repo --json | jq -r '.workflowId')
reposwarm watch "$WF_ID"

# Export for processing
reposwarm results export my-repo -o report.md
reposwarm prompts export -o prompts-backup.json
```

## Config File

Location: `~/.reposwarm/config.json`

```json
{
  "apiUrl": "https://your-api.example.com/v1",
  "apiToken": "your-bearer-token",
  "region": "us-east-1",
  "defaultModel": "us.anthropic.claude-sonnet-4-6",
  "chunkSize": 10,
  "outputFormat": "pretty"
}
```

## Shell Completions

```bash
reposwarm completion bash >> ~/.bashrc
reposwarm completion zsh >> ~/.zshrc
reposwarm completion fish > ~/.config/fish/completions/reposwarm.fish
```

## Development

```bash
go test ./...                           # Run tests (42 passing)
go test ./... -coverprofile=cov.txt     # Coverage
go vet ./...                            # Lint
go build -o reposwarm ./cmd/reposwarm  # Build
```

## Architecture

```
cmd/reposwarm/          # Entrypoint (main.go)
internal/
  api/                  # HTTP client + response types
  commands/             # All cobra command definitions
  config/               # Config file management
  output/               # Table/JSON/color formatting
docs/                   # Agent-friendly documentation
```

## License

MIT
