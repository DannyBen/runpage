package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestReaderMovesBetweenExecutableBlocks(t *testing.T) {
	markdown := []byte("# Guide\n\n```bash\nop check\n```\n\n```yaml\na: b\n```\n\n```sh\nop test\n```\n")
	var output bytes.Buffer
	model := newReaderModel(markdown, &output, 80, 20)

	if len(model.blocks) != 2 {
		t.Fatalf("len(blocks) = %d, want 2", len(model.blocks))
	}
	if !strings.Contains(model.View(), "block 1/2") {
		t.Fatalf("initial view missing block status:\n%s", model.View())
	}
	if !strings.Contains(model.View(), "ready to execute") {
		t.Fatalf("initial view missing executable block:\n%s", model.View())
	}

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(readerModel)

	if model.focused != 1 {
		t.Fatalf("focused = %d, want 1", model.focused)
	}
	if !strings.Contains(model.View(), "block 2/2") {
		t.Fatalf("updated view missing block status:\n%s", model.View())
	}
	if !strings.Contains(model.View(), "executable") {
		t.Fatalf("updated view missing second executable block:\n%s", model.View())
	}
}

func TestReadOnlyReaderDoesNotSelectOrExecuteBlocks(t *testing.T) {
	markdown := []byte("# Guide\n\n```bash\nprintf should-not-run\n```\n")
	var output bytes.Buffer
	model := newReaderModelWithMode(markdown, &output, 80, 20, true)

	if model.focused != -1 {
		t.Fatalf("focused = %d, want -1", model.focused)
	}
	if !strings.Contains(model.View(), "read only") {
		t.Fatalf("read-only status missing:\n%s", model.View())
	}
	if strings.Contains(model.View(), "ready to execute") {
		t.Fatalf("read-only view selected an executable block:\n%s", model.View())
	}

	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyTab},
		{Type: tea.KeyShiftTab},
		{Type: tea.KeyCtrlDown},
		{Type: tea.KeyCtrlUp},
		{Type: tea.KeyEnter},
	} {
		updated, command := model.Update(key)
		model = updated.(readerModel)
		if command != nil {
			t.Fatalf("key %q returned an execution command", key.String())
		}
	}

	if model.focused != -1 || model.running != -1 || len(model.results) != 0 {
		t.Fatalf("read-only model changed execution state: %#v", model)
	}
}

func TestReaderNavigationResizeAndQuit(t *testing.T) {
	markdown := []byte("# Guide\n\n```bash\nprintf first\n```\n\n```sh\nprintf second\n```\n")
	var output bytes.Buffer
	model := newReaderModel(markdown, &output, 80, 20)

	if model.Init() != nil {
		t.Fatal("Init returned a command")
	}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	model = updated.(readerModel)
	if model.focused != 1 {
		t.Fatalf("backward focus did not wrap: %d", model.focused)
	}

	updated, _ = model.Update(tea.WindowSizeMsg{Width: 42, Height: 12})
	model = updated.(readerModel)
	if model.viewport.Width != 42 || model.viewport.Height != 11 {
		t.Fatalf("viewport size = %dx%d", model.viewport.Width, model.viewport.Height)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	model = updated.(readerModel)
	if model.viewport.YOffset != 0 {
		t.Fatalf("top offset = %d", model.viewport.YOffset)
	}
	_, command := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if command == nil {
		t.Fatal("quit did not return a command")
	}
}

func TestReaderWithoutBlocksIgnoresExecutionControls(t *testing.T) {
	var output bytes.Buffer
	model := newReaderModel([]byte("# Guide\n\nNo commands.\n"), &output, 80, 20)
	model.focus(1)
	if command := model.startExecution(); command != nil {
		t.Fatal("reader without blocks returned an execution command")
	}
	if strings.Contains(model.View(), "block 1/") {
		t.Fatalf("block status shown without blocks:\n%s", model.View())
	}
}

func TestReaderViewReportsModelError(t *testing.T) {
	model := readerModel{err: fmt.Errorf("render failed")}
	if got, want := model.View(), "error: render failed\n"; got != want {
		t.Fatalf("View = %q, want %q", got, want)
	}
}

func TestCenteredOffset(t *testing.T) {
	tests := []struct {
		name     string
		line     int
		height   int
		expected int
	}{
		{name: "middle", line: 20, height: 10, expected: 15},
		{name: "near top", line: 3, height: 10, expected: 0},
		{name: "odd height", line: 20, height: 9, expected: 16},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if actual := centeredOffset(test.line, test.height); actual != test.expected {
				t.Fatalf("centeredOffset(%d, %d) = %d, want %d", test.line, test.height, actual, test.expected)
			}
		})
	}
}

func TestReaderExecutesSuccessAndFailure(t *testing.T) {
	markdown := []byte("```bash\nprintf passed\n```\n\n```sh\nprintf failed >&2\nexit 2\n```\n")
	var output bytes.Buffer
	model := newReaderModel(markdown, &output, 80, 30)

	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(readerModel)
	if model.running != 0 {
		t.Fatalf("running = %d, want 0", model.running)
	}
	updated, _ = model.Update(command())
	model = updated.(readerModel)
	if !strings.Contains(model.View(), "success") {
		t.Fatalf("success state missing:\n%s", model.View())
	}
	if !strings.Contains(model.View(), "passed") {
		t.Fatalf("stdout missing:\n%s", model.View())
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = updated.(readerModel)
	updated, command = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(readerModel)
	updated, _ = model.Update(command())
	model = updated.(readerModel)
	if !strings.Contains(model.View(), "failed (exit 2)") {
		t.Fatalf("failure state missing:\n%s", model.View())
	}
	if !strings.Contains(model.View(), "failed") {
		t.Fatalf("stderr missing:\n%s", model.View())
	}
}
