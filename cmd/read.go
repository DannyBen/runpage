package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

type readerModel struct {
	markdown   []byte
	blocks     []executableBlock
	results    map[int]executionResult
	running    int
	cancel     context.CancelFunc
	focused    int
	output     io.Writer
	viewport   viewport.Model
	blockLines []int
	err        error
}

func readDocument(path string, input io.Reader, output io.Writer) error {
	if !terminalOutput(output) {
		return fmt.Errorf("interactive reader requires a terminal (use --show for redirected output)")
	}
	markdown, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read document %s: %w", path, err)
	}

	model := newReaderModel(markdown, output, renderWidth(output), 24)
	program := tea.NewProgram(model, tea.WithInput(input), tea.WithOutput(output), tea.WithAltScreen())
	final, err := program.Run()
	if err != nil {
		return fmt.Errorf("run interactive reader: %w", err)
	}
	if result, ok := final.(readerModel); ok && result.err != nil {
		return result.err
	}
	return nil
}

func newReaderModel(markdown []byte, output io.Writer, width, height int) readerModel {
	model := readerModel{
		markdown: markdown,
		blocks:   executableBlocks(string(markdown)),
		results:  make(map[int]executionResult),
		running:  -1,
		output:   output,
		viewport: viewport.New(width, max(1, height-1)),
	}
	model.render(false)
	return model
}

func (model readerModel) Init() tea.Cmd { return nil }

func (model readerModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case executionFinishedMsg:
		model.results[message.index] = message.result
		model.running = -1
		model.cancel = nil
		model.render(false)
		return model, nil
	case tea.KeyMsg:
		switch message.String() {
		case "q", "ctrl+c":
			if model.cancel != nil {
				model.cancel()
			}
			return model, tea.Quit
		case "tab", "ctrl+down":
			model.focus(1)
			return model, nil
		case "shift+tab", "ctrl+up":
			model.focus(-1)
			return model, nil
		case "enter":
			return model, model.startExecution()
		case "g":
			model.viewport.GotoTop()
			return model, nil
		case "G":
			model.viewport.GotoBottom()
			return model, nil
		}
	case tea.WindowSizeMsg:
		offset := model.viewport.YOffset
		model.viewport.Width = message.Width
		model.viewport.Height = max(1, message.Height-1)
		model.render(false)
		model.viewport.SetYOffset(offset)
		return model, nil
	}

	var command tea.Cmd
	model.viewport, command = model.viewport.Update(message)
	return model, command
}

func (model readerModel) View() string {
	if model.err != nil {
		return fmt.Sprintf("error: %v\n", model.err)
	}
	status := "j/k scroll  •  Tab/Shift+Tab blocks  •  Enter execute  •  q quit"
	if len(model.blocks) > 0 {
		status = fmt.Sprintf("block %d/%d  •  %s", model.focused+1, len(model.blocks), status)
	}
	return model.viewport.View() + "\n" + status
}

func (model *readerModel) focus(delta int) {
	if len(model.blocks) == 0 {
		return
	}
	model.focused = (model.focused + delta + len(model.blocks)) % len(model.blocks)
	model.render(true)
}

func (model *readerModel) startExecution() tea.Cmd {
	if len(model.blocks) == 0 || model.running >= 0 {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	model.cancel = cancel
	model.running = model.focused
	index := model.focused
	block := model.blocks[index]
	model.render(true)
	columns := max(20, model.viewport.Width-codeFrameOverhead)
	return func() tea.Msg { return executeBlock(ctx, index, block, columns) }
}

func (model *readerModel) render(scrollToFocus bool) {
	rendered := renderMarkdown(model.markdown, model.viewport.Width, colorOutput(model.output), model.focused, model.running, model.results)
	model.err = nil
	model.viewport.SetContent(rendered.text)
	model.blockLines = rendered.blockLines
	if scrollToFocus {
		model.viewport.SetYOffset(centeredOffset(model.blockLines[model.focused], model.viewport.Height))
	}
}

func centeredOffset(line, viewportHeight int) int {
	return max(0, line-viewportHeight/2)
}

func terminalOutput(output io.Writer) bool {
	file, ok := output.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}
