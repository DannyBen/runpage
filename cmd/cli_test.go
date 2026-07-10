package cmd

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrintError(t *testing.T) {
	var stderr bytes.Buffer
	PrintError(errors.New("broken"), &stderr)
	PrintError(nil, &stderr)
	if got, want := stderr.String(), "error: broken\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

func TestRootHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := Execute([]string{"--help"}, "1.2.3", &stdout, &stderr)

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	assertContains(t, stdout.String(), "Markdown Ops Book")
	assertContains(t, stdout.String(), "mob [FILE] [options]")
	assertContains(t, stdout.String(), "--show, -s")
	assertContains(t, stdout.String(), "--read, -r")
	assertContains(t, stdout.String(), "--compact, -c")
	assertContains(t, stdout.String(), "--workdir DIR, -w DIR")
}

func TestVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := Execute([]string{"--version"}, "1.2.3", &stdout, &stderr)

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if got, want := stdout.String(), "1.2.3\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestCompactRequiresTerminal(t *testing.T) {
	dir := t.TempDir()
	document := filepath.Join(dir, "guide.md")
	if err := os.WriteFile(document, []byte("# Guide\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	err := Execute([]string{document, "--compact"}, "1.2.3", &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error")
	}
	assertContains(t, err.Error(), "interactive reader requires a terminal")
}

func TestReadRequiresTerminal(t *testing.T) {
	document := filepath.Join(t.TempDir(), "guide.md")
	if err := os.WriteFile(document, []byte("# Guide\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer

	err := Execute([]string{document}, "1.2.3", &stdout, &stderr)

	if err == nil {
		t.Fatal("expected error")
	}
	assertContains(t, err.Error(), "interactive reader requires a terminal")
}

func TestShowRendersMarkdown(t *testing.T) {
	document := filepath.Join(t.TempDir(), "guide.md")
	markdown := "# Install\n\nRun the checks.\n\n```bash\ngo test ./...\n```\n"
	if err := os.WriteFile(document, []byte(markdown), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	err := Execute([]string{document, "--show"}, "1.2.3", &stdout, &stderr)

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	assertContains(t, stdout.String(), "Install")
	assertContains(t, stdout.String(), "Run the checks.")
	assertContains(t, stdout.String(), "go test ./...")
	if strings.Contains(stdout.String(), "\x1b[") {
		t.Fatalf("redirected output contains ANSI escapes: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestDefaultDocumentPreference(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if err := os.WriteFile("README.md", []byte("# Readme\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	document, err := resolveDocument(nil)
	if err != nil {
		t.Fatalf("resolveDocument returned error: %v", err)
	}
	if document != "README.md" {
		t.Fatalf("document = %q, want README.md", document)
	}

	if err := os.WriteFile("mob.md", []byte("# Mob\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	document, err = resolveDocument(nil)
	if err != nil {
		t.Fatalf("resolveDocument returned error: %v", err)
	}
	if document != "mob.md" {
		t.Fatalf("document = %q, want mob.md", document)
	}
}

func TestMissingDocument(t *testing.T) {
	t.Chdir(t.TempDir())
	var stdout, stderr bytes.Buffer

	err := Execute(nil, "1.2.3", &stdout, &stderr)

	if err == nil {
		t.Fatal("expected error")
	}
	assertContains(t, err.Error(), "no document found")
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestExplicitDocumentErrors(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "missing", path: filepath.Join(dir, "missing.md"), want: "document not found"},
		{name: "directory", path: dir, want: "document is a directory"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := Execute([]string{test.path, "--show"}, "1.2.3", &stdout, &stderr)
			if err == nil {
				t.Fatal("expected error")
			}
			assertContains(t, err.Error(), test.want)
		})
	}
}

func TestResolveWorkdir(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "project")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(base, "file")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	resolved, err := resolveWorkdir(dir)
	if err != nil {
		t.Fatalf("resolveWorkdir returned error: %v", err)
	}
	if resolved != dir {
		t.Fatalf("resolved = %q, want %q", resolved, dir)
	}
	t.Chdir(base)
	resolved, err = resolveWorkdir("project")
	if err != nil {
		t.Fatalf("resolve relative workdir returned error: %v", err)
	}
	if resolved != dir {
		t.Fatalf("relative resolved = %q, want %q", resolved, dir)
	}
	if _, err := resolveWorkdir(filepath.Join(base, "missing")); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("missing error = %v", err)
	}
	if _, err := resolveWorkdir(file); err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("file error = %v", err)
	}
}

func TestWorkdirIsValidatedBeforeOpeningReader(t *testing.T) {
	document := filepath.Join(t.TempDir(), "guide.md")
	if err := os.WriteFile(document, []byte("# Guide\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, flag := range []string{"--workdir", "-w"} {
		t.Run(flag, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := Execute([]string{document, flag, filepath.Join(t.TempDir(), "missing")}, "1.2.3", &stdout, &stderr)
			if err == nil {
				t.Fatal("expected error")
			}
			assertContains(t, err.Error(), "working directory not found")
		})
	}
}

func TestModesAreMutuallyExclusive(t *testing.T) {
	document := filepath.Join(t.TempDir(), "guide.md")
	if err := os.WriteFile(document, []byte("# Guide\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer

	err := Execute([]string{document, "--show", "--compact"}, "1.2.3", &stdout, &stderr)

	if err == nil {
		t.Fatal("expected error")
	}
	assertContains(t, err.Error(), "if any flags in the group")
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("expected output to contain %q, got:\n%s", needle, haystack)
	}
}
