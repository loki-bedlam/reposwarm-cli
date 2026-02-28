# reposwarm-cli

CLI for [RepoSwarm](https://github.com/loki-bedlam) — AI-powered multi-repo architecture discovery. Built for humans and agents.

## Install

```bash
npm install -g reposwarm-cli

# Or run directly
npx reposwarm-cli discover
```

## Quick Start

```bash
# Discover all CodeCommit repos and add to tracking
reposwarm discover

# List tracked repos
reposwarm repos list

# Investigate a single repo
reposwarm investigate my-repo

# Investigate all enabled repos (auto-discovers if none exist)
reposwarm investigate --all --discover

# Agent-friendly JSON output
reposwarm repos list --json
reposwarm discover --json
```

## Commands

### `reposwarm discover`

Auto-discover repositories from CodeCommit.

```bash
reposwarm discover                    # Discover and add new repos
reposwarm discover --dry-run          # Preview without adding
reposwarm discover --force            # Re-add even if already tracked
reposwarm discover --source codecommit
```

### `reposwarm repos`

Manage tracked repositories.

```bash
reposwarm repos list                              # List all
reposwarm repos list --source CodeCommit           # Filter by source
reposwarm repos list --enabled                     # Only enabled
reposwarm repos list --filter "mesh"               # Case-insensitive name search
reposwarm repos add my-repo --url https://...      # Add manually
reposwarm repos remove my-repo                     # Remove from tracking
reposwarm repos enable my-repo                     # Enable for investigation
reposwarm repos disable my-repo                    # Disable
```

### `reposwarm investigate`

Trigger architecture investigation workflows.

```bash
reposwarm investigate my-repo                      # Single repo
reposwarm investigate --all                        # All enabled repos
reposwarm investigate --all --discover             # Auto-discover first, then investigate all
reposwarm investigate my-repo --model us.anthropic.claude-opus-4-6-v1
reposwarm investigate --all --chunk-size 20 --parallel 5
```

Options:
| Flag | Default | Description |
|------|---------|-------------|
| `--model` | `us.anthropic.claude-sonnet-4-6` | Bedrock model ID |
| `--chunk-size` | `10` | Files per analysis chunk |
| `--parallel` | `3` | Parallel repo limit (daily) |
| `--discover` | off | Auto-discover repos first |
| `--temporal-url` | `http://temporal-alb-internal:8233` | Temporal HTTP API |
| `--namespace` | `default` | Temporal namespace |
| `--task-queue` | `investigate-task-queue` | Temporal task queue |

### `reposwarm workflows`

Manage investigation workflows.

```bash
reposwarm workflows list                           # List recent
reposwarm workflows list --status Running          # Filter by status
reposwarm workflows status <workflowId>            # Get details
reposwarm workflows terminate <workflowId>         # Kill a workflow
reposwarm workflows terminate <id> --reason "Too slow"
```

### `reposwarm config`

Show current configuration.

```bash
reposwarm config
reposwarm config --json
```

## Global Options

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | off | JSON output (agent-friendly) |
| `--region` | `us-east-1` | AWS region |
| `--profile` | default | AWS CLI profile |
| `--table` | `reposwarm-cache` | DynamoDB table name |

## AWS IAM Permissions

The CLI (or the ECS task role running the RepoSwarm UI) needs the following IAM permissions:

### Required Policy

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "DynamoDBRepoTracking",
      "Effect": "Allow",
      "Action": [
        "dynamodb:Scan",
        "dynamodb:Query",
        "dynamodb:GetItem",
        "dynamodb:PutItem",
        "dynamodb:DeleteItem",
        "dynamodb:UpdateItem",
        "dynamodb:DescribeTable"
      ],
      "Resource": "arn:aws:dynamodb:*:*:table/reposwarm-cache"
    },
    {
      "Sid": "CodeCommitDiscovery",
      "Effect": "Allow",
      "Action": [
        "codecommit:ListRepositories",
        "codecommit:GetRepository",
        "codecommit:BatchGetRepositories"
      ],
      "Resource": "*"
    }
  ]
}
```

### For ECS Task Roles (RepoSwarm UI)

If running the RepoSwarm UI on ECS Fargate, attach the above policy to the **task role** (not the execution role). The task role is what the application code uses at runtime.

Example with AWS CLI:
```bash
aws iam put-role-policy \
  --role-name reposwarm-ui-task \
  --policy-name reposwarm-access \
  --policy-document file://policy.json
```

### Optional: GitHub Discovery (Future)

When GitHub discovery is added, you'll also need a GitHub personal access token with `repo` scope. Configure via:
```bash
export GITHUB_TOKEN=ghp_...
# or
reposwarm config set github.token ghp_...
```

## Agent Usage

The `--json` flag on every command makes this CLI agent-friendly:

```bash
# Agent discovers repos and gets structured output
REPOS=$(reposwarm discover --json)

# Agent triggers investigation and captures workflow ID
RESULT=$(reposwarm investigate my-repo --json)
WORKFLOW_ID=$(echo $RESULT | jq -r '.workflowId')

# Agent monitors workflow
reposwarm workflows status $WORKFLOW_ID --json
```

All errors are also JSON when `--json` is set:
```json
{
  "error": "Discovery failed",
  "details": "..."
}
```

## Development

```bash
git clone https://github.com/loki-bedlam/reposwarm-cli
cd reposwarm-cli
npm install
npm run type-check     # TypeScript validation
npm test               # Run tests
npm run dev -- discover  # Run locally via tsx
npm run build          # Compile to dist/
```

## Tech Stack

- **TypeScript** + Node.js 24
- **Commander.js** — CLI framework
- **AWS SDK v3** — CodeCommit + DynamoDB
- **Chalk** — Terminal colors
- **Vitest** — Testing

## License

MIT
