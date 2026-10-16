# Sub-commands

The `math` command groups two sub-commands. Only the leaf commands have handlers.
The application explicitly sets `WithMaxCommandDepth(2)`: `math` has depth 1 and
its leaves have depth 2. The constructor default is 7; zero permits no commands.
The limit is checked during `Create`, independently of the number of siblings.

Run from the repository root:

```shell
mise example commands-subcommands math sum 2 3 4
# Sum: 9

mise example commands-subcommands math product 2 3 4
# Product: 24

mise example commands-subcommands math --help

mise example commands-subcommands math
# Error: a subcommand is required for "commands-subcommands math"
# <help for math>
# program exit code: 2
```
