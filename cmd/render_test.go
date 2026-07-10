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

func TestRenderMarkdownDocumentStructures(t *testing.T) {
	markdown := []byte(strings.Join([]string{
		"# Guide",
		"",
		"1. First",
		"2. Second",
		"",
		"- Alpha",
		"- Beta",
		"",
		"    plain code",
		"",
		"---",
		"",
		"| Name | State |",
		"| --- | --- |",
		"| Build | Ready |",
	}, "\n"))
	rendered := renderMarkdown(markdown, 40, false, -1, -1, nil).text

	for _, want := range []string{"1. First", "2. Second", "• Alpha", "• Beta", "plain code", "───", "Name", "Build", "Ready"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("rendered document missing %q:\n%s", want, rendered)
		}
	}
}

func TestRenderMarkdownIndentedCodeBlock(t *testing.T) {
	rendered := renderMarkdown([]byte("    first line\n    second line\n"), 40, false, -1, -1, nil).text
	if !strings.Contains(rendered, "first line\nsecond line") {
		t.Fatalf("indented code block was not preserved:\n%s", rendered)
	}
}

func TestWrappingHelpers(t *testing.T) {
	if got := wrapText("one two three", 7, "> "); strings.Join(got, "|") != "> one two|> three" {
		t.Fatalf("wrapText = %#v", got)
	}
	if got := wrapText("", 0, ""); got != nil {
		t.Fatalf("wrapText empty = %#v", got)
	}
	if got := wrapPreserved("", 4); len(got) != 1 || got[0] != "" {
		t.Fatalf("wrapPreserved empty = %#v", got)
	}
	if got := wrapPreserved("abcdef", 3); strings.Join(got, "|") != "abc|def" {
		t.Fatalf("wrapPreserved = %#v", got)
	}
}
