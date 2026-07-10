# Release checklist

## 1. Inspect the working tree

```bash
git status --short
git diff --check
```

## 2. Run project checks

```bash
op check
go mod verify
```

## 3. Inspect the release configuration

```bash
goreleaser check
git tag --sort=-version:refname | head -5
```

## 4. Build a snapshot

```bash
goreleaser release --snapshot --clean
ls -lh dist/
```

## 5. Review the artifacts

```bash
find dist -maxdepth 2 -type f | sort
git status --short
```

## 6. Publish

This final command is intentionally not executable in Mob.

```bash :noop
git push origin main
git push origin --tags
```
