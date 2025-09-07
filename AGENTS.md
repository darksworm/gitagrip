# Repository Guidelines

## Project Structure & Module Organization
- `main.go`: CLI/TUI entrypoint wiring services and Bubble Tea UI.
- `internal/config`: Load/save `~/.gitagrip.toml` or `<DIR>/.gitagrip.toml` via a config service.
- `internal/discovery`: Repository scanning and repo events.
- `internal/git`: Git operations and status updates.
- `internal/eventbus`: Simple pub/sub for domain events.
- `internal/groups`: Group management.
- `internal/logic`: In-memory stores and interfaces.
- `internal/ui`: Bubble Tea model, views, input modes, and commands.

## Build, Test, and Development Commands
- Build: `go build -o gitagrip .` — compiles the TUI binary.
- Run: `go run . -d <path>` — starts UI scanning `<path>` (or CWD).
- Format: `go fmt ./...` — formats code to Go standards.
- Vet: `go vet ./...` — catches common issues.
- Test: `go test ./... -race` — runs unit tests (add as you go).

## Coding Style & Naming Conventions
- Style: Go defaults (`go fmt`); idiomatic Go with tabs and 100–120 col soft limit.
- Packages: all lowercase, no underscores (`internal/ui`, `internal/git`).
- Files: lowercase, use `_test.go` for tests.
- Types/Funcs: `CamelCase`; unexport unless needed across packages.
- Errors: wrap with context; return `error` not panics in libraries.
- Events: define in `internal/eventbus`; name types with `...Event` suffix.

## Testing Guidelines
- Framework: standard `testing`; use table-driven tests.
- Location: co-locate tests per package (e.g., `internal/git/gitservice_test.go`).
- Coverage: focus on pure logic (stores, discovery filters, grouping).
- Run: `go test ./... -race -cover` before opening a PR.

## Commit & Pull Request Guidelines
- Commits: Conventional Commits (`feat:`, `fix:`, `refactor:`) as seen in history.
- Scope: small, focused; reference files or packages in the body.
- PRs: include purpose, approach, and before/after notes; link issues.
- UI changes: add a short terminal screenshot/GIF if helpful.

## Security & Configuration Tips
- Config path: app reads/writes `.gitagrip.toml` in the target dir.
- Do not commit personal configs or logs (`gitagrip.log`).
- Avoid shelling out with untrusted input in Git operations.
