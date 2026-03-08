# RepoSwarm CLI Usability Test — Blind Agent Mission

You are an AI coding agent on a fresh machine. You have never heard of "reposwarm" before.

## Your Mission

A colleague told you about a tool called **reposwarm** that can analyze GitHub repositories and produce architecture documentation. They gave you this install command:

```
curl -fsSL https://raw.githubusercontent.com/reposwarm/reposwarm-cli/main/install.sh | sh
```

Your goals:
1. **Install reposwarm** and figure out what it does
2. **Set up a local working instance** so you can use it
3. **Configure it to use AWS Bedrock** (this machine has an IAM role with full AWS access in us-east-1, no API keys needed)
4. **Investigate this repository:** https://github.com/jonschlinkert/is-odd — produce architecture documentation for it
5. **Find and read the results** of your investigation
6. **Clean up** when you're done

You also heard there's a way to **ask questions** about investigated repos using a companion tool. If you can figure out how to set that up, do it and ask: "What does this project do and how is the code structured?"

## Environment

- **AWS:** IAM role with full access, region us-east-1
- **Docker:** Installed and running
- **GITHUB_TOKEN:** Available in your environment variable `GITHUB_TOKEN` (for pushing results to a remote repository if the tool asks for it)
- **Results repository:** https://github.com/reposwarm/e2e-arch-hub — if the tool asks where to store investigation results, use this

## Rules

- You may ONLY use the `reposwarm` CLI (and any companion CLIs it mentions) to accomplish your mission. Do not manually edit config files, docker-compose files, or any internal tool files.
- You may NOT look inside any hidden/dot folders in the current directory or home directory that belong to the tool. Discover everything through the CLI's help, commands, and output.
- You CAN read any non-hidden files, and you CAN use standard system tools (curl, docker ps, etc.) for diagnostics.
- Do NOT look at the source code of reposwarm to figure out how to use it.
- Treat this like a real user experience — you only know what the CLI tells you.

## What We Want to Learn

This is a **usability test**, not a QA test. We care about:

1. **Discoverability:** Could you figure out what to do from `--help` and error messages alone?
2. **First-run experience:** Was the setup flow intuitive? Any dead ends?
3. **Error messages:** When something went wrong, did the error tell you how to fix it?
4. **Command naming:** Were commands named what you expected? Did you guess wrong names first?
5. **Flow completeness:** Could you go from zero to results without external documentation?
6. **Friction points:** Where did you get stuck, confused, or frustrated?
7. **Missing features:** What did you expect to exist but didn't?
8. **Happy paths:** What felt smooth and well-designed?

## Report

Write a detailed report to `./agent-feedback.md` with:

### Header
- Test date, tool version(s) discovered, agent model, environment

### Mission Outcome
- Did you complete the mission? Fully / Partially / Failed
- How long did it take (rough estimate)?
- Did you need any workarounds?

### First Impressions
- What did `--help` tell you? Was it clear?
- How did you figure out the setup flow?
- What was your first mistake or wrong guess?

### Step-by-Step Journal
For every action you took, document:
- What you were trying to do
- What command you tried (including wrong guesses)
- What happened
- Whether it was intuitive or confusing
- How you recovered from errors

### Usability Scorecard
Rate each area 1-5 (1=terrible, 5=excellent):
- **Installation:** Easy to install?
- **Discoverability:** Can you figure out commands from help alone?
- **Setup flow:** First-time setup intuitive?
- **Error messages:** Helpful when things go wrong?
- **Command naming:** Commands named what you'd expect?
- **Documentation:** Enough info in --help and output?
- **Progress feedback:** Do you know what's happening while waiting?
- **End-to-end flow:** Can you go from zero to results smoothly?

### Bug List
Any bugs found, with severity (CRITICAL/HIGH/MEDIUM/LOW)

### Usability Issues (separate from bugs)
Things that work but are confusing, unintuitive, or poorly communicated.
Rate each: FRICTION (slows you down) / BLOCKER (couldn't proceed) / ANNOYANCE (minor UX issue)

### Recommendations
Top 5 things you'd change to make the CLI more intuitive for first-time users.

### What Worked Great
Genuine positive observations — what felt well-designed?

### Final Verdict
"Could a developer with no docs complete this mission? Yes / With difficulty / No"
"Could an AI agent complete this autonomously? Yes / With workarounds / No"

When completely finished, run:
```
openclaw system event --text "Done: RepoSwarm CLI usability test complete — check agent-feedback.md" --mode now
```
