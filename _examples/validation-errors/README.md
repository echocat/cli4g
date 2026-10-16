# Validation and technical errors

Both failures originate in `Validate`. An invalid integer uses
`NewValidationError(message, cause)`; a simulated read failure stays unmarked.
The error handler inspects preserved causes with `errors.As`/`errors.Is`, then
delegates to the standard responder. Cause text and input values are not appended
to the displayed validation message. Unmarked failures retain their operational
classification and receive a neutral message.

Run from the repository root:

```shell
mise example validation-errors
# Accepted: 42

mise example validation-errors invalid
# Expected: an integer
# Error: value must be an integer
# <usage text>
# program exit code: 2

mise example validation-errors --fail-read
# Configuration could not be read completely
# Error: validation-errors: validate: options could not be validated
# program exit code: 1 (without usage text)

mise example validation-errors --help
```
