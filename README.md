# mdrelease

`mdrelease` is a small CLI for markdown-driven releases. It reads the latest entry from `changelog.md`, builds a commit message and annotated git tag, and can push both to your remote.

It shells out to `git` and does not require GitHub CLI (`gh`).

Initial distribution is via `go install`.

## Install

```bash
go install github.com/jasonwillschiu/mdrelease@v0.8.0
```

## Supported Changelog Format (v1)

`mdrelease` currently supports one format only:

```md
# 1.2.3 - Release title
- First change
- Second change

# 1.2.2 - Previous release
- Previous change
```

- The latest release is the first matching `# <version> - <summary>` heading.
- Only top-level `- bullet` lines under that heading are included in the commit/tag body.

## Commands

### `mdrelease`

Runs the full release flow by default (equivalent to `mdrelease --all`):

1. Parse latest changelog entry
2. Validate git repo + remote (remote required for push steps)
3. Fetch remote refs and tags
4. Pull latest commits with `--ff-only` (fails fast if not a fast-forward)
5. Ensure the release tag does not already exist
6. `git add -A`
7. Commit using changelog summary/body
8. Create annotated tag
9. Push `HEAD`
10. Push tag

### `mdrelease check`

Validates changelog parsing and git preconditions without creating commits or tags.

### `mdrelease version`

Prints latest changelog version as:

- `<latest-changelog-version>` (for example, `5.7.0`)

## Global Convenience Flags

These work at the top level (without a subcommand):

- `--help` (also `-h`, `-help`) prints root usage and exits successfully
- `--version` (also `-version`) prints the installed `mdrelease` CLI version (`mdrelease version vX.Y.Z`)
- Root help output includes the installed `mdrelease` version and documents both version modes (`mdrelease --version` vs `mdrelease version`)

## Common Flags

- `--changelog` path to changelog file (default `changelog.md`)
- `--remote` git remote name (default `origin`)
- `--tag-prefix` tag prefix (default `v`)
- `--dry-run` print planned actions without mutating git state

Environment variable:

- `MDRELEASE_CHANGELOG` (used when `--changelog` is not provided)

Precedence: `--changelog` > `MDRELEASE_CHANGELOG` > `changelog.md`

## Release Action Flags

Use these to customize the release flow instead of the default full release:

- `--all` full release pipeline (same as default `mdrelease`)
- `--stage-all`
- `--commit`
- `--tag`
- `--push-commit`
- `--push-tag`
- `--push` alias for `--push-commit --push-tag`
- `--force-retag` overwrite an existing release tag by deleting and recreating it (local and remote when pushing tags)

Examples:

```bash
# Default full release
mdrelease

# Explicit full release
mdrelease --all

# Commit, tag, and push both commit and tag
mdrelease --commit --tag --push

# Tag-only flow (no commit)
mdrelease --tag --push-tag

# Force overwrite an existing release tag (delete/recreate + push)
mdrelease --tag --push-tag --force-retag

# Use a custom changelog file
mdrelease --changelog release-notes.md

# Print root usage
mdrelease --help

# Print mdrelease CLI version
mdrelease --version

# Print changelog version as plain semver
mdrelease version
```

## Notes / Failure Cases

- If the tag already exists, `mdrelease` fails and tells you to update your changelog version.
- Local-only flows (for example `--commit` or `--tag`) do not require a configured remote.
- Push flows fetch remote refs/tags and run `git pull --ff-only` before any push step.
- `--tag` without `--push-tag` checks local tag availability only.
- `--force-retag` allows reusing an existing version tag by deleting prior local/remote tags as needed before push.
- Default full release fails if there are no changes to commit after staging (`git add -A`).
- Default full release also requires a configured git remote named `origin` (or use `--remote <name>`).
- `mdrelease version` prints `<latest-changelog-version>`, with errors on stderr.
- `mdrelease --version` prints the mdrelease CLI version string.
