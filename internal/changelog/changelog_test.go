package changelog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseLatest_ValidEntry(t *testing.T) {
	path := writeFile(t, `
# 1.2.3 - Add release flow

- Added parser
- Added tests

# 1.2.2 - Previous
- Old
`)

	entry, err := ParseLatest(path)
	if err != nil {
		t.Fatalf("ParseLatest returned error: %v", err)
	}
	if entry.Version != "1.2.3" {
		t.Fatalf("version = %q, want %q", entry.Version, "1.2.3")
	}
	if entry.Summary != "Add release flow" {
		t.Fatalf("summary = %q", entry.Summary)
	}
	wantDesc := "- Added parser\n- Added tests"
	if entry.Description != wantDesc {
		t.Fatalf("description = %q, want %q", entry.Description, wantDesc)
	}
}

func TestParseLatest_IgnoresTopHeading(t *testing.T) {
	path := writeFile(t, `
# Changelog

Intro text

# 1.2.3-beta.1+exp.sha - Pre-release
- Works
`)

	entry, err := ParseLatest(path)
	if err != nil {
		t.Fatalf("ParseLatest returned error: %v", err)
	}
	if entry.Version != "1.2.3-beta.1+exp.sha" {
		t.Fatalf("version = %q", entry.Version)
	}
}

func TestParseLatest_NoBullets(t *testing.T) {
	path := writeFile(t, `
# 1.2.3 - Summary only

Paragraph text should be ignored.
`)

	entry, err := ParseLatest(path)
	if err != nil {
		t.Fatalf("ParseLatest returned error: %v", err)
	}
	if entry.Description != "" {
		t.Fatalf("description = %q, want empty", entry.Description)
	}
}

func TestParseLatest_InvalidFormat(t *testing.T) {
	path := writeFile(t, `
# Changelog

## 1.2.3 - Unsupported level
- No parse
`)

	_, err := ParseLatest(path)
	if err == nil {
		t.Fatal("expected parse error")
	}
	if _, ok := err.(*ParseError); !ok {
		t.Fatalf("error type = %T, want *ParseError", err)
	}
}

func writeFile(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "changelog.md")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	return path
}
