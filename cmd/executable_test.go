package cmd

import (
	"strings"
	"testing"
)

func TestExecutableBlocks(t *testing.T) {
	markdown := strings.Join([]string{
		"# Guide",
		"",
		"```bash",
		"op check",
		"```",
		"",
		"```yaml",
		"mode: read",
		"```",
		"",
		"~~~sh :noop",
		"rm something",
		"~~~",
		"",
		"```zsh",
		"op test",
		"```",
	}, "\n")

	blocks := executableBlocks(markdown)

	if len(blocks) != 2 {
		t.Fatalf("len(blocks) = %d, want 2", len(blocks))
	}
	if blocks[0].language != "bash" || blocks[0].line != 2 {
		t.Fatalf("blocks[0] = %#v", blocks[0])
	}
	if blocks[1].language != "zsh" || blocks[1].line != 14 {
		t.Fatalf("blocks[1] = %#v", blocks[1])
	}
}

func TestShellAndConsoleBlocksAreExecutable(t *testing.T) {
	markdown := "```shell\nprintf shell\n```\n\n```console\nprintf console\n```\n"
	blocks := executableBlocks(markdown)
	if len(blocks) != 2 {
		t.Fatalf("len(blocks) = %d, want 2", len(blocks))
	}
	if blocks[0].command != "printf shell" || blocks[1].command != "printf console" {
		t.Fatalf("commands = %q, %q", blocks[0].command, blocks[1].command)
	}
}

func TestPythonAndRubyBlocksAreExecutable(t *testing.T) {
	markdown := strings.Join([]string{
		"```python",
		"print('python')",
		"```",
		"```py",
		"print('py')",
		"```",
		"```ruby",
		"puts 'ruby'",
		"```",
		"```rb",
		"puts 'rb'",
		"```",
	}, "\n")
	blocks := executableBlocks(markdown)
	if len(blocks) != 4 {
		t.Fatalf("blocks = %#v, want four executable blocks", blocks)
	}
	for index, language := range []string{"python", "py", "ruby", "rb"} {
		if blocks[index].language != language {
			t.Fatalf("blocks[%d].language = %q, want %q", index, blocks[index].language, language)
		}
	}
}

func TestFenceParsingEdges(t *testing.T) {
	markdown := strings.Join([]string{
		"    ```bash",
		"ignored",
		"    ```",
		"~~~BASH",
		"printf tilde",
		"~~~~",
		"````sh",
		"printf long",
		"```",
		"````",
		"```bash noop",
		"ignored",
		"```",
		"```",
		"ignored",
		"```",
	}, "\n")

	blocks := executableBlocks(markdown)
	if len(blocks) != 2 {
		t.Fatalf("blocks = %#v, want two executable blocks", blocks)
	}
	if blocks[0].language != "bash" || blocks[0].command != "printf tilde" {
		t.Fatalf("blocks[0] = %#v", blocks[0])
	}
	if blocks[1].language != "sh" || !strings.Contains(blocks[1].command, "```") {
		t.Fatalf("blocks[1] = %#v", blocks[1])
	}
}
