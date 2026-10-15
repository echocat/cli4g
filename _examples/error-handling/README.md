# Caller-managed errors

A responder with all exit codes zero prints errors and returns them. The caller uses `cli.IsValidationError(err)` to choose its own program exit codes, after cleanup.

Run from the repository root:

```shell
mise example error-handling "Hello, World!"
# Hello, World!
# program exit code: 0

mise example error-handling --unknown
# Error: unknown flag --unknown
# <usage text>
# program exit code: 64

mise example error-handling --fail
# Error: simulated execution failure
# program exit code: 70
```
