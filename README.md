# Runpage - Interactive Command Pages

![repocard](https://repocard.dannyben.com/svg/runpage.svg)

`runpage` turns Markdown documents into interactive command pages. It keeps the
document in one continuous view, lets you move between executable code blocks,
and shows status, output, errors, and exit results directly inside each block.

Opening a document never executes its code automatically.

---
![](support/demo.gif)
---

## Install

The simplest option is with `eget`:

```bash
eget dannyben/runpage
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
runpage path/to/page.md
```

When no file is provided, Runpage looks for `runpage.md` and then `README.md` in the
current directory.

Choose a focused mode when needed:

```bash
runpage release.md --read
runpage release.md --compact
runpage release.md -cc
runpage release.md --show
runpage release.md --workdir path/to/project
```

| Mode | Purpose |
| --- | --- |
| default | Read the full document and execute selected blocks |
| `--read`, `-r` | Read the full document without execution |
| `--compact`, `-c` | Show relevant headings and executable blocks |
| `-cc` | Also hide executable source |
| `--show`, `-s` | Render the complete document and exit |

Use `--workdir DIR` (`-w DIR`) to run every executable block from a specific
directory. Without it, commands run from the directory where Runpage was started.
Use the reserved `self` path root to resolve a workdir from the directory
containing the Markdown document:

```bash
runpage path/to/page.md --workdir self
runpage path/to/page.md --workdir self/tasks
```

`self` selects the document directory, while `self/tasks` selects its `tasks`
subdirectory and `self/..` selects its parent. Use `./self` to refer to a real
launch-relative directory named `self`.

Add `:noop` to a fence info string to keep that block display-only.

Code frames are untitled by default. Add a short label after the fence language
when a frame needs its own title:

````markdown
```bash check
test "$(git branch --show-current)" = master
```
````

Multiple plain words form one label. Tokens beginning with `:` are Runpage
directives and are not included in the title, so labels compose with `:noop`:

````markdown
```bash example :noop
echo "display only"
```
````

Markdown headings remain document captions outside the code frame. H1 uses a
double ruler, H2 uses a single ruler, and H3 through H6 retain their Markdown
`###` through `######` prefixes. Compact mode preserves the relevant heading
branch while omitting unrelated headings and prose.

## Document Values

Pass `KEY:VALUE` arguments after the document to inject values before Runpage
renders or executes it:

```bash
runpage release.md version:1.0.0
```

Both `$KEY` and `{{ KEY }}` placeholders are replaced throughout the document,
including executable blocks:

```markdown
# Release $version

Deploying version {{ version }}.
```

Repeat the argument to provide more values. Runpage splits each argument at its
first colon, so values may contain additional colons. Values are inserted
literally in a single pass and are not escaped for the surrounding code.

## Document Configuration

A document can declare Runpage configuration in YAML front matter:

```yaml
---
runpage:
  required: [version]
  dependencies: [git, curl]
  workdir: self
---
```

- `required` lists non-empty document values that must be provided.
- `dependencies` lists executables required by execution-enabled modes.
- `workdir` sets the directory used to execute blocks.

Missing required values prevent the default, compact, and static modes from
opening the document. `--read` bypasses value and dependency requirements so
the unexpanded document can still be inspected safely. Dependency checks apply
only to execution-enabled modes. A CLI `--workdir` value overrides front matter.
In either location, `self` and `self/...` resolve from the directory containing
the document; other relative paths resolve from the directory where Runpage was
started.

Front matter is configuration only and is not included in rendered output.

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

The [showcase](examples/showcase.md) is a safe, self-contained demonstration of
rich Markdown, executable blocks, captured output, intentional failure, and
display-only language examples.

```bash
runpage examples/showcase.md
```

## Development

```bash
op check
op runpage --help
```
