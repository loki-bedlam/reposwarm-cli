You are an AI coding agent on a fresh machine with zero prior context. You need to test a CLI tool called 'reposwarm'.

SETUP:
- Install from: curl -fsSL https://raw.githubusercontent.com/reposwarm/reposwarm-cli/main/install.sh | sh
- Ask CLI install: curl -fsSL https://raw.githubusercontent.com/reposwarm/ask-cli/main/install.sh | sh
- This machine has an IAM role with full AWS access (us-east-1)
- Use AWS Bedrock with IAM role auth — do NOT use API keys
- GITHUB_TOKEN is available in environment for arch-hub push
- Arch-hub repo: https://github.com/reposwarm/e2e-arch-hub

EXPECTED VERSIONS (verify each — STOP and report if mismatch):
- reposwarm CLI: 1.3.156 (verify with `reposwarm version --for-agent`)
- Ask CLI: 0.2.0 (verify with `ask version`)
- API: check with `reposwarm status --for-agent` or docker image tag
- Worker: check docker image tag or worker logs on startup
- UI: check docker image tag

CRITICAL RULES:
- ALWAYS add --for-agent to EVERY reposwarm command (machine-readable output)
- Read --help output carefully before running any command
- Do NOT skip steps or assume anything — discover everything from help text
- Do NOT ask questions — figure everything out from help and error messages
- Use model alias "sonnet" when configuring provider
- If docker containers need env changes, use `reposwarm config worker-env set KEY VALUE --for-agent` then `reposwarm restart worker --for-agent`
- When waiting for an investigation, poll every 30 seconds with `reposwarm workflows progress --for-agent`
- Do NOT run `docker compose` commands directly — use `reposwarm` CLI commands for all operations
- After configuring the provider, also configure the arch-hub: set ARCH_HUB_BASE_URL and GITHUB_TOKEN in worker env

YOUR TASK:

Scenario 1: Install & Setup
1. Install the reposwarm CLI using the install script
2. Verify version matches expected (1.3.156)
3. Run `reposwarm --help` and explore all subcommands
4. Run `reposwarm new --local --for-agent` to create a local instance
5. Wait for all containers to be healthy (check with `reposwarm status --for-agent`)
6. Record actual versions of all components (CLI, API, Worker, UI from docker images/status)

Scenario 2: Provider & Arch-Hub Configuration
7. Run `reposwarm config provider setup` to configure: AWS Bedrock, IAM role auth, us-east-1, model sonnet (check --help first for exact flags)
8. Set arch-hub: `reposwarm config worker-env set ARCH_HUB_BASE_URL https://github.com/reposwarm/e2e-arch-hub --for-agent`
9. Set GitHub token: `reposwarm config worker-env set GITHUB_TOKEN '$GITHUB_TOKEN' --for-agent` (use the actual env var value)
10. Restart worker: `reposwarm restart worker --for-agent`
11. Run `reposwarm doctor --for-agent` to verify setup

Scenario 3: Investigation (single repo)
12. Run `reposwarm repos add https://github.com/jonschlinkert/is-odd --for-agent` (accepts URL directly now)
13. Run `reposwarm investigate https://github.com/jonschlinkert/is-odd --for-agent`
14. Monitor with `reposwarm workflows progress --for-agent` every 30 seconds until complete
15. Verify progress counter updates (should show N/17 steps, not stuck at 0)
16. Check results: `reposwarm workflows list --for-agent` and `reposwarm workers list --for-agent`
17. Verify arch-hub was updated (check worker logs for arch-hub push confirmation)

Scenario 4: Ask CLI
18. Install the ask CLI: curl -fsSL https://raw.githubusercontent.com/reposwarm/ask-cli/main/install.sh | sh
19. Verify ask version (should be 0.2.0) — test both `ask version` and `ask --version`
20. Run `ask setup` to configure (use defaults, point arch-hub to https://github.com/reposwarm/e2e-arch-hub)
21. Run `ask up` to start the askbox
22. Ask a question: `ask "What does is-odd do and how does it work?"`
23. Check results: `ask list`

Scenario 5: investigate --all (de-duplication)
24. Add a second repo: `reposwarm repos add https://github.com/jonschlinkert/is-even --for-agent`
25. Run `reposwarm repos list --for-agent` to verify both repos
26. Run `reposwarm investigate --all --for-agent` — verify it SKIPS is-odd (already investigated)
27. Monitor is-even investigation until complete

Scenario 6: Workflow Status & Error Handling
28. Run `reposwarm workflows status <workflow-id> --for-agent` with a real workflow ID from the list — verify it returns details (not 404)
29. Check for any stale workflows or errors

Scenario 7: Diagnostics & Cleanup
30. Run `reposwarm doctor --for-agent` — check all health indicators
31. Run `reposwarm logs worker --for-agent --tail 30`
32. Run `reposwarm stop --for-agent` (should stop ALL services without requiring a service name)
33. Verify all containers stopped

REPORT FORMAT:
Write a structured report to ./agent-feedback.md containing:

## Header
- Test date, all component versions (expected vs actual), agent model, environment

## Executive Summary
- Overall score (1-10), pass/fail verdict

## Scenario Results
For each scenario: commands run, output, pass/fail, workarounds if any

## Bug List
Severity (CRITICAL/HIGH/MEDIUM/LOW), description, repro steps, workaround

## What Worked Well
Positive UX observations

## Performance Notes
Install times, investigation duration, API response times

## Security Observations

## Final Verdict
"Could an agent complete this without manual fixes? Yes/No" with specific blockers if No

When completely finished, run this command to notify me:
openclaw system event --text "Done: RepoSwarm E2E retest complete — check /tmp/cross-agent-test/agent-feedback.md" --mode now
