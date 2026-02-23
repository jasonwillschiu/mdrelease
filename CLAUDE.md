# CLAUDE.md

## Must Follow
- Use `task` commands for standard workflows before inventing custom scripts.
- Keep executable entrypoint in the repository root (`main.go`) unless adding a new binary.
- Keep runtime code in `internal/` packages; avoid exporting internals unnecessarily.
- Update `README.md`, `AGENTS.md`, and `changelog.md` when CLI install/build behavior changes.

## Essential Commands
- `task build` builds `bin/mdrelease`
- `task test` runs `go test -v ./...`
- `task fmt` runs `go fmt ./...`
- `task vet` runs `go vet ./...`
- `task check` runs preflight checks via `./bin/mdrelease check`
- `task release-dry-run` prints the full release flow without mutating git state

## Quick Facts
- Module path: `github.com/jasonwillschiu/mdrelease-com`
- Binary: `mdrelease`
- Entrypoint: `main.go`
- Install: `go install github.com/jasonwillschiu/mdrelease-com@latest`

## Hard Invariants
- `changelog.md` newest entry must be first and match `# <version> - <summary>`.
- Release/check/version flows are orchestrated in `internal/app`.
- Git interactions go through `internal/gitutil` helpers.
- Changelog parsing rules live in `internal/changelog`; keep parser behavior covered by tests.

## Project Structure
- `main.go`: CLI entrypoint
- `internal/app/`: command parsing and release/check/version flows
- `internal/changelog/`: changelog parsing and tests
- `internal/gitutil/`: git shell helpers and git-related errors
- `docs/`: planning/prompt notes (not runtime code)
- `Taskfile.yml`: common development tasks

## Key Paths
- CLI wiring: `main.go`, `internal/app/app.go`
- Changelog parser: `internal/changelog/changelog.go`
- Git shell helpers: `internal/gitutil/gitutil.go`
- User docs: `README.md`
- Agent constraints: `AGENTS.md`
- Release history: `changelog.md`

## Reference Docs
- Read `README.md` for install, flags, and release-flow behavior.
- Read `AGENTS.md` for repo-specific coding/testing constraints used by agents.
