# Ship: plan a service deployment

`ship` plans service deployments to development or production namespaces. A local inventory makes the example runnable without a cluster; the command prints its deployment plan.

The options and `deploy` command are defined once in `main.go`. Completion uses that normal CLI definition: `WithCompletion(true)` selects the default public command and request entry, and `codec.WithCompletionHandler` adds inventory lookups to ordinary string codecs.

## Run the command

Run from the repository root:

```shell
mise example completion deploy api
# Plan: api -> development/default (replicas=1, wait=false)

mise example completion deploy checkout --context production --namespace payments --replicas 3 --wait
# Plan: checkout -> production/payments (replicas=3, wait=true)
```

Omitting `--context` uses `CLI4G_SHIP_CONTEXT`, then the `development` default.

## Try interactive completion

Build the example as `ship` (`ship.exe` on Windows), then make that executable available on `PATH`. The directory is named after the demonstrated feature; the application itself is a deployment tool:

```shell
go -C _examples/completion build -o ship .
```

In **Bash 4.4+**, with a configured `bash-completion` loader:

```bash
ship completion install bash
# At the prompt, type and press Tab:
# ship --context production deploy checkout --namespace pa<Tab>
```

In **Nushell 0.114+**:

```nu
ship completion install nu
# Start a new interactive Nu session, then type and press Tab:
# ship --context production --namespace payments deploy ch<Tab>
```

In **Windows PowerShell 5.1+ or PowerShell 7+**:

```powershell
ship completion install powershell
# Start a new PowerShell session, then type and press Tab:
# ship --context production --namespace payments deploy ch<Tab>
```

The standard handler completes `deploy`, flag names, the `development`/`production` context enum, and bool values automatically. The two callbacks in `inventory.go` add domain knowledge:

- Namespace suggestions depend on the context and the selected service.
- Service suggestions depend on the context and namespace.

cli4g supplies the complete input context, filters prefixes, and handles shell output. The callbacks return plain values; the same inventory is used by the normal deployment handler. A real client lookup would use the supplied cancellation context. Installed adapters continue to work after binary updates.

The public setup command is the ordinary `DefaultCompletionCommand`, initially named `completion`. It uses the same lifecycle and parent requirements as `deploy`. Applications can replace it with `WithCompletionCommand(...)`, including a fresh `NewCompletionCommand().WithName("shells")`. Only the separate request entry (initially `__complete`) bypasses normal validation and initialization. Configure it with `WithCompletionRequestCommandName`; empty disables it.

See the [GoDoc guide](https://pkg.go.dev/github.com/echocat/cli4g#hdr-Shell_completion) for API and setup details.
