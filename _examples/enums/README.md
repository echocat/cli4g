# Enums

Three ways to declare finite choices:

- `enum:"auto|always|never"` on a string/string-based field.
- `codec.NewMarshallerEnum` or `NewMarshallerEnumVariants` with typed values
  implementing text marshaling/unmarshaling. This example maps `iota` constants
  to `json` and `yaml` and supplies a description for each value.
- `codec.NewValueEnum` with typed Set/String alternatives.

Run from the repository root:

```shell
mise example enums
# color=auto format=json level=info extra=0

mise example enums --color never --format yaml --extra-format json debug
# color=never format=yaml level=debug extra=1

mise example enums --format toml
# Error: ... expected one of: "json", "yaml"
# <help including the valid choices and their descriptions>
# program exit code: 2

mise example enums --help
```

The helpers derive canonical input strings from the declared values once during
construction. Parsing rejects undeclared strings before `Set`/`UnmarshalText`;
resolution still creates fresh values. Defaults for these adapters are input
strings (`WithDefaultStr`, or `default` tags), not live runtime objects. A Value
enum opts into the additional requirement that `String()` produces a canonical
input accepted by `Set`.

`NewEnum(codec, requirements...)` restricts an arbitrary codec's resource-free
requirements, retaining that codec's normalization rules. `NewEnumVariants`
adds descriptions. Use `NewSlice(enumCodec)` or `NewMap[string](enumCodec)` for
repeated elements. Help advertises element choices, not all possible collections.

Custom codecs may implement `codec.Enum[R]` and return the complete valid set as
`[]codec.EnumVariant[R]` from `EnumVariants(ctx)`. This publishes metadata for help
and tooling; the custom codec remains responsible for validation. Implement
`ToRaw` with `codec.Raw(ctx, yourCodec)` to retain the optional contract. Metadata
must be resource-free and must not call `Resolve`.
