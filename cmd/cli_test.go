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
	assertContains(t, stdout.String(), "Runpage - Interactive command pages for the terminal")
	assertContains(t, stdout.String(), "runpage [FILE] [options] [KEY:VALUE...]")
	assertContains(t, stdout.String(), "--show, -s")
	assertContains(t, stdout.String(), "--read, -r")
	assertContains(t, stdout.String(), "--compact, -c")
	assertContains(t, stdout.String(), "--run TAG")
	assertContains(t, stdout.String(), "--workdir DIR, -w DIR")
	assertContains(t, stdout.String(), "--syntax")
	assertContains(t, stdout.String(), "YAML front matter")
}

func TestRunExecutesTaggedBlocksInOrder(t *testing.T) {
	document := filepath.Join(t.TempDir(), "checks.md")
	markdown := "```sh first :check\nprintf 'first\\n'\n```\n\n```sh skipped :other\nprintf 'skipped\\n'\n```\n\n```sh second :check\nprintf 'second\\n'\n```\n"
	if err := os.WriteFile(document, []byte(markdown), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, tag := range []string{"check", ":check"} {
		t.Run(tag, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := Execute([]string{document, "--run", tag, "--show"}, "1.2.3", &stdout, &stderr)
			if err != nil {
				t.Fatalf("Execute returned error: %v", err)
			}
			assertContains(t, stdout.String(), "first")
			assertContains(t, stdout.String(), "second")
			assertContains(t, stdout.String(), "2 commands: 2 succeeded")
			if got := strings.Count(stdout.String(), "┌─ first"); got != 1 {
				t.Fatalf("first frame headers = %d, want 1:\n%s", got, stdout.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q", stderr.String())
			}
		})
	}
}

type notifyingWriter struct {
	buffer *bytes.Buffer
	notify func() error
	done   bool
}

func (writer *notifyingWriter) Write(content []byte) (int, error) {
	if !writer.done {
		writer.done = true
		if err := writer.notify(); err != nil {
			return 0, err
		}
	}
	return writer.buffer.Write(content)
}

func TestRunShowWritesPageBeforeExecutingBlock(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "page-started")
	document := filepath.Join(dir, "checks.md")
	markdown := "# Slow checks\n\nAbout to run.\n\n```sh waiting :check\ntest -f '" + marker + "'\nprintf 'done\\n'\n```\n"
	if err := os.WriteFile(document, []byte(markdown), 0o644); err != nil {
		t.Fatal(err)
	}

	var buffer, stderr bytes.Buffer
	stdout := &notifyingWriter{buffer: &buffer, notify: func() error {
		return os.WriteFile(marker, nil, 0o644)
	}}
	err := Execute([]string{document, "--run", "check", "--show"}, "1.2.3", stdout, &stderr)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	assertContains(t, buffer.String(), "About to run.")
	assertContains(t, buffer.String(), "done")
}

func TestRunShowUsesConfiguredWorkdir(t *testing.T) {
	dir := t.TempDir()
	document := filepath.Join(dir, "checks.md")
	markdown := "---\nrunpage:\n  workdir: self\n---\n\n```sh :check\npwd\n```\n"
	if err := os.WriteFile(document, []byte(markdown), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	err := Execute([]string{document, "--run", "check", "--show"}, "1.2.3", &stdout, &stderr)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	assertContains(t, stdout.String(), dir)
}

func TestRunStopsAtFirstFailure(t *testing.T) {
	document := filepath.Join(t.TempDir(), "checks.md")
	markdown := "```sh :check\nprintf 'before\\n'\nexit 7\n```\n\n```sh :check\nprintf 'after\\n'\n```\n"
	if err := os.WriteFile(document, []byte(markdown), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	err := Execute([]string{document, "--run", "check", "--show"}, "1.2.3", &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error")
	}
	assertContains(t, err.Error(), "exit code 7")
	assertContains(t, stdout.String(), "before")
	assertContains(t, stdout.String(), "2 commands: 1 failed, 1 skipped")
	if strings.Contains(stdout.String(), "after\n") {
		t.Fatalf("rendered command after failure output:\n%s", stdout.String())
	}
}

func TestRunRejectsEmptyAndMissingTags(t *testing.T) {
	document := filepath.Join(t.TempDir(), "checks.md")
	if err := os.WriteFile(document, []byte("```sh :check\ntrue\n```\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	err := Execute([]string{document, "--run", ""}, "1.2.3", &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "non-empty tag") {
		t.Fatalf("empty tag error = %v", err)
	}

	err = Execute([]string{document, "--run", "missing", "--show"}, "1.2.3", &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "no executable code blocks tagged :missing") {
		t.Fatalf("missing tag error = %v", err)
	}
}

func TestSyntaxHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := Execute([]string{"--syntax"}, "1.2.3", &stdout, &stderr)

	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	for _, want := range []string{
		"Runpage document syntax",
		"```LANG [LABEL...] [:TAG...]",
		"$KEY or {{ KEY }}",
		"under the runpage key",
		"workdir: self",
		"press ? for key binding help",
	} {
		assertContains(t, stdout.String(), want)
	}
}

func TestSyntaxHelpRejectsArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := Execute([]string{"--syntax", "page.md"}, "1.2.3", &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "does not accept arguments") {
		t.Fatalf("error = %v", err)
	}
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

func TestShowInjectsDocumentValues(t *testing.T) {
	document := filepath.Join(t.TempDir(), "release.md")
	markdown := "# Release $version\n\n```sh\nprintf '{{ version }}'\n```\n"
	if err := os.WriteFile(document, []byte(markdown), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	err := Execute([]string{document, "version:1.0.0", "--show"}, "1.2.3", &stdout, &stderr)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	assertContains(t, stdout.String(), "Release 1.0.0")
	assertContains(t, stdout.String(), "printf '1.0.0'")
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

	if err := os.WriteFile("runpage.md", []byte("# Runpage\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	document, err = resolveDocument(nil)
	if err != nil {
		t.Fatalf("resolveDocument returned error: %v", err)
	}
	if document != "runpage.md" {
		t.Fatalf("document = %q, want runpage.md", document)
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

func TestWorkdirSelfIsAccepted(t *testing.T) {
	document := filepath.Join(t.TempDir(), "guide.md")
	if err := os.WriteFile(document, []byte("# Guide\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	err := Execute([]string{document, "--workdir", "self"}, "1.2.3", &stdout, &stderr)
	if err == nil || !strings.Contains(err.Error(), "interactive reader requires a terminal") {
		t.Fatalf("error = %v", err)
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
	assertContains(t, err.Error(), "can only be combined with --run")
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("expected output to contain %q, got:\n%s", needle, haystack)
	}
}
