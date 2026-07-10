# Build and verify Runpage

Runpage keeps prose and commands together in one continuous terminal document.
The presentation is intentionally quiet so the instructions remain the focus.

> Opening a document never executes its code automatically.

## Install dependencies

Download the Go modules used by the project:

```bash
go mod download
```

## Run the checks

The regular project check runs formatting, vetting, and tests.

```bash
op check
```

This block is intentionally shown but excluded from execution:

```bash :noop
git push
```

Expected stages:

- Format the Go source.
- Check for suspicious constructs.
- Run the complete test suite.

The command above will eventually be executable directly from this document.
For now, `--show` only renders it.

## Configuration example

Non-shell fences are presented as reference material and will not be
executable:

```yaml
document: README.md
mode: read
```

| Key | Intended action |
| --- | --- |
| `j` / `k` | Move down or up |
| `Tab` | Focus the next executable block |
| `Enter` | Execute the focused block |
| `q` | Quit |

Continue reading after each command and its inline result.
