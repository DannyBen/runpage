package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	assertContains(t, stdout.String(), "--list, -l")
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

func TestModes(t *testing.T) {
	dir := t.TempDir()
	document := filepath.Join(dir, "guide.md")
	if err := os.WriteFile(document, []byte("# Guide\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "list", args: []string{document, "--list"}, want: "list " + document + " (not implemented)\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := Execute(tt.args, "1.2.3", &stdout, &stderr)
			if err != nil {
				t.Fatalf("Execute returned error: %v", err)
			}
			if got := stdout.String(); got != tt.want {
				t.Fatalf("stdout = %q, want %q", got, tt.want)
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q", stderr.String())
			}
		})
	}
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

	var stdout, stderr bytes.Buffer
	if err := Execute([]string{"--list"}, "1.2.3", &stdout, &stderr); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if got, want := stdout.String(), "list README.md (not implemented)\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}

	if err := os.WriteFile("mob.md", []byte("# Mob\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"--list"}, "1.2.3", &stdout, &stderr); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if got, want := stdout.String(), "list mob.md (not implemented)\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
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

func TestModesAreMutuallyExclusive(t *testing.T) {
	document := filepath.Join(t.TempDir(), "guide.md")
	if err := os.WriteFile(document, []byte("# Guide\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer

	err := Execute([]string{document, "--show", "--list"}, "1.2.3", &stdout, &stderr)

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
