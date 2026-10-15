# Custom codec

`colorCodec` implements `codec.Codec[color, uint32]` and is registered with `WithCodec`. `ParseRequirement` validates `#RRGGBB` input and produces a packed 24-bit requirement; `Resolve` turns it into the handler's RGB struct. `FormatRequirement` preserves the requirement for roundtrips, `DisplayRequirement` formats help defaults, and `ToRaw` supplies the registry adapter. Invalid input uses `codec.NewValidationErrorf` so the application reports a validation error with help.

Run from the repository root:

```shell
mise example custom-codec
# red=51 green=102 blue=153

mise example custom-codec --color '#FF8000'
# red=255 green=128 blue=0

mise example custom-codec --color invalid
# <input error and usage text>
# program exit code: 2

mise example custom-codec --help
```
