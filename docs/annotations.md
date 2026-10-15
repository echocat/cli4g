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
| `enum` | String flag / argument | `enum:"auto\|always\|never"` | Restrict literal string choices and advertise them in help |
| `fileMode` | File input | `fileMode:"readWrite"` | `read`, `write`, or `readWrite`; implicit mode is `read` |
| `allowSpecial` | File input | `allowSpecial:"true"` | Permit nonregular files; default `false` |
| `allowStdio` | File input | `allowStdio:"true"` | Interpret `-`/`stdin` as stdin in `read` mode and `-`/`stdout` as stdout in `write` mode; default `false` |
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
- `fileMode`/`allowSpecial`/`allowStdio` require `*os.File` or `[]*os.File` and cannot be combined with `type`.
- `enum` requires a string/string-based field and cannot be combined with `type`.
  It selects a field-local literal codec. Use an explicit enum codec to retain
  custom TextMarshaler/Value parsing or to support collection elements.
- `parent` cannot be combined with input tags; borrowed options remain parent-owned.
- Anonymous struct embeddings expose their tags. Named subobjects need explicit builder binding.
- Explicit [`WithField`](https://pkg.go.dev/github.com/echocat/cli4g#FlagRequirement.WithField)
  bindings override that field's tags.

## Automatic custom codecs

Without an explicit codec or `type` tag, concrete `T` and `*T` fields are also
recognized when `*T` implements either:

- `codec.Value` (`Set(string) error` and `String() string`), or
- both `encoding.TextMarshaler` and `encoding.TextUnmarshaler`.

This applies to flags and arguments. Explicit/registered codecs take precedence;
`codec.Value` wins when a type supports both adapters. A nonempty `type` tag
requires a matching named registration and never falls back automatically.

Targets are allocated fresh for each resolution; their zero value must be ready
for `Set` or `UnmarshalText`. Text defaults are input strings. Help/default
inspection does not call `MarshalText` or `UnmarshalText`. For types needing
initialization, register an explicit factory adapter such as
`codec.NewMarshallerP`. Collection codecs for slices/maps of these custom types
must still be registered explicitly.

## Enums

```go
type options struct {
    Color string `flag:"color" enum:"auto|always|never" default:"auto"`
    Side  string `argument:"side" enum:"left|right"`
}
```

Choices are exact and case-sensitive, with declaration order preserved in help
and validation errors. An empty entry represents the empty string; duplicate
entries are configuration errors. `default` and environment values must satisfy
the same restriction as CLI inputs. Explicit `WithField` bindings override tags.

For typed alternatives, use `codec.NewMarshallerEnum` or `codec.NewValueEnum`.
The optional `codec.Enum[R]` contract also lets third-party codecs expose their
full requirement set. See the [enums example](../_examples/enums).

## Standard streams

```go
type options struct {
    Input  *os.File `argument:"input" fileMode:"read" allowStdio:"true"`
    Output *os.File `flag:"output" fileMode:"write" allowStdio:"true"`
}
```

With `allowStdio:"true"`, exact aliases return context stdin or stdout directly.
The invocation borrows these streams and does not close them. This opt-in is
independent of `allowSpecial`; without it, all names remain literal file paths.
Use `./stdin`, `./stdout`, or `./-` to address literal files with opt-in enabled.

Supply streams with `codec.ContextWithStdio(ctx, input, output, diagnostics)` and pass that
context to both `Parse` and `Execute`, or to `ParseAndExecute`. `NewApplication`
defaults to `WithFallbackToStdio(true)`: only when no streams are set in the context
does it supply `os.Stdin`, `os.Stdout` and `os.Stderr`. Disable this with
`WithFallbackToStdio(false)`. Explicit streams suppress process fallback entirely,
including `nil` stdin/stdout. Nil stderr uses stdout; pass `io.Discard` to suppress
stderr explicitly. The context helpers return `nil` for missing streams.

File aliases require non-nil `*os.File` context streams. A missing stream or an
arbitrary reader/writer returns `codec.ErrStdioUnavailable` rather than falling
back to process stdio. Arbitrary readers/writers are available to handlers through
`codec.StdioOfContext`, `codec.StdinOfContext`, and `codec.StdoutOfContext`.
Standalone codec calls must supply context streams explicitly; they never fall
back implicitly. Built-in help and error handlers use `codec.StderrOfContext` at
each invocation, or `io.Discard` when no output is available. Explicit `WithOutput`
writers take precedence over the context. These helpers and the `Print*`/`ErrPrint*`
convenience methods are also available directly in package `cli`.

Aliases with the wrong direction (`stdout` in `read` mode or `stdin` in `write`
mode), and all aliases in `readWrite` mode, are input errors. Other paths retain
the normal file behavior. Explicit append/truncate policies are rejected for
aliases; file creation and permission settings do not affect borrowed streams.

For builders, use `codec.NewFile(mode).WithAllowStdio(true)` via `WithCodec`.
Typed `codec.FileRequirement` defaults select their own policy with
`AllowStdio: true`, independently of the codec's path-input defaults.

Related: [defaults](https://pkg.go.dev/github.com/echocat/cli4g#FlagRequirement.WithDefault),
[files](https://pkg.go.dev/github.com/echocat/cli4g/codec#FileRequirement),
[parent options](https://pkg.go.dev/github.com/echocat/cli4g#hdr-Commands_and_parent_options).
