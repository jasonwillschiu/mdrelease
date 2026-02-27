package app

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jasonwillschiu/mdrelease/internal/changelog"
	"github.com/jasonwillschiu/mdrelease/internal/gitutil"
)

const (
	ExitOK        = 0
	ExitGeneral   = 1
	ExitUsage     = 2
	ExitParse     = 3
	ExitPreflight = 4
	ExitGit       = 5

	toolName = "mdrelease"
)

var ToolVersion = "v0.0.0"

type gitOps interface {
	EnsureRepo() error
	EnsureRemote(string) error
	FetchTags() error
	FetchRemote(string) error
	PullFFOnly(string) error
	EnsureTagAbsent(string) error
	EnsureTagPresent(string) error
	HasLocalTag(string) (bool, error)
	HasRemoteTag(string, string) (bool, error)
	DeleteLocalTag(string) error
	DeleteRemoteTag(string, string) error
	StageAll() error
	HasStagedChanges() (bool, error)
	Commit(string, string) error
	CreateTag(string, string, string) error
	PushHead(string) error
	PushTag(string, string) error
}

type deps struct {
	getenv func(string) string
	getwd  func() (string, error)
	newGit func(io.Writer, io.Writer, bool) gitOps
}

type usageError struct{ msg string }

func (e *usageError) Error() string { return e.msg }

type preflightError struct{ msg string }

func (e *preflightError) Error() string { return e.msg }

func Run(args []string, stdout, stderr io.Writer) int {
	d := deps{
		getenv: os.Getenv,
		getwd:  os.Getwd,
		newGit: func(out, errOut io.Writer, dryRun bool) gitOps {
			return gitutil.NewClient(out, errOut, dryRun)
		},
	}

	if err := run(args, stdout, stderr, d); err != nil {
		if _, isUsage := err.(*usageError); isUsage {
			_, _ = fmt.Fprintln(stderr, err.Error())
			_, _ = fmt.Fprintln(stderr)
			printRootUsage(stderr)
			return ExitUsage
		}

		switch {
		case errors.As(err, new(*changelog.ParseError)):
			_, _ = fmt.Fprintln(stderr, "Error:", err)
			if pe := new(changelog.ParseError); errors.As(err, &pe) {
				_, _ = fmt.Fprintf(stderr, "Expected format example in %s: %s\n", pe.Path, changelog.ExpectedFormat)
			}
			return ExitParse
		case errors.As(err, new(*preflightError)):
			_, _ = fmt.Fprintln(stderr, "Error:", err)
			return ExitPreflight
		case errors.As(err, new(*gitutil.GitError)):
			_, _ = fmt.Fprintln(stderr, "Error:", err)
			return ExitGit
		default:
			_, _ = fmt.Fprintln(stderr, "Error:", err)
			return ExitGeneral
		}
	}
	return ExitOK
}

func run(args []string, stdout, stderr io.Writer, d deps) error {
	if len(args) > 0 {
		switch args[0] {
		case "-h", "-help", "--help":
			printRootUsage(stdout)
			return nil
		case "-version", "--version":
			return runToolVersion(args[1:], stdout, stderr)
		}
	}

	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		switch args[0] {
		case "version":
			return runRepoVersion(args[1:], stdout, stderr, d)
		case "check":
			return runCheck(args[1:], stdout, stderr, d)
		default:
			return &usageError{msg: fmt.Sprintf("unknown command: %s", args[0])}
		}
	}
	return runRelease(args, stdout, stderr, d)
}

type commonConfig struct {
	changelogPath string
	remote        string
	tagPrefix     string
	dryRun        bool
}

type releaseActions struct {
	stageAll   bool
	commit     bool
	tag        bool
	pushCommit bool
	pushTag    bool
}

func runToolVersion(args []string, stdout, stderr io.Writer) error {
	if len(args) != 0 {
		return &usageError{msg: "--version does not accept additional arguments"}
	}

	_, _ = fmt.Fprintf(stdout, "%s version %s\n", toolName, ToolVersion)
	return nil
}

func runRepoVersion(args []string, stdout, stderr io.Writer, d deps) error {
	fs := flag.NewFlagSet("mdrelease version", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var changelogFlag string
	fs.StringVar(&changelogFlag, "changelog", "", "Path to changelog file (default: changelog.md)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return &usageError{msg: err.Error()}
	}
	if fs.NArg() != 0 {
		return &usageError{msg: "version does not accept positional arguments"}
	}

	path := resolveChangelogPath(changelogFlag, d.getenv)
	entry, err := changelog.ParseLatest(path)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "%s\n", entry.Version)
	return nil
}

func runCheck(args []string, stdout, stderr io.Writer, d deps) error {
	fs := flag.NewFlagSet("mdrelease check", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var cfg commonConfig
	var changelogFlag string
	fs.StringVar(&changelogFlag, "changelog", "", "Path to changelog file (default: changelog.md)")
	fs.StringVar(&cfg.remote, "remote", "origin", "Git remote name")
	fs.StringVar(&cfg.tagPrefix, "tag-prefix", "v", "Tag prefix")
	fs.BoolVar(&cfg.dryRun, "dry-run", false, "Print planned checks without running mutating steps (skips fetch --tags)")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return &usageError{msg: err.Error()}
	}
	if fs.NArg() != 0 {
		return &usageError{msg: "check does not accept positional arguments"}
	}
	cfg.changelogPath = resolveChangelogPath(changelogFlag, d.getenv)

	entry, err := changelog.ParseLatest(cfg.changelogPath)
	if err != nil {
		return err
	}

	tag := cfg.tagPrefix + entry.Version
	_, _ = fmt.Fprintf(stdout, "Release check:\n")
	_, _ = fmt.Fprintf(stdout, "  Changelog: %s\n", cfg.changelogPath)
	_, _ = fmt.Fprintf(stdout, "  Version: %s\n", entry.Version)
	_, _ = fmt.Fprintf(stdout, "  Title: %s\n", entry.Summary)
	_, _ = fmt.Fprintf(stdout, "  Tag: %s\n", tag)

	git := d.newGit(stdout, stderr, cfg.dryRun)
	if err := git.EnsureRepo(); err != nil {
		return err
	}
	if err := git.EnsureRemote(cfg.remote); err != nil {
		return err
	}
	if cfg.dryRun {
		_, _ = fmt.Fprintln(stdout, "  Fetch tags: skipped in --dry-run")
	} else {
		if err := git.FetchTags(); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(stdout, "  Fetch tags: ok")
	}
	if err := git.EnsureTagAbsent(tag); err != nil {
		return &preflightError{msg: fmt.Sprintf("no new changelog version to release: %s already exists (update %s)", tag, cfg.changelogPath)}
	}
	_, _ = fmt.Fprintln(stdout, "  Tag availability: ok")
	_, _ = fmt.Fprintln(stdout, "Check passed.")
	return nil
}

func runRelease(args []string, stdout, stderr io.Writer, d deps) error {
	fs := flag.NewFlagSet("mdrelease", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var cfg commonConfig
	var changelogFlag string
	var all bool
	var push bool
	var forceRetag bool
	var actions releaseActions

	fs.StringVar(&changelogFlag, "changelog", "", "Path to changelog file (default: changelog.md)")
	fs.StringVar(&cfg.remote, "remote", "origin", "Git remote name")
	fs.StringVar(&cfg.tagPrefix, "tag-prefix", "v", "Tag prefix")
	fs.BoolVar(&cfg.dryRun, "dry-run", false, "Print planned actions without mutating git state")
	fs.BoolVar(&all, "all", false, "Run full release pipeline (default behavior)")
	fs.BoolVar(&actions.stageAll, "stage-all", false, "Stage all changes (git add -A)")
	fs.BoolVar(&actions.commit, "commit", false, "Commit staged changes using changelog title/body")
	fs.BoolVar(&actions.tag, "tag", false, "Create annotated tag for changelog version")
	fs.BoolVar(&push, "push", false, "Push commit and tag (alias for --push-commit --push-tag)")
	fs.BoolVar(&actions.pushCommit, "push-commit", false, "Push HEAD to remote")
	fs.BoolVar(&actions.pushTag, "push-tag", false, "Push version tag to remote")
	fs.BoolVar(&forceRetag, "force-retag", false, "Overwrite an existing release tag by deleting and recreating it locally/remotely as needed")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printRootUsage(stdout)
			return nil
		}
		return &usageError{msg: err.Error()}
	}
	if fs.NArg() != 0 {
		return &usageError{msg: "mdrelease does not accept positional arguments (use subcommands: check, version)"}
	}
	cfg.changelogPath = resolveChangelogPath(changelogFlag, d.getenv)

	visited := visitedFlags(fs)
	explicitMutation := visited["stage-all"] || visited["commit"] || visited["tag"] || visited["push"] || visited["push-commit"] || visited["push-tag"]
	if all && explicitMutation {
		return &usageError{msg: "--all cannot be combined with individual release action flags"}
	}

	if push {
		actions.pushCommit = true
		actions.pushTag = true
	}

	if all || !explicitMutation {
		actions = releaseActions{
			stageAll:   true,
			commit:     true,
			tag:        true,
			pushCommit: true,
			pushTag:    true,
		}
	}

	entry, err := changelog.ParseLatest(cfg.changelogPath)
	if err != nil {
		return err
	}
	tag := cfg.tagPrefix + entry.Version

	_, _ = fmt.Fprintln(stdout, "Release info:")
	_, _ = fmt.Fprintf(stdout, "  Changelog: %s\n", cfg.changelogPath)
	_, _ = fmt.Fprintf(stdout, "  Version: %s\n", entry.Version)
	_, _ = fmt.Fprintf(stdout, "  Title: %s\n", entry.Summary)
	_, _ = fmt.Fprintf(stdout, "  Tag: %s\n", tag)
	_, _ = fmt.Fprintf(stdout, "  Actions: %s\n", actions.String())

	if cfg.dryRun {
		_, _ = fmt.Fprintln(stdout, "  Mode: dry-run")
	}

	git := d.newGit(stdout, stderr, cfg.dryRun)
	if err := git.EnsureRepo(); err != nil {
		return err
	}
	needsRemote := actions.pushCommit || actions.pushTag
	if needsRemote {
		if err := git.EnsureRemote(cfg.remote); err != nil {
			return err
		}
		if err := git.FetchRemote(cfg.remote); err != nil {
			return err
		}
		if err := git.PullFFOnly(cfg.remote); err != nil {
			return err
		}
	}

	if actions.tag {
		if forceRetag {
			if actions.pushTag {
				hasRemoteTag, err := git.HasRemoteTag(cfg.remote, tag)
				if err != nil {
					return err
				}
				if hasRemoteTag {
					_, _ = fmt.Fprintf(stdout, "Deleting remote tag %s from %s...\n", tag, cfg.remote)
					if err := git.DeleteRemoteTag(cfg.remote, tag); err != nil {
						return err
					}
				}
			}
			hasLocalTag, err := git.HasLocalTag(tag)
			if err != nil {
				return err
			}
			if hasLocalTag {
				_, _ = fmt.Fprintf(stdout, "Deleting local tag %s...\n", tag)
				if err := git.DeleteLocalTag(tag); err != nil {
					return err
				}
			}
		} else {
			if err := git.EnsureTagAbsent(tag); err != nil {
				return &preflightError{msg: fmt.Sprintf("no new changelog version to release: %s already exists (update %s)", tag, cfg.changelogPath)}
			}
		}
	}

	if forceRetag && actions.pushTag && !actions.tag {
		hasRemoteTag, err := git.HasRemoteTag(cfg.remote, tag)
		if err != nil {
			return err
		}
		if hasRemoteTag {
			_, _ = fmt.Fprintf(stdout, "Deleting remote tag %s from %s...\n", tag, cfg.remote)
			if err := git.DeleteRemoteTag(cfg.remote, tag); err != nil {
				return err
			}
		}
	}

	if actions.pushTag && !actions.tag {
		if err := git.EnsureTagPresent(tag); err != nil {
			return &preflightError{msg: fmt.Sprintf("cannot push tag %s: create it first with --tag (or use default mdrelease/--all)", tag)}
		}
	}

	if actions.stageAll {
		_, _ = fmt.Fprintln(stdout, "Staging changes...")
		if err := git.StageAll(); err != nil {
			return err
		}
	}

	if actions.commit {
		if cfg.dryRun && actions.stageAll {
			_, _ = fmt.Fprintln(stdout, "Skipping staged-change verification in --dry-run after --stage-all.")
		} else {
			hasStaged, err := git.HasStagedChanges()
			if err != nil {
				return err
			}
			if !hasStaged {
				msg := "no staged changes to commit"
				if actions.stageAll {
					msg = fmt.Sprintf("no changes to release after staging (update %s or make code changes)", cfg.changelogPath)
				}
				return &preflightError{msg: msg}
			}
		}

		_, _ = fmt.Fprintln(stdout, "Committing changes...")
		if err := git.Commit(entry.Summary, entry.Description); err != nil {
			return err
		}
	}

	createdTag := false
	if actions.tag {
		_, _ = fmt.Fprintf(stdout, "Creating tag %s...\n", tag)
		if err := git.CreateTag(tag, entry.Summary, entry.Description); err != nil {
			return err
		}
		createdTag = true
	}

	if actions.pushCommit {
		_, _ = fmt.Fprintf(stdout, "Pushing HEAD to %s...\n", cfg.remote)
		if err := git.PushHead(cfg.remote); err != nil {
			return err
		}
	}

	if actions.pushTag {
		_, _ = fmt.Fprintf(stdout, "Pushing tag %s to %s...\n", tag, cfg.remote)
		if err := git.PushTag(cfg.remote, tag); err != nil {
			if createdTag {
				return fmt.Errorf("%w (tag %s was created locally and may need manual push/retry)", err, tag)
			}
			return err
		}
	}

	if cfg.dryRun {
		_, _ = fmt.Fprintln(stdout, "Dry-run complete.")
		return nil
	}

	_, _ = fmt.Fprintf(stdout, "Release complete: %s (%s)\n", entry.Summary, tag)
	return nil
}

func resolveChangelogPath(flagValue string, getenv func(string) string) string {
	if strings.TrimSpace(flagValue) != "" {
		return flagValue
	}
	if getenv != nil {
		if v := strings.TrimSpace(getenv("MDRELEASE_CHANGELOG")); v != "" {
			return v
		}
	}
	return changelog.DefaultPath
}

func visitedFlags(fs *flag.FlagSet) map[string]bool {
	out := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		out[f.Name] = true
	})
	return out
}

func (a releaseActions) String() string {
	var parts []string
	if a.stageAll {
		parts = append(parts, "stage-all")
	}
	if a.commit {
		parts = append(parts, "commit")
	}
	if a.tag {
		parts = append(parts, "tag")
	}
	if a.pushCommit {
		parts = append(parts, "push-commit")
	}
	if a.pushTag {
		parts = append(parts, "push-tag")
	}
	if len(parts) == 0 {
		return "(none)"
	}
	return strings.Join(parts, ", ")
}

func printRootUsage(w io.Writer) {
	_, _ = fmt.Fprintln(w, "Usage:")
	_, _ = fmt.Fprintln(w, "  mdrelease [flags]        Run release (default is full release, equivalent to --all)")
	_, _ = fmt.Fprintln(w, "  mdrelease check [flags]  Validate changelog and git preconditions")
	_, _ = fmt.Fprintln(w, "  mdrelease version [flags] Print <latest-changelog-version>")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintf(w, "Installed mdrelease version: %s\n", ToolVersion)
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Global flags:")
	_, _ = fmt.Fprintln(w, "  --help, -h, -help        Print this usage")
	_, _ = fmt.Fprintln(w, "  --version, -version      Print installed mdrelease version (mdrelease version vX.Y.Z)")
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Examples:")
	_, _ = fmt.Fprintln(w, "  mdrelease")
	_, _ = fmt.Fprintln(w, "  mdrelease --all")
	_, _ = fmt.Fprintln(w, "  mdrelease --commit --tag --push")
	_, _ = fmt.Fprintln(w, "  mdrelease --tag --push-tag")
	_, _ = fmt.Fprintln(w, "  mdrelease --tag --push-tag --force-retag")
	_, _ = fmt.Fprintln(w, "  mdrelease --version")
	_, _ = fmt.Fprintln(w, "  mdrelease version")
}
