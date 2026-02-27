package gitutil

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestEnsureTagChecksUseExactTagRefs(t *testing.T) {
	repo := initRepo(t)
	runGit(t, repo, "checkout", "-b", "v1.2.3")

	c := NewClient(&bytes.Buffer{}, &bytes.Buffer{}, false)

	if err := withDir(repo, func() error { return c.EnsureTagAbsent("v1.2.3") }); err != nil {
		t.Fatalf("EnsureTagAbsent should ignore same-named branch: %v", err)
	}
	if err := withDir(repo, func() error { return c.EnsureTagPresent("v1.2.3") }); err == nil {
		t.Fatal("EnsureTagPresent should fail when only a branch exists")
	}

	runGit(t, repo, "tag", "v1.2.3")
	if err := withDir(repo, func() error { return c.EnsureTagPresent("v1.2.3") }); err != nil {
		t.Fatalf("EnsureTagPresent failed for real tag: %v", err)
	}
}

func TestEnsureTagAbsent_InvalidRefReturnsError(t *testing.T) {
	repo := initRepo(t)
	c := NewClient(&bytes.Buffer{}, &bytes.Buffer{}, false)

	err := withDir(repo, func() error { return c.EnsureTagAbsent("bad..ref") })
	if err == nil {
		t.Fatal("expected error for invalid tag ref")
	}
	var ge *GitError
	if !errors.As(err, &ge) {
		t.Fatalf("error type = %T, want *GitError", err)
	}
}

func TestHasRemoteTagAndDeleteRemoteTag(t *testing.T) {
	repo := initRepo(t)
	remoteRoot := t.TempDir()
	remote := filepath.Join(remoteRoot, "origin.git")
	runGit(t, remoteRoot, "init", "--bare", remote)

	runGit(t, repo, "remote", "add", "origin", remote)
	runGit(t, repo, "push", "-u", "origin", "HEAD")
	runGit(t, repo, "tag", "v1.2.3")
	runGit(t, repo, "push", "origin", "v1.2.3")

	c := NewClient(&bytes.Buffer{}, &bytes.Buffer{}, false)
	if err := withDir(repo, func() error {
		ok, err := c.HasRemoteTag("origin", "v1.2.3")
		if err != nil {
			return err
		}
		if !ok {
			t.Fatal("expected remote tag to exist")
		}
		return c.DeleteRemoteTag("origin", "v1.2.3")
	}); err != nil {
		t.Fatalf("remote tag delete flow failed: %v", err)
	}

	if err := withDir(repo, func() error {
		ok, err := c.HasRemoteTag("origin", "v1.2.3")
		if err != nil {
			return err
		}
		if ok {
			t.Fatal("expected remote tag to be deleted")
		}
		return nil
	}); err != nil {
		t.Fatalf("remote tag existence check failed: %v", err)
	}
}

func TestHasLocalTagAndDeleteLocalTag(t *testing.T) {
	repo := initRepo(t)
	runGit(t, repo, "tag", "v1.2.3")
	c := NewClient(&bytes.Buffer{}, &bytes.Buffer{}, false)

	if err := withDir(repo, func() error {
		ok, err := c.HasLocalTag("v1.2.3")
		if err != nil {
			return err
		}
		if !ok {
			t.Fatal("expected local tag to exist")
		}
		return c.DeleteLocalTag("v1.2.3")
	}); err != nil {
		t.Fatalf("local tag delete flow failed: %v", err)
	}

	if err := withDir(repo, func() error {
		ok, err := c.HasLocalTag("v1.2.3")
		if err != nil {
			return err
		}
		if ok {
			t.Fatal("expected local tag to be deleted")
		}
		return nil
	}); err != nil {
		t.Fatalf("local tag existence check failed: %v", err)
	}
}

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.name", "Test User")
	runGit(t, dir, "config", "user.email", "test@example.com")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("test\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	runGit(t, dir, "add", "README.md")
	runGit(t, dir, "commit", "-m", "init")
	return dir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
}

func withDir(dir string, fn func() error) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := os.Chdir(dir); err != nil {
		return err
	}
	defer func() { _ = os.Chdir(wd) }()
	return fn()
}
