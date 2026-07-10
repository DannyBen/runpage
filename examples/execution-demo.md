# Execution demo

## Successful command

```bash
printf 'hello from stdout\n'
printf 'all checks passed\n'
```

## Failed command

```sh
printf 'starting verification\n'
printf 'verification failed\n' >&2
exit 3
```

## Current shell

```console
printf 'shell: %s\n' "$SHELL"
pwd
```

## Display only

```shell :noop
echo 'this command cannot be executed'
```
