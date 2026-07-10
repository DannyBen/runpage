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

func TestRenderMarkdownCancellationStates(t *testing.T) {
	markdown := []byte("```sh\nsleep 10\n```\n")
	cancelling := renderReaderMarkdown(markdown, 80, true, 0, 0, nil, false, 0)
	if !strings.Contains(cancelling.text, ansiBold+ansiYellow+" cancelling "+ansiReset) {
		t.Fatalf("cancelling footer is not bold yellow:\n%q", cancelling.text)
	}

	results := map[int]executionResult{0: {exitCode: -1, cancelled: true}}
	cancelled := renderMarkdown(markdown, 80, true, 0, -1, results)
	if !strings.Contains(cancelled.text, ansiBold+ansiYellow+" cancelled "+ansiReset) {
		t.Fatalf("cancelled footer is not bold yellow:\n%q", cancelled.text)
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

func TestRenderMarkdownHeadingHierarchy(t *testing.T) {
	markdown := []byte(strings.Join([]string{
		"# Document",
		"",
		"## Section",
		"",
		"### Topic",
		"",
		"#### Detail",
		"",
		"##### Note",
		"",
		"###### Fine print",
	}, "\n"))
	rendered := renderMarkdown(markdown, 30, false, -1, -1, nil).text

	for _, want := range []string{
		"Document\n" + strings.Repeat("═", 30),
		"Section\n" + strings.Repeat("─", 30),
		"### Topic",
		"#### Detail",
		"##### Note",
		"###### Fine print",
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("heading hierarchy missing %q:\n%s", want, rendered)
		}
	}
}

func TestRenderMarkdownIndentedCodeBlock(t *testing.T) {
	rendered := renderMarkdown([]byte("    first line\n    second line\n"), 40, false, -1, -1, nil).text
	if !strings.Contains(rendered, "first line\nsecond line") {
		t.Fatalf("indented code block was not preserved:\n%s", rendered)
	}
}

func TestRenderMarkdownCompactShowsOnlyCaptionedExecutableBlocks(t *testing.T) {
	markdown := []byte(strings.Join([]string{
		"# Build",
		"",
		"Explanatory prose.",
		"",
		"```bash",
		"go test ./...",
		"```",
		"",
		"## Example",
		"",
		"```yaml",
		"mode: ignored",
		"```",
		"",
		"## Release",
		"",
		"```sh :noop",
		"echo ignored",
		"```",
		"",
		"```sh",
		"goreleaser release",
		"```",
	}, "\n"))
	rendered := renderMarkdownMode(markdown, 80, false, 0, -1, nil, true)

	for _, want := range []string{"Build", "go test ./...", "Release", "goreleaser release"} {
		if !strings.Contains(rendered.text, want) {
			t.Errorf("compact document missing %q:\n%s", want, rendered.text)
		}
	}
	for _, unwanted := range []string{"Explanatory prose", "Example", "mode: ignored", "echo ignored"} {
		if strings.Contains(rendered.text, unwanted) {
			t.Errorf("compact document contains %q:\n%s", unwanted, rendered.text)
		}
	}
	if len(rendered.blockLines) != 2 {
		t.Fatalf("blockLines = %#v, want two executable blocks", rendered.blockLines)
	}
}

func TestRenderMarkdownCompactPreservesHeadingBranchOnce(t *testing.T) {
	markdown := []byte(strings.Join([]string{
		"# Runbook",
		"",
		"## Verify",
		"",
		"### Local",
		"",
		"```bash check",
		"op check",
		"```",
		"",
		"```bash check",
		"op test",
		"```",
		"",
		"## Notes",
		"",
		"Nothing executable here.",
		"",
		"## Publish",
		"",
		"```bash perform",
		"op release",
		"```",
	}, "\n"))
	rendered := renderMarkdownMode(markdown, 50, false, 0, -1, nil, true).text

	for _, heading := range []string{"Runbook", "Verify", "### Local", "Publish"} {
		if strings.Count(rendered, heading) != 1 {
			t.Errorf("heading %q was not rendered exactly once:\n%s", heading, rendered)
		}
	}
	if strings.Contains(rendered, "Notes") || strings.Contains(rendered, "Nothing executable") {
		t.Fatalf("unrelated compact branch was rendered:\n%s", rendered)
	}
	if strings.Count(rendered, "┌─ check ") != 2 || !strings.Contains(rendered, "┌─ perform ") {
		t.Fatalf("fence labels missing:\n%s", rendered)
	}
}

func TestRenderMarkdownFenceLabelAndCompactHeading(t *testing.T) {
	markdown := []byte("# Long descriptive heading\n\n```bash check\nop check\n```\n")
	rendered := renderMarkdownMode(markdown, 80, false, 0, -1, nil, true)

	if !strings.Contains(rendered.text, "┌─ check ") {
		t.Fatalf("custom fence label missing:\n%s", rendered.text)
	}
	if !strings.Contains(rendered.text, "Long descriptive heading") {
		t.Fatalf("compact heading missing:\n%s", rendered.text)
	}
}

func TestRenderMarkdownDoesNotInferFenceLabel(t *testing.T) {
	markdown := []byte("## Descriptive heading\n\n```bash\nop check\n```\n")
	rendered := renderMarkdown(markdown, 80, false, 0, -1, nil)
	if strings.Contains(rendered.text, "┌─ Descriptive heading ") {
		t.Fatalf("heading was reused as a fence label:\n%s", rendered.text)
	}
	if !strings.Contains(rendered.text, "Descriptive heading\n"+strings.Repeat("─", 80)) {
		t.Fatalf("heading caption missing:\n%s", rendered.text)
	}
}

func TestRenderMarkdownDisplayOnlyFenceLabel(t *testing.T) {
	markdown := []byte("# Guide\n\n```bash example :noop\necho example\n```\n")
	rendered := renderMarkdown(markdown, 80, false, -1, -1, nil)
	if !strings.Contains(rendered.text, "┌─ example ") || !strings.Contains(rendered.text, "display only") {
		t.Fatalf("display-only label missing:\n%s", rendered.text)
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
