# CLI comparisons

Independent Go module comparing cli4g with urfave/cli v3.14.0, Cobra v1.10.2,
Kong v1.16.1, and Go's standard `flag` package. Versions are pinned in `go.mod`.

[Measured results](../docs/benchmarks.md) · [Raw samples](results.txt)

## Workload

- **Root:** `bench [flags]`, with no commands; **Subcommand:** `bench a b [flags]`,
  with two nested commands.
- Three flags on the executed root or leaf: `--count` (`int32`), `--name` (string),
  `--verbose` (bool).
- Every handler checks all values; calls rotate positive/negative integers,
  true/false, strings, and omitted flags restoring defaults.
- Library defaults for help, completion, version, output, and error responses are
  left unchanged. All measured inputs succeed and handlers produce no output.

## Definition and binding modes

| Mode | Definition and value access | Libraries |
| --- | --- | --- |
| `Direct` | Manually register each flag; read accessors or standalone value pointers | cli4g, urfave/cli, Cobra, Go `flag` |
| `PointerBindings` | Manually register each flag with a pointer to an options field | urfave/cli, Cobra, Go `flag` |
| `StructTags` | Discover options fields and input definitions through reflection and struct tags | cli4g, Kong |

Each library/shape/mode combination has its own readable `new…` constructor,
without switches between unrelated cases.

Result tables show `Direct` and `StructTags`. `PointerBindings` cases are included
only in the raw measurements.

`PointerBindings` and `StructTags` measure different capabilities. Keeping values
in a struct does not imply automatic options parsing: pointer bindings require
the caller to register each field, name, type, and default explicitly. Struct-tag
binding discovers that information, and its reflection/schema preparation is
included in construction time.

Cobra, urfave/cli, and Go `flag` do not provide native struct-tag binding in the
compared versions. No third-party binding layer is added. Results are grouped by
mode; unsupported combinations are marked `N/A`. Cross-mode timings do not
compare equivalent definition and binding features.

## Lifecycles

| Case | One measured operation |
| --- | --- |
| `OneShot` | Create definition → parse/execute once → close |
| `Reuse/N=100` | Create definition once → parse/execute 100 times → close |
| `Reuse/N=10000` | Create definition once → parse/execute 10,000 times → close |

Construction is **inside** every measured batch. Reused batches do not hide
initialization cost. The same input distribution is used in all cases.
In the raw Go output, `ns/op`, `B/op`, and `allocs/op` refer to a complete batch.
The additional `ns/invocation` metric divides batch time by N.

All result-table timing columns use **⌀µs/op**, where one op is one parse/execution
call: batch nanoseconds / call count / 1,000. Call counts are 1, 100, and 10,000, so
construction and cleanup are amortized across the calls. This makes the cost per
call directly comparable across all three lifecycles. Results show the median
of seven per-sample averages.

## Adapter differences

| Library | Direct | PointerBindings | StructTags | Reuse behavior |
| --- | --- | --- | --- | --- |
| cli4g | `WithInt32Flag`, typed helpers, invocation getters | N/A | Reflection and struct tags | Reuse compiled application; new invocation per call; cleanup included |
| urfave/cli | Flag objects and command getters | `Destination` field pointers | N/A | Native `Command.Run`; its internal per-run preparation remains timed |
| Cobra | pflag definitions and getters | pflag `*Var` field pointers | N/A | Reuse tree; reset flag values/Changed before subsequent calls |
| Go `flag` | Explicit standalone value pointers | `*Var`/`Value` field pointers | N/A | Reuse FlagSet; reset values before subsequent calls |
| Kong | N/A | N/A | Reflection, struct tags, and `Run` | Reuse parser; native Parse and Context.Run remain timed |

Go `flag` has no command tree or int32 helper. Its subcommand adapter includes a
minimal `a b` dispatcher; both shapes use a decimal int32 `flag.Value`. This is a small
baseline, not feature equivalence with the frameworks. No custom direct API is
invented for Kong.

Tests reuse every definition with all inputs three times, checking default
restoration, value changes, and exactly one handler call. No parser is rebuilt
outside the measured batch; required internal work and resets are included.

## Code characters

The `Code chars` column counts non-whitespace Unicode characters inside the
dedicated `new…` body for that case, including nested handler/run/reset/cleanup
closures. Source is normalized with gofmt; all Unicode whitespace, including
spaces, tabs, and newlines, is excluded. The source remains readable. Signatures
and the body's outer braces are excluded, as are imports, separate struct
definitions, and shared test or measurement code.

Required supporting method bodies are also counted: `int32Value.String` and
`int32Value.Set` for Go `flag`, and `kongLeaf.Run` for Kong. Moving required
adapter code outside the constructor therefore does not hide its size.

The count is calculated outside the timed region and stored as `code-chars` in
each raw sample. It is shown once per adapter because the same constructor and
run closure serve OneShot and both Reuse lifecycles. The number links to the
readable source. It describes this benchmark implementation, including the
necessary reuse/reset code, rather than a minimum possible application size.

## Cognitive complexity

The `Cognitive complexity ↓` column uses
[gocognit v1.3.0](https://github.com/uudashr/gocognit/tree/v1.3.0) directly as a Go
library, pinned in this module. Lower values indicate simpler control flow;
zero is a valid score.

For each adapter, the score is the **sum of gocognit's per-function scores** for
the same constructor and required supporting methods counted by `Code chars`.
Nested handler/run/reset/cleanup closures are included once by gocognit while
analyzing their enclosing function. Its standard rules account for their
nesting depth. Separate struct definitions and the shared test/measurement
harness are excluded.

Analysis is outside the timed region. The score is stored as
`cognitive-complexity` in each raw sample and shown once per adapter, since all
three lifecycles use the same code. This is an adapter-level control-flow
metric, not a score for naming, API familiarity, or overall readability.

## Run

```shell
mise run test:benchmarks
mise run bench
```

The reporter runs the benchmark, writes `results.txt`, and updates
`docs/benchmarks.md`.

> [!WARNING]
> Run benchmarks on an otherwise idle machine. Do not compare timings obtained with the race detector, coverage, different Go versions, or concurrent heavy work.
