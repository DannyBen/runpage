# Mob

Markdown Ops Book.

Mob presents Markdown as one continuous terminal document. Shell code blocks
can be selected and executed in place, with their status, stdout, and stderr
kept inside the block's frame.

## Features

- Terminal-native Markdown rendering with wrapped prose and compact tables.
- Keyboard navigation through the complete document.
- Executable `bash`, `sh`, `zsh`, `shell`, and `console` fences.
- Distinct ready, running, successful, and failed block states.
- Inline stdout, stderr, and exit-status results.
- Static rendering for scripts and redirected output.

Opening a document never executes its code automatically.

The static Markdown viewer is available now:

```bash
mob examples/showcase.md --show
```

Open the interactive reader:

```bash
mob examples/showcase.md
```

Open the interactive reader in read-only mode, without selecting or executing
code blocks:

```bash
mob examples/showcase.md --read
```

Navigation keys:

| Key | Action |
| --- | --- |
| `j` / `k`, arrows | Scroll the document |
| `Page Up` / `Page Down` | Scroll by viewport |
| `Tab` / `Shift+Tab` | Select the next or previous executable block |
| `Enter` | Execute the selected block |
| `g` / `G` | Move to the beginning or end |
| `q` | Quit and cancel an active command |

Add `:noop` to a fence info string to keep a shell block display-only:

````markdown
```bash :noop
git push origin main
```
````

## Usage

```bash
mob [FILE] [options]
```

When `FILE` is omitted, Mob looks for `mob.md` and then `README.md`.
Interactive reading with executable blocks is the default mode. Use `--read`
for a read-only interactive view.

```bash
mob README.md
mob README.md --show
mob README.md --list
```

`--show` renders the complete document and exits. `--list` is reserved for the
execution index and is not implemented yet.

## Development

```bash
op check
op mob --help
```

## Install

### With eget

```bash
eget DannyBen/mob --to ~/bin/mob
```

### From GitHub Releases

Download the archive for your operating system and CPU from the Releases page,
extract it, and put `mob` somewhere on your `PATH`.
