[![PkgGoDev](https://pkg.go.dev/badge/github.com/echocat/cli4g)](https://pkg.go.dev/github.com/echocat/cli4g)
[![Continuous Integration](https://github.com/echocat/cli4g/workflows/Continuous%20Integration/badge.svg)](https://github.com/echocat/cli4g/actions?query=workflow%3A%22Continuous+Integration%22)
[![Coverage Status](https://coveralls.io/repos/github/echocat/cli4g/badge.svg?branch=main)](https://coveralls.io/github/echocat/cli4g?branch=main)

# cli4g

Typed command-line applications for Go: flags, arguments, subcommands,
environment values, automatic help, shell completion, and invocation-owned resources.

## Install

Go **1.27.0+** is required.

```shell
go get github.com/echocat/cli4g
```

## Quick start

```go
package main

import (
    "context"
    "fmt"
    "os"

    cli "github.com/echocat/cli4g"
)

type options struct {
    Name     string `argument:"name" required:"true"`
    Greeting string `flag:"greeting" short:"g" default:"Hello"`
}

func main() {
    cli.NewApplication[options]("greet", "Greet someone.").
		WithHandler(greet).
		MustCreateParseAndExecute(context.Background(), nil, os.Args[1:]...)
}

func greet(_ context.Context, opts options) error {
	_, err := fmt.Printf("%s, %s!\n", opts.Greeting, opts.Name)
	return err
}
```

```shell
greet Ada                 # Hello, Ada!
greet -g Hey Ada          # Hey, Ada!
greet --help
```

The default error handler exits with `2` for input errors, `130` for cancellation, or `1` for other failures. For embedding, disable it with [`WithErrorHandlerEnabled(false)`](https://pkg.go.dev/github.com/echocat/cli4g#ApplicationRequirement.WithErrorHandlerEnabled).

## Why cli4g?

- **Typed options:** struct tags define inputs; handlers receive typed values.
- **Ordered lifecycle:** parsing resolves and validates all selected options before initialization; handlers follow successful initialization.
- **Owned resources and cleanup:** `ParseAndExecute` closes invocation-owned resources from child to parent, on success and returned errors—including partial initialization failures.
- **Reusable definitions:** compile once, then parse independent invocations repeatedly or concurrently.
- **Profiles:** select compatible flag, argument, and command variants for each parse.
- **Consistent inputs:** explicit CLI/environment/default precedence and validation rules.
- **Extensible values:** custom codecs support application-specific types and resources.
- **Efficient reuse:** competitive runtime and allocation costs in the measured workloads.

## Comparison at a glance

Qualitative ratings for the [compared versions](_benchmarks):
**★★★** strong · **★★☆** moderate/partial · **★☆☆** limited · **—** no native support.

| Capability | **cli4g** | [urfave/cli](https://github.com/urfave/cli/tree/v3.14.0) | [Cobra](https://github.com/spf13/cobra/tree/v1.10.2) | [Kong](https://github.com/alecthomas/kong/tree/v1.16.1) | [Go `flag`](https://pkg.go.dev/flag@go1.27.2) |
| --- | :---: | :---: | :---: | :---: | :---: |
| Struct-tag option binding | ★★★ | — | — | ★★★ | — |
| Definition simplicity (measured adapters)[^complexity] | ★★★ | ★★★ | ★☆☆ | ★★★ | ★☆☆ |
| Command trees and automatic help | ★★★ | ★★★ | ★★★ | ★★★ | — |
| Custom value types | ★★★ | ★★★ | ★★★ | ★★★ | ★★★ |
| Lifecycle and resource cleanup | ★★★ | ★★☆ | ★★☆ | ★★☆ | — |
| Compile once, isolated concurrent parses | ★★★ | — | — | — | — |
| Per-parse schema profiles | ★★★ | — | — | — | — |
| Shell completion | ★★★ | ★★★ | ★★★ | — | — |
| Setup/compile + first invocation speed[^performance] | ★★☆ | ★★☆ | ★★★ | ★☆☆ | ★★★ |
| Parse/execute speed on reuse[^performance] | ★★★ | ★★☆ | ★★★ | ★☆☆ | ★★★ |

Isolated parses keep each invocation's values independent. Resource cleanup
includes rollback after partial initialization failures; hooks alone need application-managed ownership.

[^complexity]: Based on the benchmark adapters' [cognitive complexity](https://github.com/echocat/cli4g/blob/ce5d951/docs/benchmarks.md#cognitive-complexity); lower is better:
    cli4g **1**, urfave/cli **0**, Cobra **14**, Kong **3**, Go `flag` **11–14**.
    Includes handlers, reuse/reset, cleanup, and required helper methods; measures control flow rather than overall API ergonomics.

[^performance]: Based on the [measured root/subcommand workloads](https://github.com/echocat/cli4g/blob/ce5d951/docs/benchmarks.md).
    Setup is `OneShot`: definition creation/compilation plus one parse/execute and cleanup.
    Reuse is `Reuse/100` and `Reuse/10000`, including amortized setup and cleanup.
    Native binding modes differ; Go `flag` is a minimal baseline with application-written command dispatch.

See [benchmarks and methodology](docs/benchmarks.md) for timings, allocations,
code size, and cognitive complexity.

## Documentation

- [Struct-tag/Annotations reference](docs/annotations.md)
- [Profiles and schema variants](docs/profiles.md)
- [CLI API and usage](https://pkg.go.dev/github.com/echocat/cli4g)
- [Codecs and custom values](https://pkg.go.dev/github.com/echocat/cli4g/codec)
- [CLI benchmark results](docs/benchmarks.md)
- [Runnable examples](_examples) — `mise example <name> [arguments...]`
- [Development](docs/development.md) · [Releasing](docs/releasing.md)

## License

[License](LICENSE) · [Contribute](https://github.com/echocat/cli4g/fork)
