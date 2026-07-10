package cmd

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"golang.org/x/term"
)

const defaultRenderWidth = 80

func showDocument(path string, output io.Writer) error {
	markdown, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read document %s: %w", path, err)
	}

	rendered := renderMarkdown(markdown, renderWidth(output), colorOutput(output), -1, -1, nil)
	_, err = io.WriteString(output, rendered.text+"\n")
	return err
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
