# Context streams

`codec.ContextWithStdio` injects a `strings.Reader` and a `bytes.Buffer` into an embedded application's context. It explicitly passes the host's context stderr for diagnostics. `WithFallbackToStdio(false)` disables implicit process stdio access. The handler uses `codec.StdioOfContext`; the single-stream shortcuts are `codec.StdinOfContext`, `codec.StdoutOfContext` and `codec.StderrOfContext`.

The host explicitly selects process streams at its outermost boundary with `codec.ContextWithFallbackToStdio(ctx, true)`, then prints the captured result after execution. The inner context overrides the host's streams. The application borrows all injected streams and never closes them.

Run from the repository root:

```shell
mise example stdio
# Injected input.

mise example stdio --uppercase
# INJECTED INPUT.

mise example stdio --help
```

`NewApplication` enables fallback by default. Explicit streams always take precedence, including `nil`: `ContextWithStdio(ctx, nil, output, nil)` leaves stdin unavailable and uses `output` for both stdout and stderr. Pass `io.Discard` as stderr to suppress diagnostics explicitly. Without context streams and with fallback disabled, the getters return `nil`.

Use the same context for `Parse` and `Execute` when calling them separately. File aliases are resolved during `Parse`; `*os.File` options require actual `*os.File` context streams. Arbitrary readers/writers like those above are used directly through the context helpers, not through the file codec.

Built-in help and error handlers write to context stderr, or discard their output when no stream is available. Explicit `WithOutput` writers override the context. The same APIs are available in `cli`, including `cli.Print*` and `cli.ErrPrint*` convenience methods that do nothing when the corresponding output is unavailable. See [custom-help](../custom-help) for explicitly configured help output.
