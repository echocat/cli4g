# CLI benchmark results

Measured 2026-10-15 with `go1.27.2`, `linux/amd64`, `-cpu=1`, 200 ms per sample, seven samples.

CPU: AMD Ryzen 9 7950X3D 16-Core Processor. Values below are sample medians. [Source and methodology](../_benchmarks) · [Raw results](../_benchmarks/results.txt).

- **OneShot:** one definition + one parse/execution + cleanup.
- **Reuse/N:** one definition + N parse/execution calls + cleanup; construction is timed.
- `int32`, string, bool; identical rotating explicit/default inputs and checked handler results.
- Library defaults for help, completion, version, output, and error responses are left unchanged; all measured inputs succeed and handlers produce no output.
- Timings compare these adapters and workloads, not all features of each library.

All timing columns use **⌀µs/op**, where one op is one parse/execution call. Each sample's batch time is divided by its call count (1, 100, or 10,000) and converted to microseconds, including amortized construction and cleanup. The reported value is the median of these per-sample averages.

## Definition and binding modes

- **Direct:** manually registered flags, read through accessors or standalone value pointers.
- **StructTags:** options fields and input definitions discovered through reflection and struct tags; discovery and binding preparation are timed.

Cobra, urfave/cli, and Go `flag` do not provide native StructTags binding in the compared versions. `N/A` means the mode is not supported by the benchmarked native API.

## Code characters

**Code chars** counts non-whitespace Unicode characters inside each case's dedicated, gofmt-formatted `new…` body, including nested handler/run/reset/cleanup closures. All Unicode whitespace is excluded, including spaces, tabs, and newlines. Signatures, outer function braces, imports, separate struct definitions, and shared test/measurement code are excluded. Required supporting method bodies are included: Go `flag`'s `int32Value.String`/`Set` and Kong's `kongLeaf.Run`. Counts are stored with the timing samples and link to the readable adapter source. The same adapter supports all three lifecycles, so its code count is shown once.

## Cognitive complexity

**Cognitive complexity ↓** is calculated with [gocognit v1.3.0](https://github.com/uudashr/gocognit/tree/v1.3.0). Lower values indicate simpler control flow; zero is valid. The adapter score is the sum of the individual scores for the same constructor and required supporting methods used by Code chars. Nested closures are included once within their enclosing function, with gocognit's nesting rules. Analysis runs outside the timed region and scores are stored as `cognitive-complexity` in the raw samples. This measures control-flow complexity, not naming quality or overall readability.

## Root / Direct

Invocation: `bench [flags]`.

| Library | Code chars | Cognitive complexity ↓ | OneShot ⌀µs/op | Reuse/100 ⌀µs/op | Reuse/10000 ⌀µs/op | OneShot B / allocs | Reuse/10000 B / allocs per invocation |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| cli4g | [678](../_benchmarks/cli4g_test.go) | 1 | 8.02 | 1.20 | 1.13 | 13744 / 176 | 801.3 / 16.68 |
| urfave | [368](../_benchmarks/urfave_test.go) | 0 | 5.98 | 4.25 | 4.25 | 8184 / 214 | 4400.4 / 162.67 |
| cobra | [748](../_benchmarks/cobra_test.go) | 14 | 2.94 | 1.55 | 1.59 | 5642 / 49 | 2159.2 / 16.34 |
| flag | [665](../_benchmarks/flag_test.go) | 11 | 0.37 | 0.17 | 0.17 | 842 / 12 | 24.1 / 1.00 |
| kong | N/A | N/A | N/A | N/A | N/A | N/A | N/A |

## Root / StructTags

Invocation: `bench [flags]`.

| Library | Code chars | Cognitive complexity ↓ | OneShot ⌀µs/op | Reuse/100 ⌀µs/op | Reuse/10000 ⌀µs/op | OneShot B / allocs | Reuse/10000 B / allocs per invocation |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| cli4g | [328](../_benchmarks/cli4g_test.go) | 1 | 10.21 | 1.29 | 1.17 | 14212 / 200 | 876.1 / 19.35 |
| urfave | N/A | N/A | N/A | N/A | N/A | N/A | N/A |
| cobra | N/A | N/A | N/A | N/A | N/A | N/A | N/A |
| flag | N/A | N/A | N/A | N/A | N/A | N/A | N/A |
| kong | [320](../_benchmarks/kong_test.go) | 3 | 20.46 | 9.31 | 8.94 | 20631 / 359 | 6771.7 / 176.37 |

## Subcommand / Direct

Invocation: `bench a b [flags]`.

| Library | Code chars | Cognitive complexity ↓ | OneShot ⌀µs/op | Reuse/100 ⌀µs/op | Reuse/10000 ⌀µs/op | OneShot B / allocs | Reuse/10000 B / allocs per invocation |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| cli4g | [788](../_benchmarks/cli4g_test.go) | 1 | 13.28 | 1.93 | 1.76 | 20730 / 266 | 1506.1 / 25.69 |
| urfave | [473](../_benchmarks/urfave_test.go) | 0 | 14.95 | 10.84 | 11.04 | 21792 / 561 | 12905.0 / 429.35 |
| cobra | [851](../_benchmarks/cobra_test.go) | 14 | 8.49 | 2.28 | 2.17 | 18590 / 136 | 1589.9 / 26.68 |
| flag | [734](../_benchmarks/flag_test.go) | 14 | 0.39 | 0.17 | 0.17 | 842 / 12 | 24.1 / 1.00 |
| kong | N/A | N/A | N/A | N/A | N/A | N/A | N/A |

## Subcommand / StructTags

Invocation: `bench a b [flags]`.

| Library | Code chars | Cognitive complexity ↓ | OneShot ⌀µs/op | Reuse/100 ⌀µs/op | Reuse/10000 ⌀µs/op | OneShot B / allocs | Reuse/10000 B / allocs per invocation |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| cli4g | [438](../_benchmarks/cli4g_test.go) | 1 | 15.05 | 1.99 | 1.83 | 21198 / 290 | 1580.9 / 28.36 |
| urfave | N/A | N/A | N/A | N/A | N/A | N/A | N/A |
| cobra | N/A | N/A | N/A | N/A | N/A | N/A | N/A |
| flag | N/A | N/A | N/A | N/A | N/A | N/A | N/A |
| kong | [324](../_benchmarks/kong_test.go) | 3 | 33.09 | 17.11 | 16.63 | 33917 / 558 | 15005.8 / 315.73 |

## Scope

Versions, adapter differences, reset policy, and reproduction commands are in the [benchmark module README](../_benchmarks). No I/O, help rendering, malformed inputs, parallel execution, or resource acquisition is measured. These are local measurements, not a portable performance guarantee.
