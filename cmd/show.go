package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

const defaultRenderWidth = 80

func showDocument(markdown []byte, output io.Writer) error {
	rendered := renderMarkdown(markdown, renderWidth(output), colorOutput(output), -1, -1, nil)
	_, err := io.WriteString(output, rendered.text+"\n")
	return err
}

func showRunDocument(markdown []byte, tag string, output io.Writer, compact int, workdir string) error {
	width := renderWidth(output)
	blocks := executableBlocks(string(markdown))
	selected := taggedBlockIndexes(blocks, tag)
	if len(selected) == 0 {
		return fmt.Errorf("no executable code blocks tagged :%s", strings.TrimPrefix(tag, ":"))
	}

	color := colorOutput(output)
	results := make(map[int]executionResult, len(selected))
	summary := runSummary{total: len(selected)}
	emitted := 0
	var runErr error
	lines := strings.Split(string(markdown), "\n")

	for position, index := range selected {
		block := blocks[index]
		prefix := []byte(strings.Join(lines[:min(block.endLine+1, len(lines))], "\n"))
		running := renderReaderMarkdownMode(prefix, width, color, -1, index, results, compact > 0, compact > 1, -1)
		runningLines := strings.Split(running.text, "\n")
		stop := running.blockRanges[index].start
		if err := writeRenderedLines(output, runningLines, emitted, stop); err != nil {
			return err
		}

		message := executeBlock(context.Background(), index, block, width-codeFrameOverhead, workdir)
		results[index] = message.result
		completed := renderReaderMarkdownMode(prefix, width, color, -1, -1, results, compact > 0, compact > 1, -1)
		completedLines := strings.Split(completed.text, "\n")
		if err := writeRenderedLines(output, completedLines, stop, len(completedLines)); err != nil {
			return err
		}
		emitted = len(completedLines)

		if message.result.exitCode != 0 {
			summary.failed = 1
			summary.skipped = len(selected) - position - 1
			runErr = fmt.Errorf("tag :%s failed at code block on line %d with exit code %d", strings.TrimPrefix(tag, ":"), block.line+1, message.result.exitCode)
			break
		}
		summary.succeeded++
	}

	final := renderReaderMarkdownMode(markdown, width, color, -1, -1, results, compact > 0, compact > 1, -1)
	finalLines := strings.Split(final.text, "\n")
	if err := writeRenderedLines(output, finalLines, emitted, len(finalLines)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(output, "\n%s\n", formatRunSummary(summary)); err != nil {
		return err
	}
	return runErr
}

func writeRenderedLines(output io.Writer, lines []string, start, end int) error {
	start = min(max(0, start), len(lines))
	end = min(max(start, end), len(lines))
	if start == end {
		return nil
	}
	_, err := io.WriteString(output, strings.Join(lines[start:end], "\n")+"\n")
	return err
}

func formatRunSummary(summary runSummary) string {
	parts := make([]string, 0, 3)
	if summary.succeeded > 0 {
		parts = append(parts, fmt.Sprintf("%d succeeded", summary.succeeded))
	}
	if summary.failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", summary.failed))
	}
	if summary.skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", summary.skipped))
	}
	return fmt.Sprintf("%d %s: %s", summary.total, plural(summary.total, "command", "commands"), strings.Join(parts, ", "))
}

func plural(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

func colorOutput(output io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	file, ok := output.(*os.File)
	return ok && term.IsTerminal(int(file.Fd()))
}

func renderWidth(output io.Writer) int {
	if file, ok := output.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		if width, _, err := term.GetSize(int(file.Fd())); err == nil && width > 0 {
			return width
		}
	}
	if width, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && width > 0 {
		return width
	}
	return defaultRenderWidth
}
