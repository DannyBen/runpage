# Agent Notes

## Project Shape

- `main.go` is the small binary entrypoint and injects `Version`.
- `cmd/` owns CLI parsing, command behavior, help text, and tests.
- `op.conf` is the user-facing command catalog.
- `.goreleaser.yml` and `.github/workflows/` own release automation.

## Useful Commands

```bash
go test ./...
op check
go run . --help
```

## Editing Rules

- Keep the Markdown reader document-oriented: prose and executable blocks must
  remain in one continuous view.
- Keep stdout scriptable in non-interactive modes; errors go to stderr.
- Keep dependencies minimal and add focused tests for CLI behavior.
- Do not run parallel commands that scan, test, build, format, generate, render,
  or use git under `/vagrant`; shared-folder access can hang the VM.
