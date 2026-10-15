# Files

Struct tags (`argument`, `fileMode`, `allowSpecial`) supply `*os.File` to the handler. The invocation opens and closes the handle; help does not open it.

Run from the repository root; file paths are relative to this example directory:

```shell
mise example files sample.txt
# First line.
# Second line.

mise example files --help

mise example files does-not-exist.txt
# <file-open error, without usage text>
# program exit code: 1
```
