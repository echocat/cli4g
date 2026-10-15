# Options factory

`WithCreateOptionsHandler` returns fresh options and a greeting service for each invocation, using an existing configuration. The service's output is taken from `codec.StdoutOfContext(ctx)` so concurrent invocations can use different streams. `Greeting` has no declared default, so the factory value remains until a CLI flag overrides it. The untagged service is supplied by the factory. Help does not invoke the factory or display factory values as declared defaults.

Run from the repository root:

```shell
mise example options-factory
# Welcome, World!

mise example options-factory Carl
# Welcome, Carl!

mise example options-factory -g Hey Carl
# Hey, Carl!

mise example options-factory --help
```
