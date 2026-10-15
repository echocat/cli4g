# Repeatable flags

Each occurrence of `--tag` or `-t` appends one value to a `[]string` field,
preserving order and duplicates.

Run from the repository root:

```shell
mise example repeatable-flags
# tags=[]

mise example repeatable-flags --tag blue --tag green
# tags=[blue green]

mise example repeatable-flags -t blue -t green -t blue
# tags=[blue green blue]

mise example repeatable-flags --help
```
