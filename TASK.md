# TASK: Add `reposwarm new --local` for automated local environment setup

## Goal
Add a `--local` flag to `reposwarm new` that **actually creates and starts** a complete local RepoSwarm dev environment — Temporal, API server, Worker, UI — then configures the CLI and verifies everything works.

## Current State
- `reposwarm new` only generates markdown guides (INSTALL.md, REPOSWARM_INSTALL.md)
- Optionally hands off to a coding agent (`--agent`)
- Does NOT execute any setup steps

## What `--local` Must Do (in order)

### 1. Check prerequisites
- Docker + Docker Compose must be present
- Node.js, Python3, Git must be present
- If anything missing, print clear error and exit

### 2. Create directory structure
```
{installDir}/
├── temporal/docker-compose.yml
├── worker/   (cloned from GitHub)
├── api/      (cloned from GitHub)
└── ui/       (cloned from GitHub)
```

### 3. Start Temporal via Docker Compose

**CRITICAL BUG FIX:** The current `temporalCompose()` in `internal/bootstrap/guide.go` uses `DB=sqlite` which is NO LONGER SUPPORTED by `temporalio/auto-setup`. The error is:
```
Unsupported driver specified: 'DB=sqlite'. Valid drivers are: mysql8, postgres12, postgres12_pgx, cassandra.
```

**Fix:** Use postgres companion container:
```yaml
services:
  temporal:
    image: temporalio/auto-setup:latest
    ports:
      - "7233:7233"
    environment:
      - DB=postgres12
      - POSTGRES_USER=temporal
      - POSTGRES_PWD=temporal
      - POSTGRES_SEEDS=postgres
      - DYNAMIC_CONFIG_FILE_PATH=config/dynamicconfig/development-sql.yaml
    depends_on:
      postgres:
        condition: service_healthy

  postgres:
    image: postgres:16-alpine
    ports:
      - "5432:5432"
    environment:
      - POSTGRES_USER=temporal
      - POSTGRES_PASSWORD=temporal
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U temporal"]
      interval: 5s
      timeout: 5s
      retries: 10
    volumes:
      - temporal-data:/var/lib/postgresql/data

  temporal-ui:
    image: temporalio/ui:latest
    ports:
      - "8233:8080"
    environment:
      - TEMPORAL_ADDRESS=temporal:7233
    depends_on:
      - temporal

volumes:
  temporal-data:
```

**UPDATE the `temporalCompose()` function in guide.go too** so the generated guides also have the correct compose.

### 4. Clone and start API server
```bash
cd {installDir}
git clone https://github.com/loki-bedlam/reposwarm-api.git api
cd api && npm install && npm run build
```
Create `.env` with:
- `PORT=3000`
- `TEMPORAL_ADDRESS=localhost:7233`
- `TEMPORAL_NAMESPACE=default`
- `TEMPORAL_TASK_QUEUE=investigate-task-queue`
- `AWS_REGION={detected_region}`
- `DYNAMODB_TABLE=reposwarm-cache`
- `BEARER_TOKEN={generated_token}` (generate a random 32-char hex token)
- `AUTH_MODE=local` (tells API to use simple bearer token auth, not Cognito)

Start with `nohup npm start > {installDir}/api/api.log 2>&1 &`

### 5. Clone and start Worker
```bash
cd {installDir}
git clone https://github.com/royosherove/repo-swarm.git worker
cd worker
python3 -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt
```
Create `.env` with proper Temporal + AWS + DynamoDB config.
Start with `nohup python -m worker.main > {installDir}/worker/worker.log 2>&1 &`

### 6. Clone and start UI
```bash
cd {installDir}
git clone https://github.com/loki-bedlam/reposwarm-ui.git ui
cd ui && npm install
```
Create `.env.local` with `NEXT_PUBLIC_API_URL=http://localhost:3000`
Start with `nohup npm run dev > {installDir}/ui/ui.log 2>&1 &`

### 7. Auto-configure CLI
Write config file directly at `~/.reposwarm/config.json` with apiUrl and apiToken.

### 8. Verify
Wait for services to be healthy (with retries), then:
- Check Temporal: HTTP GET `http://localhost:7233/api/v1/namespaces`
- Check API: HTTP GET `http://localhost:3000/v1/health`
- Check UI: HTTP GET `http://localhost:3001`
- Print summary with status of each component

### 9. Print next steps
```
RepoSwarm local environment is running!

  Temporal UI:  http://localhost:8233
  API Server:   http://localhost:3000
  UI:           http://localhost:3001

  CLI configured and connected.

  Try:
    reposwarm repos add is-odd --url https://github.com/jonschlinkert/is-odd --source GitHub
    reposwarm investigate is-odd
```

## Implementation Plan

### New files:
- `internal/bootstrap/local.go` — the main `SetupLocal()` function that orchestrates everything

### Modified files:
- `internal/commands/new.go` — add `--local` flag, call `SetupLocal()` when set
- `internal/bootstrap/guide.go` — fix `temporalCompose()` to use postgres instead of sqlite

### Bug fix:
- `internal/commands/new.go` — fix `--guide-only --json` not writing files (line ~47: the JSON+guideOnly code path skips file creation)

## Code Style
- Use the existing `output.F` formatter for all user-facing output (Section, Info, Success, Warning, Error)
- For JSON mode (`--json`), return structured status of each step
- Use `exec.Command` for running external commands
- Print each step as it runs with status
- If any critical step fails, clean up and exit with clear error
- Each step should have a verification check before moving to the next

## Testing
After implementing, run:
```bash
export PATH=$PATH:/usr/local/go/bin
go build -o reposwarm ./cmd/reposwarm
go test ./... -count=1
```

## Constraints
- Go 1.24
- No new external Go dependencies (use stdlib only)
- Must work on linux/arm64 and linux/amd64
- All output through the existing formatter system
