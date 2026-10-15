# Custom help

`NewTemplatedHelpHandler` uses custom application, usage, flag, and argument
templates. They consume preformatted labels and defaults and write help to stdout.
The same layout is used for explicit help and input-error responses.

Run from the repository root:

```shell
mise example custom-help --help
# Help for custom-help
# A greeting with a custom help layout.
#
# Run: custom-help [flags] <NAME>
#
# Inputs:
#   NAME: Name to greet. [required]
#
# Options:
#   -g, --greeting GREETING: Greeting to use. [default: Hello]
#   -h, --help: Show help.

mise example custom-help Carl
# Hello, Carl!

mise example custom-help --unknown
# Error: unknown flag --unknown
# <custom help layout>
# program exit code: 2
```
