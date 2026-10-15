[![PkgGoDev](https://pkg.go.dev/badge/github.com/echocat/cli4g)](https://pkg.go.dev/github.com/echocat/cli4g)
[![Continuous Integration](https://github.com/echocat/cli4g/workflows/Continuous%20Integration/badge.svg)](https://github.com/echocat/cli4g/actions?query=workflow%3A%22Continuous+Integration%22)
[![Coverage Status](https://coveralls.io/repos/github/echocat/cli4g/badge.svg?branch=main)](https://coveralls.io/github/echocat/cli4g?branch=main)

# cli4g

Typed command-line applications for Go: flags, arguments, subcommands,
environment values, automatic help, and invocation-owned resources.

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
    cli.NewApplication[options]().
		WithName("greet").
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
- **Consistent inputs:** explicit CLI/environment/default precedence and validation rules.
- **Extensible values:** custom codecs support application-specific types and resources.
- **Efficient reuse:** competitive runtime and allocation costs in the measured workloads.

See [benchmarks and methodology](docs/benchmarks.md) for timings, allocations,
code size, and cognitive complexity.

## Documentation

- [Struct-tag/Annotations reference](docs/annotations.md)
- [CLI API and usage](https://pkg.go.dev/github.com/echocat/cli4g)
- [Codecs and custom values](https://pkg.go.dev/github.com/echocat/cli4g/codec)
- [CLI benchmark results](docs/benchmarks.md)
- [Runnable examples](_examples) — `mise example <name> [arguments...]`
- [Development](docs/development.md) · [Releasing](docs/releasing.md)

## License

[License](LICENSE) · [Contribute](https://github.com/echocat/cli4g/fork)
