# Development

Use [mise](https://mise.jdx.dev/) **2026.9.18+** to select the locked tools.

```shell
mise trust
mise install
mise run check:full
```

## Common tasks

| Command | Purpose |
| --- | --- |
| `mise run format` | Format Go code and tidy modules |
| `mise run lint` | Check Go, modules, and workflows |
| `mise run test` | Test Go code and examples |
| `mise run test:go --race` | Go tests with race detection |
| `mise run test --coverage` | Go and example coverage |
| `mise run test:matrix` | Minimum and stable Go versions |
| `mise run test:module` | Test the distributable module archive without nested modules or Git metadata |
| `mise run test:fuzz` | Bounded parser fuzzing |
| `mise run test:benchmarks` | Check comparison adapters and their module |
| `mise run bench` | Measure CLI lifecycle comparisons and update the results page |
| `mise run check:full` | Full quality gate |
| `mise run tools:update` | Refresh locked tool versions |

List all tasks with `mise tasks`. Race detection needs a C toolchain:
GCC on Linux, Xcode Command Line Tools on macOS, or MinGW-w64 on Windows.

## Examples and coverage

Examples are independent modules under [`_examples`](../_examples).

```shell
mise example simple Ada
mise run test:examples:code
mise run test:examples:examples
mise run test:go --coverage -- -run TestTaggedCalendarDefaultsEnvironmentAndCLI
```

Plain `go test ./...` does not build example modules. Example tasks use
`-count=1` to include changes in those modules.

Module-archive checks build a Go module zip from repository sources and run its
tests after extraction. Example-list tests use their own fixtures; mise task
integration requires a Git checkout. Explicit example checks still require the
complete repository.

- Go coverage: `profile.cov`.
- Example-binary coverage: `profile.examples.cov`, including library calls.
- `CLI4G_CI=1` enables race/verbose/coverage for Go tests and binary coverage
  for examples; Go tests then write `profile.cov.full`.

## CI

- Go linting: Linux, macOS, Windows.
- Runtime tests: minimum/stable Go on each OS, including example binaries.
- Shared checks: once on Linux; formats, module archives, workflows, example code,
  benchmark-adapter correctness, fuzzing, and vulnerabilities. Timing measurements
  are run separately on an idle machine, not in CI.
- Coverage: one Coveralls upload containing every matrix profile.
- CodeQL: separate manual-build analysis.

Tool requests are in `mise.toml`; lockfiles select exact versions and checksums.
The weekly dependency workflow verifies updates with `check:full` before opening
a draft PR. Tool updates require releases at least seven days old.
