# Files

Struct tags (`argument`, `fileMode`, `allowSpecial`, `allowStdio`) supply `*os.File` to the handler. The invocation opens and closes file handles; help does not open them. With `allowStdio:"true"`, `-` or `stdin` borrows context stdin, which the invocation leaves open. Use `./stdin` or `./-` for literal filenames.

Run from the repository root; file paths are relative to this example directory:

```shell
mise example files sample.txt
# First line.
# Second line.

mise example files --help

mise example files -
# Reads standard input until EOF.

mise example files does-not-exist.txt
# <file-open error, without usage text>
# program exit code: 1
```
