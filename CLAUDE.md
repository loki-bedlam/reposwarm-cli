# CLAUDE.md — Go Best Practices for reposwarm-cli

## Project Structure
```
cmd/reposwarm/     — main entrypoint only
internal/api/      — HTTP client for RepoSwarm API
internal/config/   — Config file management (~/.reposwarm/config.json)
internal/output/   — Output formatting (table, JSON, raw)
internal/commands/ — Cobra command definitions
docs/              — Agent-friendly markdown docs
```

## Go Conventions
- **Go 1.24+** — use latest stdlib features
- **Standard library first** — prefer net/http over third-party HTTP clients
- **Error wrapping** — use `fmt.Errorf("context: %w", err)` for all errors
- **No panic** — return errors, never panic in library code
- **Table-driven tests** — use `[]struct{ name, ... }` pattern
- **Testable design** — accept interfaces, return structs
- **Context propagation** — pass `context.Context` through API calls

## Dependencies (minimal)
- `github.com/spf13/cobra` — CLI framework
- `github.com/fatih/color` — Terminal colors
- `github.com/olekukonez/tablewriter` — Table output
- No other deps unless absolutely necessary

## Coding Rules
- Run `go vet ./...` and `go test ./...` before committing
- All exported types/functions must have doc comments
- Keep functions under 50 lines where possible
- Use `internal/` to prevent external imports
- Error messages: lowercase, no trailing punctuation, include context
- JSON struct tags: use `json:"snake_case"`
- Config keys: camelCase in JSON, flag-case on CLI

## Testing
- Test files next to source: `foo.go` → `foo_test.go`
- Use `httptest.NewServer` for API client tests
- Use `t.TempDir()` for config file tests
- Test both JSON and pretty output modes
- Aim for 80%+ coverage on internal packages

## Build
```bash
go build -o reposwarm ./cmd/reposwarm
go test ./...
go vet ./...
```
