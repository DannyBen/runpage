package cmd

import (
	"strings"
	"testing"
)

func TestRenderMarkdownUsesSimpleTerminalStyles(t *testing.T) {
	markdown := []byte("# Release\n\nUse **care** here.\n\n> Important note.\n\n```bash\nop check\n```\n")
	rendered := renderMarkdown(markdown, 80, true, 0, -1, nil)

	if !strings.Contains(rendered.text, ansiBold+"Release"+ansiReset) {
		t.Fatalf("heading is not bold:\n%q", rendered.text)
	}
	if !strings.Contains(rendered.text, "▌ Important note.") {
		t.Fatalf("blockquote marker missing:\n%q", rendered.text)
	}
	if !strings.Contains(rendered.text, ansiBold+"care"+ansiReset) {
		t.Fatalf("emphasis is not bold:\n%q", rendered.text)
	}
	if !strings.Contains(rendered.text, ansiBoldBlue+"▌"+ansiReset) {
		t.Fatalf("focused executable has no bold blue rail:\n%q", rendered.text)
	}
	if !strings.Contains(rendered.text, ansiCyan+"│ "+ansiReset+"op check") {
		t.Fatalf("focused code does not retain cyan border and normal text:\n%q", rendered.text)
	}
}

func TestRenderMarkdownColorsExecutionState(t *testing.T) {
	markdown := []byte("```bash\nop check\n```\n\n```sh\nop test\n```\n")
	results := map[int]executionResult{
		0: {stdout: "passed\n"},
		1: {exitCode: 2, stderr: "failed\n"},
	}
	rendered := renderMarkdown(markdown, 80, true, -1, -1, results)

	if !strings.Contains(rendered.text, ansiGreen+"│ "+ansiReset+"op check") {
		t.Fatalf("successful block border is not green:\n%q", rendered.text)
	}
	if !strings.Contains(rendered.text, ansiRed+"│ "+ansiReset+"op test") {
		t.Fatalf("failed block border is not red:\n%q", rendered.text)
	}
	if !strings.Contains(rendered.text, ansiGreen+"│ "+ansiReset+"passed") {
		t.Fatalf("indented stdout missing:\n%q", rendered.text)
	}
	if !strings.Contains(rendered.text, ansiRed+"│ "+ansiReset+"failed") {
		t.Fatalf("red stderr missing:\n%q", rendered.text)
	}
}

func TestRenderMarkdownColorsRunningBlock(t *testing.T) {
	markdown := []byte("```bash\nsleep 1\n```\n")
	rendered := renderMarkdown(markdown, 80, true, -1, 0, nil)
	if !strings.Contains(rendered.text, ansiBold+ansiYellow+" running "+ansiReset) {
		t.Fatalf("running footer is not bold yellow:\n%q", rendered.text)
	}
}
