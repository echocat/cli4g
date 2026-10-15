# Struct tags

Tags configure exported options fields. Use a struct or pointer to a struct.
See the [CLI Godoc](https://pkg.go.dev/github.com/echocat/cli4g#hdr-Options_and_bindings)
for the input and lifecycle rules.

| Tag | Applies to | Example | Meaning / default |
| --- | --- | --- | --- |
| `flag` | Flag | `flag:"port\|listen-port"` | Primary name and aliases, separated by `\|` |
| `argument` | Argument | `argument:"file"` | Positional input; declaration order determines its slot |
| `arg` | Argument | `arg:"file"` | Alias for `argument`; `argument` takes precedence |
| `short` | Flag | `short:"p\|P"` | Single-rune short aliases |
| `env` | Flag | `env:"PORT\|APP_PORT"` | Environment fallback names, tried in order |
| `default` | Flag / argument | `default:"8080"` | One codec input token; `default:""` is a declared default |
| `required` | Flag / argument | `required:"true"` | Demand explicit input; default `false` |
| `description` | Flag / argument | `description:"Listen port."` | Help description |
| `placeholder` | Flag / argument | `placeholder:"PORT"` | Help value label; normally derived from the input name |
| `visible` | Flag | `visible:"false"` | Hide from default help; default `true` |
| `type` | Flag / argument | `type:"date"` | Select a registered named codec |
| `fileMode` | File input | `fileMode:"readWrite"` | `read`, `write`, or `readWrite`; implicit mode is `read` |
| `allowSpecial` | File input | `allowSpecial:"true"` | Permit nonregular files; default `false` |
| `parent` | Exported struct pointer | `parent:"application"` | Borrow root options or, with `matchingType`, the nearest matching ancestor |

```go
type options struct {
    Port    int32    `flag:"port" short:"p" env:"PORT" default:"8080"`
    Input   string   `argument:"input" required:"true"`
    Labels  []string `flag:"label"`
}
```

## Boundaries

- A field cannot be both a flag and an argument.
- Required inputs cannot have declared defaults; options-factory values do not satisfy them.
- Slice flags append one value per occurrence. A slice argument must be last.
- Map flags accept one `key=value` pair per occurrence; map arguments must be last.
  Duplicate decoded keys are parse errors. String-keyed maps of standard scalar
  values are selected automatically; register other key types explicitly with
  [`codec.NewMap[K](valueCodec)`](https://pkg.go.dev/github.com/echocat/cli4g/codec#NewMap).
  CLI maps replace defaults. Values are split at the first `=` only, so
  `--labels env=prod`, `--labels=env=prod`, and `-Lenv=prod` are equivalent
  for `Labels map[string]string` tagged `flag:"labels" short:"L"`.
- `fileMode`/`allowSpecial` require `*os.File` or `[]*os.File` and cannot be combined with `type`.
- `parent` cannot be combined with input tags; borrowed options remain parent-owned.
- Anonymous struct embeddings expose their tags. Named subobjects need explicit builder binding.
- Explicit [`WithField`](https://pkg.go.dev/github.com/echocat/cli4g#FlagRequirement.WithField)
  bindings override that field's tags.

Related: [defaults](https://pkg.go.dev/github.com/echocat/cli4g#FlagRequirement.WithDefault),
[files](https://pkg.go.dev/github.com/echocat/cli4g/codec#FileRequirement),
[parent options](https://pkg.go.dev/github.com/echocat/cli4g#hdr-Commands_and_parent_options).
