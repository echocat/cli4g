# Validation and technical errors

Both failures originate in `Validate`. An invalid integer uses `NewValidationErrorf`; a simulated read failure stays unmarked. The error handler inspects preserved causes with `errors.As`/`errors.Is`, then delegates to the standard responder.

Run from the repository root:

```shell
mise example validation-errors
# Accepted: 42

mise example validation-errors invalid
# Cause: strconv.NumError
# Error: value must be an integer: <original parse error>
# <usage text>
# program exit code: 2

mise example validation-errors --fail-read
# Cause: io.ErrUnexpectedEOF
# Error: validation-errors: validate: reading configuration: unexpected EOF
# program exit code: 1 (without usage text)

mise example validation-errors --help
```
