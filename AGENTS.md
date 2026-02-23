# Repository Guidelines

## Project Structure & Module Organization

This repository is a small Go CLI (`mdrelease`) for changelog-driven releases.

- `main.go`: CLI entrypoint.
- `internal/app/`: command parsing and release/check/version flows.
- `internal/changelog/`: changelog parsing logic and tests.
- `internal/gitutil/`: git shelling helpers and git-related errors.
- `docs/`: prompt/planning notes (not runtime code).
- `changelog.md`: default input file parsed by the CLI.
- `Taskfile.yml`: common development tasks.

Keep new code in `internal/` packages unless it must be part of the executable entrypoint.

- Keep root CLI aliases `--help` and `--version` aligned with root usage output and `version` subcommand behavior.

## Build, Test, and Development Commands

Use `task` for the standard workflow:

- `task build`: builds `bin/mdrelease`.
- `task test`: runs `go test -v ./...`.
- `task fmt`: runs `go fmt ./...`.
- `task vet`: runs `go vet ./...`.
- `task lint`: runs `golangci-lint` and `gopls` checks (best-effort).
- `task check`: builds and runs `./bin/mdrelease check`.
- `task release-dry-run`: prints the full release flow without mutating git state.

Direct Go equivalents are also valid (for example, `go build .`).

## Coding Style & Naming Conventions

Follow standard Go conventions and let `gofmt` define formatting (tabs, imports, spacing). Use short, lowercase package names (`app`, `changelog`, `gitutil`) and descriptive exported identifiers (`ParseLatest`, `EnsureRepo`).

Prefer:

- table-free, focused unit tests for small parser/CLI behaviors
- explicit error messages with context
- `*_test.go` files colocated with the package under test

## Testing Guidelines

Tests use Go’s built-in `testing` package. Name tests `TestXxx` and keep them deterministic (use `t.TempDir()` for file fixtures, as in `internal/changelog/changelog_test.go`).

Run all tests with `task test` or `go test ./...`. No coverage threshold is enforced currently, but changes should include tests for parsing, CLI flags, and git preflight behavior when applicable.

## Commit & Pull Request Guidelines

Git history is minimal (`Initial commit`), so follow a simple convention: short imperative subject lines (for example, `Add dry-run release task`).

For pull requests, include:

- what changed and why
- user-visible CLI behavior changes (flags/output)
- test coverage notes (`task test`, `task check`)
- sample command/output when changing release flow behavior
