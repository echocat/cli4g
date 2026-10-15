# Validation

The options' `Validate` method checks the decoded values before execution.
Invalid values are explicitly marked with `cli.NewValidationErrorf` and produce
help and program exit code 2. Unmarked errors remain operational failures.

Run from the repository root:

```shell
mise example validation -n 2 World
# Hello, World!
# Hello, World!

mise example validation --count 0 World
# Error: count must be between 1 and 5
# <usage text>
# program exit code: 2

mise example validation ""
# Error: name must not be blank
# <usage text>
# program exit code: 2
```
