# 0.5.0 - Fix: Tighten tag checks and local release flow
- Require a git remote only for push actions so local `--commit` and `--tag` flows work without `origin`.
- Validate tag presence/absence against `refs/tags/<tag>` to avoid branch/ref name collisions.
- Validate tag ref names before checks and preserve hard errors instead of treating all failures as "tag missing".
- Add tests for local-only release flows and exact tag-ref behavior.

# 0.4.0 - Update: Rename public module path
- Rename the public Go module path to `github.com/jasonwillschiu/mdrelease`.
- Update `go install` documentation to use the renamed repository path.
- Update internal imports to match the new module path.

# 0.3.0 - Add: Top-level help/version flags
- Add root `--help` (`-h`, `-help`) support that prints usage and exits successfully.
- Add root `--version` (`-version`) alias for the existing `version` subcommand.
- Add CLI tests covering the new top-level flag behavior.

# 0.2.0 - Update: Root entrypoint + public module path
- Move the CLI entrypoint to repository-root `main.go` for simpler `go install`.
- Align the Go module path with the public GitHub repository (`mdrelease-com`).
- Update task build/install commands to target the module root package.
- Refresh install docs to use `go install github.com/jasonwillschiu/mdrelease-com@latest`.

# 0.1.1 - Fix: Satisfy errcheck for CLI output writes
- Explicitly ignore `fmt.Fprint*` return values in CLI output paths.
- Ignore deferred changelog file close errors to satisfy lint checks.
- Keep release/check/version behavior unchanged while passing lint.

# 0.1.0 - Initial release
- Add `mdrelease` CLI for markdown-driven release automation.
- Parse the latest changelog entry into a commit message and annotated git tag body.
- Validate git repository state and remote before running release steps.
- Support dry-run, check-only, and version-only commands for safer workflows.
- Allow configurable changelog path, remote name, and tag prefix via flags.
