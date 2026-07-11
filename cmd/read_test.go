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

func TestReaderLineScrollingMovesFocusAtOneThirdAnchor(t *testing.T) {
	markdown := []byte(strings.Join([]string{
		"# Guide",
		"",
		"```sh",
		"printf first",
		"printf second",
		"```",
		"",
		"Some explanation between the commands.",
		"",
		"More explanation between the commands.",
		"",
		"```sh",
		"printf next",
		"```",
		"",
		"Enough trailing prose to keep scrolling possible.",
		"",
		"One more trailing paragraph.",
	}, "\n"))

	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyDown},
		{Type: tea.KeyRunes, Runes: []rune{'j'}},
	} {
		t.Run(key.String(), func(t *testing.T) {
			var output bytes.Buffer
			model := newReaderModel(markdown, &output, 80, 8)
			if len(model.blockRanges) != 2 {
				t.Fatalf("block ranges = %#v", model.blockRanges)
			}
			anchorRow := model.viewport.Height / 3
			startOffset := model.blockRanges[1].start - anchorRow - 1
			model.viewport.SetYOffset(startOffset)

			updated, _ := model.Update(key)
			model = updated.(readerModel)

			if model.focused != 1 {
				t.Fatalf("focused = %d, want 1 at anchor line %d", model.focused, model.viewport.YOffset+anchorRow)
			}
			if model.viewport.YOffset != startOffset+1 {
				t.Fatalf("scroll offset = %d, want %d; focus change recentered the viewport", model.viewport.YOffset, startOffset+1)
			}
		})
	}
}

func TestReaderUpwardScrollingMovesFocusWhenBlockCrossesAnchor(t *testing.T) {
	markdown := []byte(strings.Join([]string{
		"# Guide",
		"",
		"```sh",
		"printf first",
		"printf second",
		"```",
		"",
		"Paragraph one.",
		"",
		"Paragraph two.",
		"",
		"```sh",
		"printf next",
		"```",
		"",
		"Trailing paragraph.",
	}, "\n"))
	var output bytes.Buffer
	model := newReaderModel(markdown, &output, 80, 8)
	model.focused = 1
	model.render(false)
	anchorRow := model.viewport.Height / 3
	startOffset := model.blockRanges[0].end - anchorRow + 1
	model.viewport.SetYOffset(startOffset)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(readerModel)

	if model.focused != 0 {
		t.Fatalf("focused = %d, want 0 at anchor line %d", model.focused, model.viewport.YOffset+anchorRow)
	}
	if model.viewport.YOffset != startOffset-1 {
		t.Fatalf("scroll offset = %d, want %d; focus change recentered the viewport", model.viewport.YOffset, startOffset-1)
	}
}

func TestReaderScrollKeyWithoutMovementKeepsFocus(t *testing.T) {
	markdown := []byte("```sh\nprintf first\nprintf second\n```\n\n```sh\nprintf next\n```\n")
	var output bytes.Buffer
	model := newReaderModel(markdown, &output, 80, 6)
	model.focused = 1
	model.render(false)
	model.viewport.GotoTop()

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyUp})
	model = updated.(readerModel)

	if model.viewport.YOffset != 0 || model.focused != 1 {
		t.Fatalf("top-boundary scroll = offset:%d focused:%d, want offset:0 focused:1", model.viewport.YOffset, model.focused)
	}
}

func TestReadOnlyReaderDoesNotSelectOrExecuteBlocks(t *testing.T) {
	markdown := []byte("# Guide\n\n```bash\nprintf should-not-run\n```\n")
	var output bytes.Buffer
	model := newReaderModelWithMode(markdown, &output, 80, 20, true, 0, "")

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

func TestReaderTogglesKeyBindingHelp(t *testing.T) {
	markdown := []byte("# Guide\n\n```bash\nop check\n```\n")
	var output bytes.Buffer
	model := newReaderModel(markdown, &output, 80, 20)

	updated, command := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	model = updated.(readerModel)
	if command != nil || !model.showHelp {
		t.Fatalf("opening help = showHelp:%v command:%v", model.showHelp, command)
	}
	for _, want := range []string{"Runpage key bindings", "Tab / Shift+Tab", "Enter", "?/Esc close"} {
		if !strings.Contains(model.View(), want) {
			t.Errorf("help view missing %q:\n%s", want, model.View())
		}
	}

	updated, command = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(readerModel)
	if command != nil || model.showHelp {
		t.Fatalf("closing help = showHelp:%v command:%v", model.showHelp, command)
	}
	if !strings.Contains(model.View(), "ready to execute") {
		t.Fatalf("document did not return after closing help:\n%s", model.View())
	}
}

func TestReadOnlyKeyBindingHelpOmitsExecutionControls(t *testing.T) {
	var output bytes.Buffer
	model := newReaderModelWithMode([]byte("# Guide\n"), &output, 80, 20, true, 0, "")
	model.showHelp = true

	if strings.Contains(model.View(), "Select next") || strings.Contains(model.View(), "Execute selected") {
		t.Fatalf("read-only help includes execution controls:\n%s", model.View())
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

func TestStartingExecutionOnlyScrollsBlockIntoView(t *testing.T) {
	markdown := []byte(strings.Join([]string{
		"# Guide",
		"",
		"Intro one.",
		"",
		"Intro two.",
		"",
		"Intro three.",
		"",
		"```sh",
		"printf first",
		"printf second",
		"```",
		"",
		"Trailing one.",
		"",
		"Trailing two.",
		"",
		"Trailing three.",
	}, "\n"))

	tests := []struct {
		name       string
		startAt    func(lineRange, int) int
		expectedAt func(lineRange, int) int
	}{
		{
			name:       "fully visible",
			startAt:    func(block lineRange, _ int) int { return block.start - 1 },
			expectedAt: func(block lineRange, _ int) int { return block.start - 1 },
		},
		{
			name:       "hidden above",
			startAt:    func(block lineRange, _ int) int { return block.start + 1 },
			expectedAt: func(block lineRange, _ int) int { return block.start },
		},
		{
			name:       "hidden below",
			startAt:    func(block lineRange, height int) int { return block.end - height },
			expectedAt: func(block lineRange, height int) int { return block.end - height + 1 },
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			model := newReaderModel(markdown, &output, 80, 8)
			block := model.blockRanges[0]
			model.viewport.SetYOffset(test.startAt(block, model.viewport.Height))

			if command := model.startExecution(); command == nil {
				t.Fatal("startExecution returned no command")
			}
			defer model.cancel()

			if got, want := model.viewport.YOffset, test.expectedAt(block, model.viewport.Height); got != want {
				t.Fatalf("scroll offset = %d, want %d", got, want)
			}
		})
	}
}

func TestStartingOversizedVisibleBlockKeepsScrollPosition(t *testing.T) {
	markdown := []byte("# Guide\n\n```sh\nline1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\n```\n\nTrailing prose.\n")
	var output bytes.Buffer
	model := newReaderModel(markdown, &output, 80, 6)
	startOffset := model.blockRanges[0].start + 2
	model.viewport.SetYOffset(startOffset)

	if command := model.startExecution(); command == nil {
		t.Fatal("startExecution returned no command")
	}
	defer model.cancel()

	if model.viewport.YOffset != startOffset {
		t.Fatalf("oversized visible block moved from offset %d to %d", startOffset, model.viewport.YOffset)
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

func TestReaderCancelsBeforeQuitting(t *testing.T) {
	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'q'}},
		{Type: tea.KeyEsc},
	} {
		t.Run(key.String(), func(t *testing.T) {
			var output bytes.Buffer
			model := newReaderModel([]byte("```sh\nsleep 10\n```\n"), &output, 80, 20)
			cancelled := false
			model.running = 0
			model.cancel = func() { cancelled = true }

			updated, command := model.Update(key)
			model = updated.(readerModel)
			if command != nil {
				t.Fatal("first cancel key quit the reader")
			}
			if !cancelled || !model.cancelling {
				t.Fatalf("cancel state = cancelled:%v cancelling:%v", cancelled, model.cancelling)
			}
			if !strings.Contains(model.View(), "cancelling") {
				t.Fatalf("cancelling status missing:\n%s", model.View())
			}

			_, command = model.Update(key)
			if command == nil {
				t.Fatal("second cancel key did not quit")
			}
		})
	}
}

func TestReaderShowsCancelledResult(t *testing.T) {
	var output bytes.Buffer
	model := newReaderModel([]byte("```sh\nsleep 10\n```\n"), &output, 80, 20)
	model.running = 0
	model.cancelling = true
	updated, _ := model.Update(executionFinishedMsg{index: 0, result: executionResult{exitCode: -1, cancelled: true}})
	model = updated.(readerModel)

	if model.running != -1 || model.cancelling {
		t.Fatalf("execution state was not cleared: %#v", model)
	}
	if !strings.Contains(model.View(), "cancelled") {
		t.Fatalf("cancelled result missing:\n%s", model.View())
	}
}
