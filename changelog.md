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
