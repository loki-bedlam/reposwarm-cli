package bootstrap

import (
	"fmt"
	"strings"
)

// GenerateGuide creates a markdown installation guide tailored to the detected environment.
func GenerateGuide(env *Environment, installDir string) string {
	var sb strings.Builder

	sb.WriteString("# RepoSwarm Local Installation Guide\n\n")
	sb.WriteString(fmt.Sprintf("Generated for: **%s/%s**\n", env.OS, env.Arch))
	sb.WriteString(fmt.Sprintf("Install directory: `%s`\n\n", installDir))

	// Table of contents
	sb.WriteString("## Contents\n\n")
	sb.WriteString("1. [Prerequisites](#prerequisites)\n")
	sb.WriteString("2. [Temporal Server](#temporal-server)\n")
	sb.WriteString("3. [RepoSwarm Worker](#reposwarm-worker)\n")
	sb.WriteString("4. [RepoSwarm API Server](#reposwarm-api-server)\n")
	sb.WriteString("5. [RepoSwarm UI](#reposwarm-ui)\n")
	sb.WriteString("6. [Configuration](#configuration)\n")
	sb.WriteString("7. [Verification](#verification)\n\n")

	sb.WriteString("---\n\n")

	// Prerequisites
	sb.WriteString("## Prerequisites\n\n")
	missing := env.MissingDeps()
	if len(missing) > 0 {
		sb.WriteString("### ⚠️ Missing dependencies — install these first:\n\n")
		sb.WriteString(installInstructions(env, missing))
	} else {
		sb.WriteString("✅ All required dependencies are installed.\n\n")
	}

	sb.WriteString("### Required\n")
	sb.WriteString("- Docker & Docker Compose (for Temporal)\n")
	sb.WriteString("- Node.js 22+ (for API server & UI)\n")
	sb.WriteString("- Python 3.11+ (for worker)\n")
	sb.WriteString("- Git\n\n")

	sb.WriteString("### Optional\n")
	sb.WriteString("- AWS CLI (for CodeCommit repo discovery)\n")
	sb.WriteString("- Go 1.24+ (for CLI development)\n\n")

	// Temporal
	sb.WriteString("## Temporal Server\n\n")
	sb.WriteString("Temporal orchestrates the investigation workflows.\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("cd %s\n", installDir))
	sb.WriteString("mkdir -p temporal && cd temporal\n\n")
	sb.WriteString("cat > docker-compose.yml << 'EOF'\n")
	sb.WriteString(temporalCompose())
	sb.WriteString("EOF\n\n")
	sb.WriteString("docker compose up -d\n")
	sb.WriteString("```\n\n")
	sb.WriteString("Verify: `curl http://localhost:7233/api/v1/namespaces` should return JSON.\n")
	sb.WriteString("Temporal UI: http://localhost:8233\n\n")

	// Worker
	sb.WriteString("## RepoSwarm Worker\n\n")
	sb.WriteString("The worker runs AI-powered architecture investigations.\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("cd %s\n", installDir))
	sb.WriteString("git clone https://github.com/royosherove/repo-swarm.git worker\n")
	sb.WriteString("cd worker\n\n")
	sb.WriteString("# Create virtual environment\n")
	sb.WriteString("python3 -m venv .venv\n")
	sb.WriteString("source .venv/bin/activate\n\n")
	sb.WriteString("# Install dependencies\n")
	sb.WriteString("pip install -r requirements.txt\n\n")
	sb.WriteString("# Configure environment\n")
	sb.WriteString("cat > .env << 'EOF'\n")
	sb.WriteString("TEMPORAL_ADDRESS=localhost:7233\n")
	sb.WriteString("TEMPORAL_NAMESPACE=default\n")
	sb.WriteString("TEMPORAL_TASK_QUEUE=investigate-task-queue\n")
	sb.WriteString(fmt.Sprintf("AWS_REGION=%s\n", env.AWSRegion))
	sb.WriteString("DYNAMODB_TABLE=reposwarm-cache\n")
	sb.WriteString("DEFAULT_MODEL=us.anthropic.claude-sonnet-4-6\n")
	sb.WriteString("EOF\n\n")
	sb.WriteString("# Start the worker\n")
	sb.WriteString("python -m worker.main\n")
	sb.WriteString("```\n\n")

	// API Server
	sb.WriteString("## RepoSwarm API Server\n\n")
	sb.WriteString("REST API that the CLI and UI talk to.\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("cd %s\n", installDir))
	sb.WriteString("git clone https://github.com/loki-bedlam/reposwarm-api.git api\n")
	sb.WriteString("cd api\n\n")
	sb.WriteString("npm install\n\n")
	sb.WriteString("# Configure environment\n")
	sb.WriteString("cat > .env << 'EOF'\n")
	sb.WriteString("PORT=3000\n")
	sb.WriteString("TEMPORAL_ADDRESS=localhost:7233\n")
	sb.WriteString("TEMPORAL_NAMESPACE=default\n")
	sb.WriteString("TEMPORAL_TASK_QUEUE=investigate-task-queue\n")
	sb.WriteString(fmt.Sprintf("AWS_REGION=%s\n", env.AWSRegion))
	sb.WriteString("DYNAMODB_TABLE=reposwarm-cache\n")
	sb.WriteString("BEARER_TOKEN=your-secret-token-here\n")
	sb.WriteString("EOF\n\n")
	sb.WriteString("# Build and start\n")
	sb.WriteString("npm run build\n")
	sb.WriteString("npm start\n")
	sb.WriteString("```\n\n")
	sb.WriteString("API will be at: http://localhost:3000/v1/health\n\n")

	// UI
	sb.WriteString("## RepoSwarm UI\n\n")
	sb.WriteString("Next.js dashboard for browsing repos, results, and workflows.\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString(fmt.Sprintf("cd %s\n", installDir))
	sb.WriteString("git clone https://github.com/loki-bedlam/reposwarm-ui.git ui\n")
	sb.WriteString("cd ui\n\n")
	sb.WriteString("npm install\n\n")
	sb.WriteString("# Configure environment\n")
	sb.WriteString("cat > .env.local << 'EOF'\n")
	sb.WriteString("NEXT_PUBLIC_API_URL=http://localhost:3000\n")
	sb.WriteString("EOF\n\n")
	sb.WriteString("npm run dev\n")
	sb.WriteString("```\n\n")
	sb.WriteString("UI will be at: http://localhost:3001\n\n")

	// Configuration
	sb.WriteString("## Configuration\n\n")
	sb.WriteString("Connect the CLI to your local API server:\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("reposwarm config set apiUrl http://localhost:3000/v1\n")
	sb.WriteString("reposwarm config set apiToken your-secret-token-here\n")
	sb.WriteString("reposwarm status\n")
	sb.WriteString("```\n\n")

	// DynamoDB note
	sb.WriteString("### DynamoDB\n\n")
	sb.WriteString("RepoSwarm stores repo metadata and investigation results in DynamoDB.\n\n")
	sb.WriteString("**Option A: AWS DynamoDB** (requires AWS credentials)\n")
	sb.WriteString("- Set `AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` in each `.env`\n")
	sb.WriteString("- Table `reposwarm-cache` must exist (HASH: `repository_name` S, RANGE: `analysis_timestamp` N)\n\n")
	sb.WriteString("**Option B: DynamoDB Local** (no AWS account needed)\n")
	sb.WriteString("```bash\n")
	sb.WriteString("docker run -d -p 8000:8000 amazon/dynamodb-local\n")
	sb.WriteString("# Add to each .env:\n")
	sb.WriteString("# DYNAMODB_ENDPOINT=http://localhost:8000\n")
	sb.WriteString("```\n\n")

	// Verification
	sb.WriteString("## Verification\n\n")
	sb.WriteString("Run these to confirm everything works:\n\n")
	sb.WriteString("```bash\n")
	sb.WriteString("# Check API\n")
	sb.WriteString("reposwarm status\n\n")
	sb.WriteString("# List repos\n")
	sb.WriteString("reposwarm repos list\n\n")
	sb.WriteString("# Discover CodeCommit repos (if AWS configured)\n")
	sb.WriteString("reposwarm discover\n\n")
	sb.WriteString("# Trigger investigation\n")
	sb.WriteString("reposwarm investigate <repo-name>\n\n")
	sb.WriteString("# Watch it run\n")
	sb.WriteString("reposwarm watch\n")
	sb.WriteString("```\n\n")

	sb.WriteString("---\n\n")
	sb.WriteString("## Architecture\n\n")
	sb.WriteString("```\n")
	sb.WriteString("┌──────────────┐     ┌──────────────┐     ┌──────────────┐\n")
	sb.WriteString("│  CLI / UI    │────▶│  API Server   │────▶│  Temporal    │\n")
	sb.WriteString("│  (client)    │     │  (Express)    │     │  (workflow)  │\n")
	sb.WriteString("└──────────────┘     └──────────────┘     └──────┬───────┘\n")
	sb.WriteString("                                                  │\n")
	sb.WriteString("                     ┌──────────────┐     ┌──────▼───────┐\n")
	sb.WriteString("                     │  DynamoDB     │◀────│  Worker      │\n")
	sb.WriteString("                     │  (storage)    │     │  (Python/AI) │\n")
	sb.WriteString("                     └──────────────┘     └──────────────┘\n")
	sb.WriteString("```\n")

	return sb.String()
}

func temporalCompose() string {
	return `services:
  temporal:
    image: temporalio/auto-setup:latest
    ports:
      - "7233:7233"
    environment:
      - DB=sqlite
      - DYNAMIC_CONFIG_FILE_PATH=config/dynamicconfig/development-sql.yaml
      - SKIP_DEFAULT_NAMESPACE_CREATION=false

  temporal-ui:
    image: temporalio/ui:latest
    ports:
      - "8233:8080"
    environment:
      - TEMPORAL_ADDRESS=temporal:7233
    depends_on:
      - temporal
`
}

func installInstructions(env *Environment, missing []string) string {
	var sb strings.Builder
	for _, dep := range missing {
		sb.WriteString(fmt.Sprintf("**%s:**\n", dep))
		switch {
		case strings.HasPrefix(dep, "docker"):
			if env.OS == "darwin" {
				sb.WriteString("```bash\nbrew install --cask docker\n```\n")
			} else if env.HasApt {
				sb.WriteString("```bash\ncurl -fsSL https://get.docker.com | sh\n```\n")
			} else {
				sb.WriteString("Visit https://docs.docker.com/get-docker/\n")
			}
		case strings.HasPrefix(dep, "node"):
			if env.HasBrew {
				sb.WriteString("```bash\nbrew install node@22\n```\n")
			} else {
				sb.WriteString("```bash\ncurl -fsSL https://deb.nodesource.com/setup_22.x | sudo -E bash -\nsudo apt-get install -y nodejs\n```\n")
			}
		case strings.HasPrefix(dep, "python"):
			if env.HasBrew {
				sb.WriteString("```bash\nbrew install python@3.12\n```\n")
			} else if env.HasApt {
				sb.WriteString("```bash\nsudo apt-get install -y python3 python3-venv python3-pip\n```\n")
			}
		case dep == "git":
			if env.HasBrew {
				sb.WriteString("```bash\nbrew install git\n```\n")
			} else if env.HasApt {
				sb.WriteString("```bash\nsudo apt-get install -y git\n```\n")
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}
