# Embedded options

Both commands embed the same `OutputOptions` struct. Its tagged `uppercase` flag
becomes an own flag in each command, with independent options instances. Embedding
reuses a definition rather than borrowing parent options.

Run from the repository root:

```shell
mise example embedded-options greet Carl
# Hello, Carl!

mise example embedded-options greet -u Carl
# HELLO, CARL!

mise example embedded-options echo --uppercase "Some text"
# SOME TEXT

mise example embedded-options echo --help
```
