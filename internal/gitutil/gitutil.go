package gitutil

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type GitError struct {
	Op  string
	Err error
}

func (e *GitError) Error() string {
	if e.Err == nil {
		return e.Op
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

func (e *GitError) Unwrap() error { return e.Err }

type Client struct {
	Stdout io.Writer
	Stderr io.Writer
	DryRun bool
}

func NewClient(stdout, stderr io.Writer, dryRun bool) *Client {
	return &Client{
		Stdout: stdout,
		Stderr: stderr,
		DryRun: dryRun,
	}
}

func (c *Client) EnsureRepo() error {
	out, err := c.output("git", "rev-parse", "--is-inside-work-tree")
	if err != nil || strings.TrimSpace(out) != "true" {
		return &GitError{Op: "validate git repository", Err: fmt.Errorf("not a git repository")}
	}
	return nil
}

func (c *Client) EnsureRemote(remote string) error {
	if err := c.run("git", "remote", "get-url", remote); err != nil {
		return &GitError{
			Op: "validate git remote",
			Err: fmt.Errorf(
				"no %q remote set (mdrelease uses git remotes, not gh; set one with `git remote add %s <url>` or pass --remote <name>)",
				remote,
				remote,
			),
		}
	}
	return nil
}

func (c *Client) FetchTags() error {
	if c.DryRun {
		c.printf("[dry-run] git fetch --tags\n")
		return nil
	}
	if err := c.runWithStreams("git", "fetch", "--tags"); err != nil {
		return &GitError{Op: "fetch tags", Err: err}
	}
	return nil
}

func (c *Client) FetchRemote(remote string) error {
	if c.DryRun {
		c.printf("[dry-run] git fetch --tags --prune %s\n", remote)
		return nil
	}
	if err := c.runWithStreams("git", "fetch", "--tags", "--prune", remote); err != nil {
		return &GitError{Op: "fetch remote refs", Err: err}
	}
	return nil
}

func (c *Client) PullFFOnly(remote string) error {
	if c.DryRun {
		c.printf("[dry-run] git pull --ff-only %s\n", remote)
		return nil
	}
	if err := c.runWithStreams("git", "pull", "--ff-only", remote); err != nil {
		return &GitError{Op: "pull fast-forward", Err: err}
	}
	return nil
}

func (c *Client) EnsureTagAbsent(tag string) error {
	ref := "refs/tags/" + tag
	if err := c.ensureValidRef(ref); err != nil {
		return &GitError{Op: "validate tag absence", Err: err}
	}
	err := c.runQuietAllowNotFound("git", "show-ref", "--verify", "--quiet", ref)
	if err == nil {
		return &GitError{Op: "validate tag absence", Err: fmt.Errorf("tag %s already exists", tag)}
	}
	var nf *notFoundError
	if errors.As(err, &nf) {
		return nil
	}
	return &GitError{Op: "validate tag absence", Err: err}
}

func (c *Client) EnsureTagPresent(tag string) error {
	ref := "refs/tags/" + tag
	if err := c.ensureValidRef(ref); err != nil {
		return &GitError{Op: "validate local tag", Err: err}
	}
	err := c.runQuietAllowNotFound("git", "show-ref", "--verify", "--quiet", ref)
	if err != nil {
		var nf *notFoundError
		if errors.As(err, &nf) {
			return &GitError{Op: "validate local tag", Err: fmt.Errorf("tag %s does not exist locally", tag)}
		}
		return &GitError{Op: "validate local tag", Err: err}
	}
	return nil
}

func (c *Client) HasLocalTag(tag string) (bool, error) {
	ref := "refs/tags/" + tag
	if err := c.ensureValidRef(ref); err != nil {
		return false, &GitError{Op: "check local tag", Err: err}
	}
	err := c.runQuietAllowNotFound("git", "show-ref", "--verify", "--quiet", ref)
	if err != nil {
		var nf *notFoundError
		if errors.As(err, &nf) {
			return false, nil
		}
		return false, &GitError{Op: "check local tag", Err: err}
	}
	return true, nil
}

func (c *Client) HasRemoteTag(remote, tag string) (bool, error) {
	ref := "refs/tags/" + tag
	if err := c.ensureValidRef(ref); err != nil {
		return false, &GitError{Op: "check remote tag", Err: err}
	}
	out, err := c.output("git", "ls-remote", "--tags", "--refs", remote, ref)
	if err != nil {
		return false, &GitError{Op: "check remote tag", Err: err}
	}
	return strings.TrimSpace(out) != "", nil
}

func (c *Client) DeleteLocalTag(tag string) error {
	if c.DryRun {
		c.printf("[dry-run] git tag -d %s\n", tag)
		return nil
	}
	if err := c.runWithStreams("git", "tag", "-d", tag); err != nil {
		return &GitError{Op: "delete local tag", Err: err}
	}
	return nil
}

func (c *Client) DeleteRemoteTag(remote, tag string) error {
	ref := "refs/tags/" + tag
	if err := c.ensureValidRef(ref); err != nil {
		return &GitError{Op: "delete remote tag", Err: err}
	}
	if c.DryRun {
		c.printf("[dry-run] git push %s :%s\n", remote, ref)
		return nil
	}
	if err := c.runWithStreams("git", "push", remote, ":"+ref); err != nil {
		return &GitError{Op: "delete remote tag", Err: err}
	}
	return nil
}

func (c *Client) ensureValidRef(ref string) error {
	if err := c.run("git", "check-ref-format", ref); err != nil {
		return fmt.Errorf("invalid ref name %q", ref)
	}
	return nil
}

type notFoundError struct {
	name string
	args []string
}

func (e *notFoundError) Error() string {
	return fmt.Sprintf("%s %s: not found", e.name, strings.Join(e.args, " "))
}

func (c *Client) runQuietAllowNotFound(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	var stderr bytes.Buffer
	cmd.Stdout = io.Discard
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return &notFoundError{name: name, args: args}
		}
		if stderr.Len() > 0 {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return err
	}
	return nil
}

func (c *Client) StageAll() error {
	if c.DryRun {
		c.printf("[dry-run] git add -A\n")
		return nil
	}
	if err := c.runWithStreams("git", "add", "-A"); err != nil {
		return &GitError{Op: "stage changes", Err: err}
	}
	return nil
}

func (c *Client) HasStagedChanges() (bool, error) {
	out, err := c.output("git", "diff", "--cached", "--name-only")
	if err != nil {
		return false, &GitError{Op: "check staged changes", Err: err}
	}
	return strings.TrimSpace(out) != "", nil
}

func (c *Client) Commit(summary, description string) error {
	if c.DryRun {
		c.printf("[dry-run] git commit -m %q", summary)
		if description != "" {
			c.printf(" -m <description>")
		}
		c.printf("\n")
		return nil
	}

	args := []string{"commit", "-m", summary}
	if description != "" {
		args = append(args, "-m", description)
	}
	if err := c.runWithStreams("git", args...); err != nil {
		return &GitError{Op: "commit changes", Err: err}
	}
	return nil
}

func (c *Client) CreateTag(tag, summary, description string) error {
	if c.DryRun {
		c.printf("[dry-run] git tag -a %s -m %q", tag, summary)
		if description != "" {
			c.printf(" (with description)")
		}
		c.printf("\n")
		return nil
	}

	message := summary
	if description != "" {
		message = summary + "\n\n" + description
	}
	if err := c.run("git", "tag", "-a", tag, "-m", message); err != nil {
		return &GitError{Op: "create tag", Err: err}
	}
	return nil
}

func (c *Client) PushHead(remote string) error {
	if c.DryRun {
		c.printf("[dry-run] git push %s HEAD\n", remote)
		return nil
	}
	if err := c.runWithStreams("git", "push", remote, "HEAD"); err != nil {
		return &GitError{Op: "push commit", Err: err}
	}
	return nil
}

func (c *Client) PushTag(remote, tag string) error {
	if c.DryRun {
		c.printf("[dry-run] git push %s %s\n", remote, tag)
		return nil
	}
	if err := c.runWithStreams("git", "push", remote, tag); err != nil {
		return &GitError{Op: "push tag", Err: err}
	}
	return nil
}

func (c *Client) output(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return "", err
	}
	return string(out), nil
}

func (c *Client) run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return err
	}
	return nil
}

func (c *Client) runWithStreams(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = c.Stdout
	cmd.Stderr = c.Stderr
	return cmd.Run()
}

func (c *Client) printf(format string, args ...any) {
	if c.Stdout != nil {
		_, _ = fmt.Fprintf(c.Stdout, format, args...)
	}
}
