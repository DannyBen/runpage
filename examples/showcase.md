---
runpage:
  dependencies: [sh]
  workdir: self
---

# Runpage showcase

Runpage turns an ordinary Markdown document into an interactive command page.
Prose and executable blocks stay together in one continuous terminal view.

> Opening a page is always safe: commands run only when you select a block and
> press `Enter`.

## Rich Markdown

Use familiar Markdown to explain a workflow before asking anyone to run it:

- **Bold text** highlights an important result.
- Inline code such as `runpage --syntax` stays easy to spot.
- Lists, tables, quotes, and headings provide structure.

1. Read the instructions.
2. Select a command with `Tab` or `Shift+Tab`.
3. Execute it with `Enter`.

| State | Border |
| --- | --- |
| Ready | Cyan |
| Running | Blue |
| Success | Green |
| Failed | Red |

---

## Executable blocks

### A successful command

Blocks may have short labels. This one prints several lines to standard output:

```sh greeting
printf '%s\n' \
  'Hello from Runpage.' \
  'This command is harmless.' \
  'It only writes to standard output.'
```

### A multiline script

The entire fenced block is sent to its interpreter as one script:

```sh tiny report
for item in prose commands results; do
  printf '  ✓ %s\n' "$item"
done
printf 'Showcase complete.\n'
```

### An intentional failure

Failures remain in the document with their exit code and standard error. This
is deliberate and does not change anything on the system:

```sh expected failure
printf 'Checking an imaginary requirement...\n'
printf 'Requirement not found (this is the planned demo failure).\n' >&2
exit 2
```

### Back to success

Each block has its own result, so execution can continue after a failure:

```sh final check
printf 'Nothing was installed, deleted, committed, or pushed.\n'
printf 'Safe to run again.\n'
```

## Display-only blocks

Non-shell languages are rendered as reference material. The reserved `:noop` tag
can also make a supported language explicitly display-only.

```yaml configuration
runpage:
  dependencies: [sh]
  workdir: self
```

```python example :noop
message = "Shown as Python, never executed"
print(message)
```

```ruby example :noop
puts "Shown as Ruby, never executed"
```

## Done

Press `?` at any time to see the complete viewer key bindings. Press `q` to
leave the page.
