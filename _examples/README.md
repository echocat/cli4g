# Examples

Run from the repository root with `mise example <name> [arguments...]`.

| Example | Shows |
| --- | --- |
| [simple](simple) | Typed options, a default flag, a required argument, and help |
| [argument-syntax](argument-syntax) | Optional/variadic arguments and the -- terminator |
| [commands](commands) | Two independent commands with no root handler |
| [commands-subcommands](commands-subcommands) | A command group with two leaf commands |
| [commands-inheritance](commands-inheritance) | Borrowing application and parent command options |
| [commands-introspection](commands-introspection) | Inspecting the invocation's command path and help metadata |
| [typed-context](typed-context) | Reading typed application, command, and flag values from context |
| [stdio](stdio) | Injecting in-memory context streams and disabling process-stdio fallback |
| [builder-fields](builder-fields) | Flag/argument constructors with names, descriptions, and WithField instead of tags |
| [embedded-options](embedded-options) | Reusing tagged flag groups through struct embedding |
| [options-factory](options-factory) | Creating options from configuration and injecting a service |
| [validation](validation) | Options validation and input-error responses |
| [validation-errors](validation-errors) | Explicit validation markers, technical errors, and preserved causes |
| [environment](environment) | CLI/environment/default priority |
| [enums](enums) | Typed text/Value enums, enum tags, and choices in help and errors |
| [completion](completion) | The `ship` deployment tool with automatic completion and inventory-aware namespace/service suggestions |
| [profiles](profiles) | Per-parse typed flag, argument, and command variants |
| [repeatable-flags](repeatable-flags) | Collecting repeated flag values in a slice |
| [date-time](date-time) | Date/DateTime codecs, custom layouts, and repeatable dates |
| [files](files) | File tags, runtime handles, and automatic resource cleanup |
| [lifecycle](lifecycle) | Options initialization and cleanup after success or failure |
| [cancellation](cancellation) | Ctrl+C and an endless context-aware handler with a configurable duration |
| [manual-invocation](manual-invocation) | Explicit Create/Parse/Execute/Close and independent invocations |
| [custom-values](custom-values) | Automatic Set/String adapters and cumulative map flags |
| [custom-text-marshaller](custom-text-marshaller) | Adapting TextMarshaler/TextUnmarshaler types |
| [custom-codec](custom-codec) | Implementing a codec with separate requirement and runtime types |
| [named-codecs](named-codecs) | Selecting different codecs for the same Go type through type tags |
| [custom-resource-codec](custom-resource-codec) | Acquiring resources in Resolve and registering scope cleanup |
| [custom-help](custom-help) | Custom help templates with automatic column alignment and word wrapping |
| [error-handling](error-handling) | Returning error responses and caller-selected exit codes |

Handlers can use `cli.Print`, `cli.Printf`, and `cli.Println` for stdout and
`cli.ErrPrint`, `cli.ErrPrintf`, and `cli.ErrPrintln` for stderr. Missing output is
a no-op. `cli.StdioOfContext(ctx)` returns all three streams for direct I/O;
single-stream helpers and the same conveniences are also available in `codec`.
`NewApplication` defaults to process stdin/stdout/stderr when no streams are set.
`cli.ContextWithStdio(ctx, stdin, stdout, stderr)` takes precedence; nil stderr
shares stdout. Disable implicit fallback with `WithFallbackToStdio(false)`.
Built-in help/error handlers also honor context stderr. See [stdio](stdio).
