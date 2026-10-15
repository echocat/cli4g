# Command introspection

A root handler uses `InvocationFromContext` and `NameStack()` to print the
selected command path with `--verbose`. The path uses canonical names, even when
called through aliases. `HelpInfo()` and `HelpHandler()` display help when the
selected target still has subcommands.

Run from the repository root:

```shell
mise example commands-introspection -v hello Carl
# Executing: greet
# Hello, Carl!

mise example commands-introspection -v math add 2 3 4
# Executing: math sum
# Sum: 9

mise example commands-introspection math
# <help for the math command group>

mise example commands-introspection math sum --help
```
