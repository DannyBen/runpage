package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestShowDocumentErrors(t *testing.T) {
	if err := showDocument(filepath.Join(t.TempDir(), "missing.md"), os.Stdout); err == nil {
		t.Fatal("expected missing document error")
	}

	document := filepath.Join(t.TempDir(), "guide.md")
	if err := os.WriteFile(document, []byte("# Guide\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := showDocument(document, failingWriter{}); err == nil || err.Error() != "write failed" {
		t.Fatalf("write error = %v", err)
	}
}

func TestOutputOptionsForNonTerminal(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if colorOutput(os.Stdout) {
		t.Fatal("NO_COLOR output was colored")
	}

	t.Setenv("COLUMNS", "57")
	if got := renderWidth(failingWriter{}); got != 57 {
		t.Fatalf("renderWidth = %d, want 57", got)
	}
	t.Setenv("COLUMNS", "invalid")
	if got := renderWidth(failingWriter{}); got != defaultRenderWidth {
		t.Fatalf("renderWidth = %d, want %d", got, defaultRenderWidth)
	}
}
