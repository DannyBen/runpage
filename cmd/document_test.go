package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseFrontMatter(t *testing.T) {
	markdown := []byte(strings.Join([]string{
		"---",
		"title: Release checklist",
		"runpage:",
		"  required: [version, channel]",
		"  dependencies:",
		"    - git",
		"    - curl",
		"  workdir: .",
		"---",
		"# Release {{ version }}",
		"",
	}, "\n"))

	config, body, err := parseFrontMatter(markdown)
	if err != nil {
		t.Fatalf("parseFrontMatter returned error: %v", err)
	}
	if strings.Join(config.Required, ",") != "version,channel" {
		t.Fatalf("required = %#v", config.Required)
	}
	if strings.Join(config.Dependencies, ",") != "git,curl" || config.Workdir != "." {
		t.Fatalf("config = %#v", config)
	}
	if got := string(body); got != "# Release {{ version }}\n" {
		t.Fatalf("body = %q", got)
	}
}

func TestParseFrontMatterLeavesOrdinaryDocumentsAlone(t *testing.T) {
	markdown := []byte("# Guide\n\n---\n")
	config, body, err := parseFrontMatter(markdown)
	if err != nil || len(config.Required) != 0 || string(body) != string(markdown) {
		t.Fatalf("config = %#v, body = %q, err = %v", config, body, err)
	}
}

func TestParseFrontMatterErrors(t *testing.T) {
	for _, markdown := range []string{
		"---\nrunpage:\n  required: [version]\n",
		"---\nrunpage: [invalid\n---\n# Guide\n",
	} {
		if _, _, err := parseFrontMatter([]byte(markdown)); err == nil {
			t.Fatalf("front matter %q did not return an error", markdown)
		}
	}
}

func TestValidateDocumentConfig(t *testing.T) {
	for _, config := range []documentConfig{
		{Required: []string{"bad-key"}},
		{Required: []string{"version", "version"}},
		{Dependencies: []string{""}},
		{Dependencies: []string{"git", "git"}},
	} {
		if err := validateDocumentConfig(config); err == nil {
			t.Fatalf("config %#v did not return an error", config)
		}
	}
}

func TestDocumentRequirements(t *testing.T) {
	config := documentConfig{Required: []string{"version", "channel"}}
	if err := requireDocumentValues(config, documentValues{"version": "1.0.0"}); err == nil || !strings.Contains(err.Error(), "channel") {
		t.Fatalf("missing value error = %v", err)
	}
	if err := requireDocumentValues(config, documentValues{"version": "1.0.0", "channel": ""}); err == nil {
		t.Fatal("empty required value was accepted")
	}
	if err := requireDocumentValues(config, documentValues{"version": "1.0.0", "channel": "stable"}); err != nil {
		t.Fatalf("complete values returned error: %v", err)
	}
}

func TestDocumentDependencies(t *testing.T) {
	if err := requireDependencies(documentConfig{Dependencies: []string{"go"}}); err != nil {
		t.Fatalf("installed dependency returned error: %v", err)
	}
	if err := requireDependencies(documentConfig{Dependencies: []string{"runpage-command-that-does-not-exist"}}); err == nil {
		t.Fatal("missing dependency was accepted")
	}
}

func TestDocumentWorkdirOverride(t *testing.T) {
	override := t.TempDir()
	configured := t.TempDir()
	document := filepath.Join(t.TempDir(), "release.md")
	resolved, err := resolveDocumentWorkdir(documentConfig{Workdir: configured}, "", document)
	if err != nil {
		t.Fatalf("configured workdir returned error: %v", err)
	}
	if resolved != configured {
		t.Fatalf("configured resolved = %q, want %q", resolved, configured)
	}

	resolved, err = resolveDocumentWorkdir(documentConfig{Workdir: filepath.Join(t.TempDir(), "missing")}, override, document)
	if err != nil {
		t.Fatalf("override returned error: %v", err)
	}
	if resolved != override {
		t.Fatalf("resolved = %q, want %q", resolved, override)
	}
}

func TestDocumentWorkdirSelf(t *testing.T) {
	base := t.TempDir()
	documentDir := filepath.Join(base, "docs")
	if err := os.Mkdir(documentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	document := filepath.Join(documentDir, "release.md")

	resolved, err := resolveDocumentWorkdir(documentConfig{Workdir: "self"}, "", document)
	if err != nil {
		t.Fatalf("front matter self returned error: %v", err)
	}
	if resolved != documentDir {
		t.Fatalf("front matter self = %q, want %q", resolved, documentDir)
	}

	resolved, err = resolveDocumentWorkdir(documentConfig{Workdir: filepath.Join(base, "missing")}, "self", document)
	if err != nil {
		t.Fatalf("CLI self returned error: %v", err)
	}
	if resolved != documentDir {
		t.Fatalf("CLI self = %q, want %q", resolved, documentDir)
	}

	tasksDir := filepath.Join(documentDir, "tasks")
	if err := os.Mkdir(tasksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	resolved, err = resolveDocumentWorkdir(documentConfig{Workdir: "self/tasks"}, "", document)
	if err != nil {
		t.Fatalf("self/tasks returned error: %v", err)
	}
	if resolved != tasksDir {
		t.Fatalf("self/tasks = %q, want %q", resolved, tasksDir)
	}

	resolved, err = resolveDocumentWorkdir(documentConfig{Workdir: "self/.."}, "", document)
	if err != nil {
		t.Fatalf("self/.. returned error: %v", err)
	}
	if resolved != base {
		t.Fatalf("self/.. = %q, want %q", resolved, base)
	}

	selfDir := filepath.Join(base, "self")
	if err := os.Mkdir(selfDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(base)
	resolved, err = resolveDocumentWorkdir(documentConfig{Workdir: "./self"}, "", document)
	if err != nil {
		t.Fatalf("./self returned error: %v", err)
	}
	if resolved != selfDir {
		t.Fatalf("./self = %q, want %q", resolved, selfDir)
	}
}

func TestFrontMatterModeValidation(t *testing.T) {
	document := filepath.Join(t.TempDir(), "release.md")
	markdown := strings.Join([]string{
		"---",
		"runpage:",
		"  required: [version]",
		"  dependencies: [runpage-command-that-does-not-exist]",
		"---",
		"# Release {{ version }}",
	}, "\n")
	if err := os.WriteFile(document, []byte(markdown), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("show requires values but skips dependencies", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		err := Execute([]string{document, "--show"}, "1.2.3", &stdout, &stderr)
		if err == nil || !strings.Contains(err.Error(), "missing required document values: version") {
			t.Fatalf("error = %v", err)
		}
		stdout.Reset()
		err = Execute([]string{document, "--show", "version:1.0.0"}, "1.2.3", &stdout, &stderr)
		if err != nil {
			t.Fatalf("show returned error: %v", err)
		}
		if strings.Contains(stdout.String(), "runpage:") || !strings.Contains(stdout.String(), "Release 1.0.0") {
			t.Fatalf("show output = %q", stdout.String())
		}
	})

	t.Run("read bypasses values and dependencies", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		err := Execute([]string{document, "--read"}, "1.2.3", &stdout, &stderr)
		if err == nil || !strings.Contains(err.Error(), "interactive reader requires a terminal") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("execution checks dependencies", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		err := Execute([]string{document, "version:1.0.0"}, "1.2.3", &stdout, &stderr)
		if err == nil || !strings.Contains(err.Error(), "required dependency not found") {
			t.Fatalf("error = %v", err)
		}
	})
}
