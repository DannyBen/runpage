package cmd

import (
	"bytes"
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
