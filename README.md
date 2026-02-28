# reposwarm-cli

CLI for [RepoSwarm](https://github.com/loki-bedlam/reposwarm-ui) — AI-powered multi-repo architecture discovery.

Written in Go. Single binary, zero dependencies, instant startup.

## Install

**Download binary:**
```bash
# Linux ARM64
curl -sL https://github.com/loki-bedlam/reposwarm-cli/releases/latest/download/reposwarm-linux-arm64 -o reposwarm
chmod +x reposwarm
sudo mv reposwarm /usr/local/bin/
```

**Build from source:**
```bash
git clone https://github.com/loki-bedlam/reposwarm-cli.git
cd reposwarm-cli
go build -o reposwarm ./cmd/reposwarm
```

## Quick Start

```bash
# 1. Configure API connection
reposwarm config init

# 2. List tracked repos
reposwarm repos list

# 3. Browse investigation results
reposwarm results list
reposwarm results read is-odd

# 4. Trigger new investigation
reposwarm investigate my-repo
```

## Commands

### Configuration
| Command | Description |
|---------|-------------|
| `reposwarm config init` | Interactive setup (API URL + token) |
| `reposwarm config show` | Display current config |
| `reposwarm config set <key> <value>` | Update a config value |

### Repositories
| Command | Description |
|---------|-------------|
| `reposwarm repos list` | List all tracked repos |
| `reposwarm repos add <name>` | Add a repo |
| `reposwarm repos remove <name>` | Remove a repo |
| `reposwarm repos enable <name>` | Enable for investigation |
| `reposwarm repos disable <name>` | Disable from investigation |
| `reposwarm discover` | Auto-discover CodeCommit repos |

### Investigation
| Command | Description |
|---------|-------------|
| `reposwarm investigate <repo>` | Investigate single repo |
| `reposwarm investigate --all` | Investigate all enabled repos |

### Workflows
| Command | Description |
|---------|-------------|
| `reposwarm workflows list` | List recent workflows |
| `reposwarm workflows status <id>` | Workflow details |
| `reposwarm workflows terminate <id>` | Stop a workflow |

### Results
| Command | Description |
|---------|-------------|
| `reposwarm results list` | Repos with investigation results |
| `reposwarm results show <repo>` | List sections for a repo |
| `reposwarm results read <repo>` | Read ALL sections (one document) |
| `reposwarm results read <repo> <section>` | Read single section |
| `reposwarm results meta <repo> [section]` | Metadata only (no content) |
| `reposwarm results export <repo> -o file.md` | Export to file |
| `reposwarm results search <query>` | Search across all results |

## Global Flags

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON (agent-friendly) |
| `--api-url <url>` | Override API URL |
| `--api-token <token>` | Override API token |
| `--no-color` | Disable colored output |
| `--verbose` | Show debug info |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `REPOSWARM_API_URL` | API server URL |
| `REPOSWARM_API_TOKEN` | Bearer token for authentication |

## Agent Usage

All commands support `--json` for machine-readable output:

```bash
# Get repos as JSON array
reposwarm repos list --json | jq '.[].name'

# Get investigation results as JSON
reposwarm results read is-odd hl_overview --json | jq '.content'

# Trigger and get workflow ID
reposwarm investigate my-repo --json | jq '.workflowId'
```

## Config File

Stored at `~/.reposwarm/config.json`:

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
# Bash
reposwarm completion bash >> ~/.bashrc

# Zsh
reposwarm completion zsh >> ~/.zshrc

# Fish
reposwarm completion fish > ~/.config/fish/completions/reposwarm.fish
```

## License

MIT
