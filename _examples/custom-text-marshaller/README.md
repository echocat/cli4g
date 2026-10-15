# Custom text marshaller

A date type implements `encoding.TextMarshaler` and `encoding.TextUnmarshaler`. `WithCodec(codec.NewMarshaller[date]())` registers the adapter for the tagged field. Its requirement is the input string; `UnmarshalText` creates the date during resolution, while `MarshalText` formats it for the example's output. Invalid dates use `codec.NewValidationErrorf`, preserving the original parse error with `%w`.

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
