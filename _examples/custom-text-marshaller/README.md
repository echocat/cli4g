# Custom text marshaller

A date type implements `encoding.TextMarshaler` and `encoding.TextUnmarshaler`. The tagged field selects the text adapter automatically, without `WithCodec` or a `type` tag. Its requirement is the input string; `UnmarshalText` creates the date during resolution, while `MarshalText` formats it for the example's output. Invalid dates use `codec.NewValidationError(message, cause)` to describe the required format without displaying the input. The original parse error remains accessible through `errors.As`.

Automatic detection supports value and pointer fields whose pointer implements
both text interfaces and whose zero value is ready for unmarshaling. Explicit
or registered codecs take precedence. Types also implementing `codec.Value`
retain their existing Set/String behavior. Use `codec.NewMarshallerP` with an
explicit factory when construction requires more than a zero value.

Run from the repository root:

```shell
mise example custom-text-marshaller
# date=2026-01-01

mise example custom-text-marshaller --date 2026-10-13
# date=2026-10-13

mise example custom-text-marshaller --date 2026-02-30
# <input error and usage text>
# program exit code: 2

mise example custom-text-marshaller --help
```
