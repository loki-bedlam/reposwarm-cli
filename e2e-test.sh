#!/bin/bash
# RepoSwarm + Ask CLI — End-to-End Test Script
#
# Runs the full write/read pipeline on a clean instance:
#   RepoSwarm: install → provider → investigate → results
#   Ask CLI:   setup → askbox up → query → results browse → lifecycle
#
# Requirements:
#   - reposwarm and ask binaries on PATH
#   - Docker + Docker Compose
#   - IAM role with bedrock:InvokeModel (for Bedrock provider)
#   - GITHUB_TOKEN env var (for private arch-hub repos)
#
# Usage:
#   export GITHUB_TOKEN="ghp_..."
#   ./e2e-test.sh [--arch-hub URL] [--repo NAME] [--repo-url URL] [--provider PROVIDER] [--region REGION]
#
# All parameters have sensible defaults for the reposwarm/e2e-arch-hub test repo.

set -o pipefail

# ── Defaults ──
ARCH_HUB_URL="${E2E_ARCH_HUB_URL:-https://github.com/reposwarm/e2e-arch-hub.git}"
ARCH_HUB_ORG="${E2E_ARCH_HUB_ORG:-reposwarm}"
ARCH_HUB_REPO="${E2E_ARCH_HUB_REPO:-e2e-arch-hub}"
REPO_NAME="${E2E_REPO_NAME:-is-odd}"
REPO_URL="${E2E_REPO_URL:-https://github.com/jonschlinkert/is-odd}"
REPO_SOURCE="${E2E_REPO_SOURCE:-GitHub}"
PROVIDER="${E2E_PROVIDER:-bedrock}"
REGION="${E2E_REGION:-us-east-1}"
AUTH_METHOD="${E2E_AUTH_METHOD:-iam-role}"
MODEL="${E2E_MODEL:-sonnet}"

# ── Parse CLI args ──
while [[ $# -gt 0 ]]; do
  case $1 in
    --arch-hub)     ARCH_HUB_URL="$2"; shift 2 ;;
    --arch-hub-org) ARCH_HUB_ORG="$2"; shift 2 ;;
    --arch-hub-repo) ARCH_HUB_REPO="$2"; shift 2 ;;
    --repo)         REPO_NAME="$2"; shift 2 ;;
    --repo-url)     REPO_URL="$2"; shift 2 ;;
    --provider)     PROVIDER="$2"; shift 2 ;;
    --region)       REGION="$2"; shift 2 ;;
    --auth)         AUTH_METHOD="$2"; shift 2 ;;
    --model)        MODEL="$2"; shift 2 ;;
    --help|-h)
      echo "Usage: $0 [--arch-hub URL] [--repo NAME] [--repo-url URL] [--provider PROVIDER] [--region REGION]"
      echo "  Environment: GITHUB_TOKEN (required for private repos)"
      exit 0 ;;
    *) echo "Unknown arg: $1"; exit 1 ;;
  esac
done

# ── Validate ──
if [ -z "$GITHUB_TOKEN" ]; then
  echo "ERROR: GITHUB_TOKEN env var is required"
  exit 1
fi

BUGS=0
LOG="/tmp/e2e-test-$(date -u +%Y%m%d-%H%M%S).log"
> "$LOG"

log()  { echo "$@" | tee -a "$LOG"; }
bug()  { log "  ❌ BUG: $1"; BUGS=$((BUGS+1)); }
pass() { log "  ✅ $1"; }

log ""
log "╔══════════════════════════════════════════════════╗"
log "║  RepoSwarm + Ask CLI — E2E Test                  ║"
log "╚══════════════════════════════════════════════════╝"
log ""
log "  Provider:  $PROVIDER ($REGION, $AUTH_METHOD)"
log "  Model:     $MODEL"
log "  Repo:      $REPO_NAME ($REPO_URL)"
log "  Arch-hub:  $ARCH_HUB_URL"
log "  Log:       $LOG"
log ""

# ── STEP 1: RepoSwarm local setup ──
log "━━━ STEP 1: reposwarm new --local ━━━"
OUTPUT=$(reposwarm new --local --for-agent --force 2>&1)
RC=$?
echo "$OUTPUT" >> "$LOG"
echo "$OUTPUT" | tail -5
if [ $RC -eq 0 ]; then pass "STEP 1: Setup complete"; else bug "STEP 1: Setup failed (rc=$RC)"; fi

# ── STEP 2: Provider setup ──
log ""
log "━━━ STEP 2: Provider setup ━━━"
OUTPUT=$(reposwarm config provider setup \
  --for-agent --non-interactive \
  --provider "$PROVIDER" --region "$REGION" \
  --auth-method "$AUTH_METHOD" --model "$MODEL" --pin 2>&1)
RC=$?
echo "$OUTPUT" >> "$LOG"
echo "$OUTPUT" | grep -E 'Model:|✓|ERROR|WARNING'

MODEL_VALUE=$(grep '^ANTHROPIC_MODEL=' ~/.reposwarm/temporal/worker.env | cut -d= -f2)
log "  Model in worker.env: $MODEL_VALUE"
# Model should NOT be a bare alias (sonnet/opus/haiku) — must be resolved
if echo "$MODEL_VALUE" | grep -qE '^(sonnet|opus|haiku)$'; then
  bug "STEP 2: Model is unresolved alias: $MODEL_VALUE"
elif [ -n "$MODEL_VALUE" ]; then
  pass "STEP 2: Model resolved ($MODEL_VALUE)"
else
  bug "STEP 2: No ANTHROPIC_MODEL in worker.env"
fi

# ── STEP 3: Arch-hub + GitHub token ──
log ""
log "━━━ STEP 3: Arch-hub + GitHub token ━━━"
reposwarm config set archHubUrl "$ARCH_HUB_URL" --for-agent >> "$LOG" 2>&1

echo "GITHUB_TOKEN=$GITHUB_TOKEN" >> ~/.reposwarm/temporal/worker.env
echo "ARCH_HUB_BASE_URL=https://github.com/$ARCH_HUB_ORG" >> ~/.reposwarm/temporal/worker.env
echo "ARCH_HUB_REPO_NAME=$ARCH_HUB_REPO" >> ~/.reposwarm/temporal/worker.env

cd ~/.reposwarm/temporal
docker compose stop worker >> "$LOG" 2>&1
docker compose rm -f worker >> "$LOG" 2>&1
docker compose up -d worker >> "$LOG" 2>&1
sleep 8

CONTAINER_TOKEN=$(docker exec reposwarm-worker env 2>&1 | grep '^GITHUB_TOKEN=' | cut -d= -f2 | head -c 4)
ARCH_BASE=$(docker exec reposwarm-worker env 2>&1 | grep '^ARCH_HUB_BASE_URL=' | cut -d= -f2)
log "  Container GITHUB_TOKEN prefix: ${CONTAINER_TOKEN}..."
log "  Container ARCH_HUB_BASE_URL: $ARCH_BASE"

if [ -n "$CONTAINER_TOKEN" ]; then
  pass "STEP 3a: GITHUB_TOKEN in container"
else
  bug "STEP 3a: GITHUB_TOKEN missing"
fi
if [ -n "$ARCH_BASE" ]; then
  pass "STEP 3b: ARCH_HUB_BASE_URL set"
else
  bug "STEP 3b: ARCH_HUB_BASE_URL missing"
fi

# ── STEP 4: Add repo + investigate ──
log ""
log "━━━ STEP 4: Add repo + investigate ━━━"
cd ~
OUTPUT=$(reposwarm repos add "$REPO_NAME" --url "$REPO_URL" --source "$REPO_SOURCE" --for-agent 2>&1)
echo "$OUTPUT" >> "$LOG"
echo "$OUTPUT"

OUTPUT=$(reposwarm investigate "$REPO_NAME" --for-agent 2>&1)
echo "$OUTPUT" >> "$LOG"
echo "$OUTPUT"

FINAL_STATUS="timeout"
for i in $(seq 1 60); do
  sleep 10
  WF_JSON=$(reposwarm wf list --for-agent --json 2>&1)
  WF_RC=$?

  WF_STATUS=$(echo "$WF_JSON" | python3 -c "
import json, sys
try:
    data = json.load(sys.stdin)
    latest = [w for w in data if '$REPO_NAME' in w.get('workflowId', '')]
    latest.sort(key=lambda x: x.get('startTime', ''), reverse=True)
    print(latest[0]['status'] if latest else 'none')
except Exception as e:
    print('parse_error', file=sys.stderr)
    print('parse_error')
" 2>>"$LOG")

  log "  [$(date -u +%H:%M:%S)] wf_rc=$WF_RC status=$WF_STATUS"

  if [ "$WF_STATUS" = "Completed" ]; then FINAL_STATUS="Completed"; break; fi
  if [ "$WF_STATUS" = "Failed" ] || [ "$WF_STATUS" = "Terminated" ]; then
    FINAL_STATUS="$WF_STATUS"
    docker logs reposwarm-worker --tail 10 >> "$LOG" 2>&1
    break
  fi
  if [ "$WF_STATUS" = "parse_error" ]; then
    log "  [WARN] JSON parse failed, raw: $WF_JSON"
  fi
done

if [ "$FINAL_STATUS" = "Completed" ]; then
  pass "STEP 4: Investigation completed"
else
  bug "STEP 4: Investigation $FINAL_STATUS"
fi

# ── STEP 5: Results ──
log ""
log "━━━ STEP 5: reposwarm results list ━━━"
RESULTS_TEXT=$(reposwarm results list --for-agent 2>&1)
RESULTS_RC=$?
echo "$RESULTS_TEXT" >> "$LOG"
echo "$RESULTS_TEXT"

RESULTS_JSON=$(reposwarm results list --for-agent --json 2>&1)
RESULTS_JSON_RC=$?
echo "$RESULTS_JSON" >> "$LOG"

SECTION_COUNT=$(echo "$RESULTS_JSON" | python3 -c '
import json, sys
try:
    data = json.load(sys.stdin)
    if data and len(data) > 0:
        print(data[0].get("sectionCount", data[0].get("sections", 0)))
    else:
        print(0)
except Exception as e:
    print(0)
    print(f"JSON parse error: {e}", file=sys.stderr)
' 2>>"$LOG")

log "  results rc=$RESULTS_RC, json rc=$RESULTS_JSON_RC, sections=$SECTION_COUNT"

if [ "$RESULTS_RC" -eq 0 ] && [ "$SECTION_COUNT" -gt 0 ] 2>/dev/null; then
  pass "STEP 5: $SECTION_COUNT sections found"
else
  bug "STEP 5: Results failed (rc=$RESULTS_RC, sections=$SECTION_COUNT)"
fi

# ── STEP 6: Ask setup ──
log ""
log "━━━ STEP 6: ask setup ━━━"
OUTPUT=$(ask setup --for-agent \
  --provider "$PROVIDER" --region "$REGION" \
  --auth "$AUTH_METHOD" --model "$MODEL" \
  --arch-hub "$ARCH_HUB_URL" \
  --skip-docker 2>&1)
SETUP_RC=$?
echo "$OUTPUT" >> "$LOG"
echo "$OUTPUT"
log "  ask setup rc=$SETUP_RC"

if [ $SETUP_RC -ne 0 ]; then
  bug "STEP 6: ask setup failed (rc=$SETUP_RC)"
else
  log "  --- askbox.env ---"
  cat ~/.ask/askbox.env >> "$LOG" 2>&1

  ASK_MODEL=$(grep '^ANTHROPIC_MODEL=' ~/.ask/askbox.env 2>/dev/null | cut -d= -f2)
  ASK_TOKEN=$(grep '^GITHUB_TOKEN=' ~/.ask/askbox.env 2>/dev/null | cut -d= -f2 | head -c 4)
  ASK_ARCH=$(grep '^ARCH_HUB_URL=' ~/.ask/askbox.env 2>/dev/null | cut -d= -f2)
  COMPOSE_MOUNT=$(grep 'arch-hub' ~/.ask/docker-compose.yml 2>/dev/null | grep '/home/')

  log "  Model: $ASK_MODEL"
  log "  Token prefix: ${ASK_TOKEN}..."
  log "  Arch-hub URL: $ASK_ARCH"
  log "  Compose mount: $COMPOSE_MOUNT"

  if echo "$ASK_MODEL" | grep -qE '^(sonnet|opus|haiku)$'; then
    bug "STEP 6a: Model is unresolved alias ($ASK_MODEL)"
  elif [ -n "$ASK_MODEL" ]; then
    pass "STEP 6a: Model resolved"
  else
    bug "STEP 6a: No ANTHROPIC_MODEL in askbox.env"
  fi
  if [ -n "$ASK_TOKEN" ]; then pass "STEP 6b: GITHUB_TOKEN present"; else bug "STEP 6b: GITHUB_TOKEN missing"; fi
  if [ -n "$ASK_ARCH" ]; then pass "STEP 6c: ARCH_HUB_URL set"; else bug "STEP 6c: ARCH_HUB_URL missing"; fi
  if [ -n "$COMPOSE_MOUNT" ]; then pass "STEP 6d: Bind mount in compose"; else bug "STEP 6d: No bind mount"; fi
fi

# ── STEP 7: Askbox up ──
log ""
log "━━━ STEP 7: ask up + status ━━━"
OUTPUT=$(ask up --for-agent 2>&1)
UP_RC=$?
echo "$OUTPUT" >> "$LOG"
echo "$OUTPUT"
log "  ask up rc=$UP_RC"

log "  Waiting 15s for askbox to clone arch-hub..."
sleep 15

STATUS_OUT=$(ask status --for-agent 2>&1)
STATUS_RC=$?
echo "$STATUS_OUT" >> "$LOG"
echo "$STATUS_OUT"
log "  ask status rc=$STATUS_RC"

HEALTH_JSON=$(curl -s http://localhost:8082/health 2>&1)
echo "$HEALTH_JSON" >> "$LOG"

ARCH_READY=$(echo "$HEALTH_JSON" | python3 -c '
import json, sys
try:
    d = json.load(sys.stdin)
    print(d.get("arch_hub_ready", False))
except:
    print("parse_error")
' 2>>"$LOG")
ARCH_REPOS=$(echo "$HEALTH_JSON" | python3 -c '
import json, sys
try:
    d = json.load(sys.stdin)
    print(d.get("arch_hub_repos", 0))
except:
    print(0)
' 2>>"$LOG")

log "  arch_hub_ready=$ARCH_READY, repos=$ARCH_REPOS"

if [ "$UP_RC" -eq 0 ] && [ "$STATUS_RC" -eq 0 ] && [ "$ARCH_READY" = "True" ]; then
  pass "STEP 7a: Askbox healthy, arch-hub loaded ($ARCH_REPOS repos)"
else
  bug "STEP 7a: up=$UP_RC status=$STATUS_RC arch_ready=$ARCH_READY"
  docker logs askbox 2>&1 | head -15
fi

HOST_FILES=$(ls ~/.ask/arch-hub/*.arch.md 2>/dev/null | wc -l)
log "  Host arch-hub files: $HOST_FILES"
ls -la ~/.ask/arch-hub/ >> "$LOG" 2>&1

if [ "$HOST_FILES" -gt 0 ]; then
  pass "STEP 7b: $HOST_FILES arch files on host"
else
  bug "STEP 7b: No arch files on host"
fi

# ── STEP 8: Ask question ──
log ""
log "━━━ STEP 8: ask question ━━━"
ASK_OUT=$(ask --for-agent "What does $REPO_NAME do? One paragraph." 2>&1)
ASK_RC=$?
echo "$ASK_OUT" >> "$LOG"
echo "$ASK_OUT" | head -10
log "  ask rc=$ASK_RC"

if [ $ASK_RC -eq 0 ]; then
  pass "STEP 8: Question answered"
else
  bug "STEP 8: Ask failed (rc=$ASK_RC)"
fi

# ── STEP 9: Results browse ──
log ""
log "━━━ STEP 9: ask results ━━━"

log "  --- results list ---"
LIST_OUT=$(ask results list --path ~/.ask/arch-hub --for-agent 2>&1)
LIST_RC=$?
echo "$LIST_OUT" >> "$LOG"
echo "$LIST_OUT"
log "  list rc=$LIST_RC"

log "  --- results read ---"
READ_OUT=$(ask results read "$REPO_NAME" --path ~/.ask/arch-hub --for-agent 2>&1)
READ_RC=$?
echo "$READ_OUT" >> "$LOG"
echo "$READ_OUT" | head -5
log "  read rc=$READ_RC"

log "  --- results search ---"
SEARCH_OUT=$(ask results search 'dependencies' --path ~/.ask/arch-hub --for-agent --max 3 2>&1)
SEARCH_RC=$?
echo "$SEARCH_OUT" >> "$LOG"
echo "$SEARCH_OUT"
log "  search rc=$SEARCH_RC"

log "  --- results export ---"
EXPORT_OUT=$(ask results export "$REPO_NAME" --path ~/.ask/arch-hub -o /tmp/export.md --for-agent 2>&1)
EXPORT_RC=$?
echo "$EXPORT_OUT" >> "$LOG"
echo "$EXPORT_OUT"
log "  export rc=$EXPORT_RC"

if [ $LIST_RC -eq 0 ] && [ $READ_RC -eq 0 ] && [ $SEARCH_RC -eq 0 ] && [ $EXPORT_RC -eq 0 ]; then
  EXPORT_LINES=$(wc -l < /tmp/export.md 2>/dev/null || echo 0)
  pass "STEP 9: All results commands work (export=$EXPORT_LINES lines)"
else
  bug "STEP 9: list=$LIST_RC read=$READ_RC search=$SEARCH_RC export=$EXPORT_RC"
fi

# ── STEP 10: Docker lifecycle ──
log ""
log "━━━ STEP 10: Docker lifecycle ━━━"

log "  --- ask down ---"
DOWN_OUT=$(ask down --for-agent 2>&1)
DOWN_RC=$?
echo "$DOWN_OUT" >> "$LOG"
echo "$DOWN_OUT"
log "  down rc=$DOWN_RC"

log "  --- ask status (expect failure) ---"
STATUS_DOWN_OUT=$(ask status --for-agent 2>&1)
STATUS_DOWN_RC=$?
echo "$STATUS_DOWN_OUT" >> "$LOG"
log "  status-after-down rc=$STATUS_DOWN_RC (expected non-zero)"

log "  --- ask up ---"
UP2_OUT=$(ask up --for-agent 2>&1)
UP2_RC=$?
echo "$UP2_OUT" >> "$LOG"
echo "$UP2_OUT"
log "  up rc=$UP2_RC"

sleep 12

log "  --- ask status (expect success) ---"
STATUS_UP_OUT=$(ask status --for-agent 2>&1)
STATUS_UP_RC=$?
echo "$STATUS_UP_OUT" >> "$LOG"
echo "$STATUS_UP_OUT"
log "  status-after-up rc=$STATUS_UP_RC"

if [ $DOWN_RC -eq 0 ] && [ $STATUS_DOWN_RC -ne 0 ] && [ $UP2_RC -eq 0 ] && [ $STATUS_UP_RC -eq 0 ]; then
  pass "STEP 10: Docker lifecycle (down=$DOWN_RC, unreachable=$STATUS_DOWN_RC, up=$UP2_RC, healthy=$STATUS_UP_RC)"
else
  bug "STEP 10: down=$DOWN_RC status_down=$STATUS_DOWN_RC up=$UP2_RC status_up=$STATUS_UP_RC"
fi

# ── SUMMARY ──
log ""
log "╔══════════════════════════════════════════════════╗"
if [ $BUGS -eq 0 ]; then
  log "║  ✅ PASSED: 0 bugs found                         ║"
else
  log "║  ❌ FAILED: $BUGS bug(s) found                     ║"
fi
log "╚══════════════════════════════════════════════════╝"
log ""
log "Full log: $LOG"

exit $BUGS
