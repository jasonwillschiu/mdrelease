package app

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeGit struct {
	calls               []string
	hasStaged           bool
	ensureTagAbsentErr  error
	ensureTagPresentErr error
	pushTagErr          error
}

func (f *fakeGit) EnsureRepo() error { f.calls = append(f.calls, "EnsureRepo"); return nil }
func (f *fakeGit) EnsureRemote(remote string) error {
	f.calls = append(f.calls, "EnsureRemote:"+remote)
	return nil
}
func (f *fakeGit) FetchTags() error { f.calls = append(f.calls, "FetchTags"); return nil }
func (f *fakeGit) EnsureTagAbsent(tag string) error {
	f.calls = append(f.calls, "EnsureTagAbsent:"+tag)
	return f.ensureTagAbsentErr
}
func (f *fakeGit) EnsureTagPresent(tag string) error {
	f.calls = append(f.calls, "EnsureTagPresent:"+tag)
	return f.ensureTagPresentErr
}
func (f *fakeGit) StageAll() error { f.calls = append(f.calls, "StageAll"); return nil }
func (f *fakeGit) HasStagedChanges() (bool, error) {
	f.calls = append(f.calls, "HasStagedChanges")
	return f.hasStaged, nil
}
func (f *fakeGit) Commit(summary, desc string) error {
	f.calls = append(f.calls, "Commit:"+summary)
	return nil
}
func (f *fakeGit) CreateTag(tag, summary, desc string) error {
	f.calls = append(f.calls, "CreateTag:"+tag)
	return nil
}
func (f *fakeGit) PushHead(remote string) error {
	f.calls = append(f.calls, "PushHead:"+remote)
	return nil
}
func (f *fakeGit) PushTag(remote, tag string) error {
	f.calls = append(f.calls, "PushTag:"+remote+":"+tag)
	return f.pushTagErr
}

func TestResolveChangelogPath_PrefersFlagThenEnvThenDefault(t *testing.T) {
	getenv := func(k string) string {
		if k == "MDRELEASE_CHANGELOG" {
			return "env.md"
		}
		return ""
	}

	if got := resolveChangelogPath("flag.md", getenv); got != "flag.md" {
		t.Fatalf("got %q", got)
	}
	if got := resolveChangelogPath("", getenv); got != "env.md" {
		t.Fatalf("got %q", got)
	}
	if got := resolveChangelogPath("", func(string) string { return "" }); got != "changelog.md" {
		t.Fatalf("got %q", got)
	}
}

func TestRun_HelpFlagPrintsRootUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := Run([]string{"--help"}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d", code, ExitOK)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Fatalf("stdout missing usage, got: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr not empty: %q", stderr.String())
	}
}

func TestRun_VersionFlagAliasesVersionCommand(t *testing.T) {
	changelogPath := writeChangelog(t)
	var stdout, stderr bytes.Buffer

	code := Run([]string{"--version", "--changelog", changelogPath}, &stdout, &stderr)
	if code != ExitOK {
		t.Fatalf("exit code = %d, want %d (stderr: %s)", code, ExitOK, stderr.String())
	}
	if got := strings.TrimSpace(stdout.String()); got != "1.2.3" {
		t.Fatalf("stdout = %q, want %q", got, "1.2.3")
	}
}

func TestRunRelease_DefaultIsAll(t *testing.T) {
	changelogPath := writeChangelog(t)
	fg := &fakeGit{hasStaged: true}

	var stdout, stderr bytes.Buffer
	err := run([]string{"--changelog", changelogPath}, &stdout, &stderr, deps{
		getenv: func(string) string { return "" },
		newGit: func(out, errOut io.Writer, dry bool) gitOps { return fg },
	})
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	wantOrder := []string{
		"EnsureRepo",
		"EnsureRemote:origin",
		"FetchTags",
		"EnsureTagAbsent:v1.2.3",
		"StageAll",
		"HasStagedChanges",
		"Commit:Release title",
		"CreateTag:v1.2.3",
		"PushHead:origin",
		"PushTag:origin:v1.2.3",
	}
	if got := strings.Join(fg.calls, "|"); got != strings.Join(wantOrder, "|") {
		t.Fatalf("call order mismatch:\n got: %v\nwant: %v", fg.calls, wantOrder)
	}
}

func TestRunRelease_RejectsAllWithIndividualFlags(t *testing.T) {
	changelogPath := writeChangelog(t)

	err := run([]string{"--all", "--tag", "--changelog", changelogPath}, &bytes.Buffer{}, &bytes.Buffer{}, deps{
		getenv: func(string) string { return "" },
		newGit: func(out, errOut io.Writer, dry bool) gitOps { return &fakeGit{} },
	})
	if err == nil {
		t.Fatal("expected error")
	}
	var ue *usageError
	if !errors.As(err, &ue) {
		t.Fatalf("error type %T, want usageError", err)
	}
}

func TestRunRelease_TagOnlyFlow(t *testing.T) {
	changelogPath := writeChangelog(t)
	fg := &fakeGit{}

	err := run([]string{"--changelog", changelogPath, "--tag", "--push-tag"}, &bytes.Buffer{}, &bytes.Buffer{}, deps{
		getenv: func(string) string { return "" },
		newGit: func(out, errOut io.Writer, dry bool) gitOps { return fg },
	})
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	got := strings.Join(fg.calls, "|")
	if strings.Contains(got, "StageAll") || strings.Contains(got, "Commit:") {
		t.Fatalf("unexpected commit path calls: %v", fg.calls)
	}
}

func TestRunRelease_FailsWhenNoChangesAfterStageAll(t *testing.T) {
	changelogPath := writeChangelog(t)
	fg := &fakeGit{hasStaged: false}

	err := run([]string{"--changelog", changelogPath}, &bytes.Buffer{}, &bytes.Buffer{}, deps{
		getenv: func(string) string { return "" },
		newGit: func(out, errOut io.Writer, dry bool) gitOps { return fg },
	})
	if err == nil {
		t.Fatal("expected error")
	}
	var pe *preflightError
	if !errors.As(err, &pe) {
		t.Fatalf("error type %T, want preflightError", err)
	}
}

func TestRunRelease_PushTagFailureMentionsLocalTag(t *testing.T) {
	changelogPath := writeChangelog(t)
	fg := &fakeGit{
		hasStaged:  true,
		pushTagErr: fmt.Errorf("push failed"),
	}

	err := run([]string{"--changelog", changelogPath}, &bytes.Buffer{}, &bytes.Buffer{}, deps{
		getenv: func(string) string { return "" },
		newGit: func(out, errOut io.Writer, dry bool) gitOps { return fg },
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "created locally") {
		t.Fatalf("missing partial success guidance: %v", err)
	}
}

func writeChangelog(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "changelog.md")
	content := "# 1.2.3 - Release title\n\n- First change\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write changelog: %v", err)
	}
	return path
}
