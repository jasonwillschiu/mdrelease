# mdrelease

`mdrelease` is a small CLI for markdown-driven releases. It reads the latest entry from `changelog.md`, builds a commit message and annotated git tag, and can push both to your remote.

It shells out to `git` and does not require GitHub CLI (`gh`).

Initial distribution is via `go install`.

## Install

```bash
go install github.com/jasonwillschiu/mdrelease-com@latest
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
2. Validate git repo + remote
3. Fetch tags
4. Ensure the release tag does not already exist
5. `git add -A`
6. Commit using changelog summary/body
7. Create annotated tag
8. Push `HEAD`
9. Push tag

### `mdrelease check`

Validates changelog parsing and git preconditions without creating commits or tags.

### `mdrelease version`

Prints only the latest changelog version (stdout only), which is safe for scripting.

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

# Use a custom changelog file
mdrelease --changelog release-notes.md
```

## Notes / Failure Cases

- If the tag already exists, `mdrelease` fails and tells you to update your changelog version.
- Default full release fails if there are no changes to commit after staging (`git add -A`).
- Default full release also requires a configured git remote named `origin` (or use `--remote <name>`).
- `mdrelease version` prints only the version string, with errors on stderr.
