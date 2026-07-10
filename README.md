# Mob - Markdown Ops Book

![repocard](https://repocard.dannyben.com/svg/mob.svg)

`mob` turns Markdown documents into interactive terminal runbooks. It keeps the
document in one continuous view, lets you move between executable code blocks,
and shows status, output, errors, and exit results directly inside each block.

Opening a document never executes its code automatically.

## Install

The simplest option is with `eget`:

```bash
eget dannyben/mob
```

Additional installation methods, including Go and GitHub Release archives,
are in [INSTALL.md](INSTALL.md).

## Highlights

- Read a complete Markdown document in a terminal-native interactive view.
- Move directly between executable code blocks and run the selected block.
- Keep command status, standard output, standard error, and exit results inline.
- Run every executable block from an explicit working directory when needed.
- Use compact mode for an execution-focused view without surrounding prose.
- Use read-only mode to browse a document without selecting or executing code.
- Render a static, script-friendly view for redirected output.
- Mark individual blocks as display-only with `:noop`.

## Usage

Open a document in the interactive reader:

```bash
mob path/to/runbook.md
```

When no file is provided, Mob looks for `mob.md` and then `README.md` in the
current directory.

Choose a focused mode when needed:

```bash
mob runbook.md --read
mob runbook.md --compact
mob runbook.md --show
mob runbook.md --workdir path/to/project
```

| Mode | Purpose |
| --- | --- |
| default | Read the full document and execute selected blocks |
| `--read`, `-r` | Read the full document without execution |
| `--compact`, `-c` | Show only captioned executable blocks |
| `--show`, `-s` | Render the complete document and exit |

Use `--workdir DIR` (`-w DIR`) to run every executable block from a specific
directory. Without it, commands run from the directory where Mob was started.

Add `:noop` to a fence info string to keep that block display-only.

## Navigation

| Key | Action |
| --- | --- |
| `j` / `k`, arrows | Scroll the document |
| `Page Up` / `Page Down` | Scroll by viewport |
| `Tab` / `Shift+Tab` | Select the next or previous executable block |
| `Ctrl+Up` / `Ctrl+Down` | Select the previous or next executable block |
| `Enter` | Execute the selected block |
| `g` / `G` | Move to the beginning or end |
| `q` / `Esc` | Cancel an active command; press again to quit |

## Examples

The [examples](examples/) folder contains documents for static rendering,
interactive execution, compact mode, and execution results.

```bash
mob examples/showcase.md
mob examples/execution-demo.md --compact
mob examples/release-checklist.md --read
```

## Development

```bash
op check
op mob --help
```
