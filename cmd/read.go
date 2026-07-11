package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"
)

const viewportBottomMargin = 4

type readerModel struct {
	markdown    []byte
	blocks      []executableBlock
	results     map[int]executionResult
	running     int
	cancel      context.CancelFunc
	focused     int
	output      io.Writer
	viewport    viewport.Model
	blockLines  []int
	blockRanges []lineRange
	readOnly    bool
	compact     int
	workdir     string
	cancelling  bool
	showHelp    bool
	err         error
}

func readDocument(markdown []byte, input io.Reader, output io.Writer, readOnly bool, compact int, workdir string) error {
	if !terminalOutput(output) {
		return fmt.Errorf("interactive reader requires a terminal (use --show for redirected output)")
	}

	model := newReaderModelWithMode(markdown, output, renderWidth(output), 24, readOnly, compact, workdir)
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
	return newReaderModelWithMode(markdown, output, width, height, false, 0, "")
}

func newReaderModelWithMode(markdown []byte, output io.Writer, width, height int, readOnly bool, compact int, workdir string) readerModel {
	focused := 0
	if readOnly {
		focused = -1
	}
	model := readerModel{
		markdown: markdown,
		blocks:   executableBlocks(string(markdown)),
		results:  make(map[int]executionResult),
		running:  -1,
		focused:  focused,
		readOnly: readOnly,
		compact:  compact,
		workdir:  workdir,
		output:   output,
		viewport: viewport.New(width, max(1, height-1)),
	}
	model.render(false)
	return model
}

func (model readerModel) Init() tea.Cmd { return nil }

func (model readerModel) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	followScroll := false
	switch message := message.(type) {
	case executionFinishedMsg:
		model.results[message.index] = message.result
		model.running = -1
		model.cancel = nil
		model.cancelling = false
		model.render(false)
		return model, nil
	case tea.KeyMsg:
		key := message.String()
		followScroll = !model.showHelp && (key == "up" || key == "down" || key == "j" || key == "k")
		switch key {
		case "?":
			model.showHelp = !model.showHelp
			return model, nil
		case "esc":
			if model.showHelp {
				model.showHelp = false
				return model, nil
			}
			fallthrough
		case "q":
			if model.running >= 0 && !model.cancelling {
				if model.cancel != nil {
					model.cancel()
				}
				model.cancelling = true
				model.render(false)
				return model, nil
			}
			return model, tea.Quit
		case "ctrl+c":
			if model.cancel != nil {
				model.cancel()
			}
			return model, tea.Quit
		case "tab", "ctrl+down":
			if !model.readOnly {
				model.focus(1)
			}
			return model, nil
		case "shift+tab", "ctrl+up":
			if !model.readOnly {
				model.focus(-1)
			}
			return model, nil
		case "enter":
			if model.readOnly {
				return model, nil
			}
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
	offset := model.viewport.YOffset
	model.viewport, command = model.viewport.Update(message)
	if followScroll && model.viewport.YOffset != offset {
		model.focusAtScrollAnchor()
	}
	return model, command
}

func (model readerModel) View() string {
	if model.err != nil {
		return fmt.Sprintf("error: %v\n", model.err)
	}
	if model.showHelp {
		return model.helpView()
	}
	status := "j/k scroll  •  Tab/Shift+Tab blocks  •  Enter execute  •  ? help  •  q/Esc quit"
	if model.readOnly {
		status = "read only  •  j/k scroll  •  ? help  •  q/Esc quit"
		return model.viewport.View() + "\n" + status
	}
	if len(model.blocks) > 0 {
		status = fmt.Sprintf("block %d/%d  •  %s", model.focused+1, len(model.blocks), status)
	}
	if model.running >= 0 {
		status = "running  •  q/Esc cancel"
	}
	if model.cancelling {
		status = "cancelling  •  q/Esc again to quit"
	}
	if model.compact > 0 {
		status = "compact  •  " + status
	}
	return model.viewport.View() + "\n" + status
}

func (model readerModel) helpView() string {
	lines := []string{
		"Runpage key bindings",
		"",
		"j / k, ↑ / ↓       Scroll",
		"PgUp / PgDn         Scroll one page",
		"g / G               Go to top / bottom",
	}
	if !model.readOnly {
		lines = append(lines,
			"Tab / Shift+Tab     Select next / previous block",
			"Ctrl+↓ / Ctrl+↑     Select next / previous block",
			"Enter               Execute selected block",
		)
	}
	lines = append(lines,
		"?                   Close help",
		"Esc                 Close help",
		"q                   Quit (or cancel running block)",
		"Ctrl+C              Quit immediately",
	)
	help := viewport.New(model.viewport.Width, model.viewport.Height)
	help.SetContent(strings.Join(lines, "\n"))
	return help.View() + "\nhelp  •  ?/Esc close  •  q quit"
}

func (model *readerModel) focus(delta int) {
	if len(model.blocks) == 0 {
		return
	}
	model.focused = (model.focused + delta + len(model.blocks)) % len(model.blocks)
	model.render(true)
}

func (model *readerModel) focusAtScrollAnchor() {
	if model.readOnly || len(model.blockRanges) == 0 {
		return
	}
	anchor := model.viewport.YOffset + model.viewport.Height/3
	for index, block := range model.blockRanges {
		if anchor < block.start || anchor > block.end || index == model.focused {
			continue
		}
		offset := model.viewport.YOffset
		model.focused = index
		model.render(false)
		model.viewport.SetYOffset(offset)
		return
	}
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
	model.render(false)
	model.scrollBlockIntoView(index)
	columns := max(20, model.viewport.Width-codeFrameOverhead)
	return func() tea.Msg { return executeBlock(ctx, index, block, columns, model.workdir) }
}

func (model *readerModel) scrollBlockIntoView(index int) {
	if index < 0 || index >= len(model.blockRanges) {
		return
	}
	block := model.blockRanges[index]
	top := model.viewport.YOffset
	bottom := top + model.viewport.Height - 1
	if block.start >= top && block.end <= bottom {
		return
	}
	if block.end-block.start+1 > model.viewport.Height && block.start <= bottom && block.end >= top {
		return
	}
	if block.start < top {
		model.viewport.SetYOffset(block.start)
		return
	}
	model.viewport.SetYOffset(block.end - model.viewport.Height + 1)
}

func (model *readerModel) render(scrollToFocus bool) {
	cancelling := -1
	if model.cancelling {
		cancelling = model.running
	}
	rendered := renderReaderMarkdownMode(model.markdown, model.viewport.Width, colorOutput(model.output), model.focused, model.running, model.results, model.compact > 0, model.compact > 1, cancelling)
	model.err = nil
	model.viewport.SetContent(rendered.text + strings.Repeat("\n", viewportBottomMargin))
	model.blockLines = rendered.blockLines
	model.blockRanges = rendered.blockRanges
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
