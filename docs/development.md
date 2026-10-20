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
| `mise run test:shells` | Mandatory interactive Bash/Fish/Nushell/PowerShell/Zsh E2E tests in Linux containers |
| `mise run test:fuzz` | Bounded parser fuzzing |
| `mise run test:benchmarks` | Check comparison adapters and their module |
| `mise run bench` | Measure CLI lifecycle comparisons and update the results page |
| `mise run check:full` | Full quality gate |
| `mise run tools:update` | Refresh locked tool versions |

List all tasks with `mise tasks`. Race detection needs a C toolchain:
GCC on Linux, Xcode Command Line Tools on macOS, or MinGW-w64 on Windows.

## Shell E2E tests

`check:full` includes a mandatory container suite. Install Podman (recommended:
rootless) or Docker; the runner prefers Podman when both are present. On macOS or
Windows, start the runtime's Linux VM first.

```shell
mise run test:shells
mise run test:shells --engine podman
mise run test:shells --engine docker
mise run test:shells --coverage
```

The runner cross-builds static Linux fixture binaries with the project's Go
toolchain. Its temporary build context contains only those binaries and the test
assets, not the repository, Git metadata, or local configuration. No bind mounts,
daemon socket mounts, privileged containers, or host shell installations are needed.
The test containers have no network access; image building needs registry/package
access. Containers and temporary files are cleaned up; image layers remain cached.

The Containerfiles pin Bash **5.2.37**, Nushell **0.114.1**, and PowerShell **7.5.0**
images by manifest digest, Fish **4.0.2** through Alpine package `fish=4.0.2-r0`,
and Zsh **5.9** through `zsh=5.9-r5`.
PowerShell runs separately on Ubuntu (amd64) or Azure
Linux (arm64), keeping its native .NET environment. The runner cross-compiles a Go
test binary with the `shell_e2e` build tag.
Its Go PTY driver uses Linux terminal ioctls from the existing `golang.org/x/sys`
dependency and embedded shell setup scripts. It installs adapters through the
real CLI command, starts interactive Bash/Readline, Fish, Nu/Reedline, PowerShell/PSReadLine, and Zsh/ZLE,
presses Tab, and reads the line-editor buffer. It verifies on-demand Bash loading,
Fish/Nu/Zsh autoload, PowerShell profile loading/preservation, quotes, Unicode, paths, context-dependent values, real `compopt`
NoSpace/KeepOrder behavior, cursor edits, and binary replacement without adapter
reinstallation. Initialization markers prove that requests bypass normal startup.
PowerShell tests also execute completed arguments in both Standard and Legacy
native argument modes, with ASCII pipeline encoding. The adapter targets 5.1+;
Windows PowerShell 5.1 itself is not part of this Linux E2E suite.
Zsh tests cover native menus/KeepOrder, NoSpace, mid-word and Unicode cursor edits,
directory-only completion, preserved startup content/styles, existing compinit,
non-exported custom ZDOTDIR settings, and sourcing the generated autoload file.
Fish tests cover descriptions/order in the native completion API and actual pager,
empty/multiline values, escapes, non-evaluated substitutions, native suffix matching,
directory-only paths, command decorators, preserved config, and binary replacement.
Missing/wrong-version shells and missing runtimes are errors, never skips.

`--coverage` instruments the actual fixture binaries, collects their Go runtime
coverage from both stopped containers using the engine's copy command, and merges
it into `profile.shells.cov` with Go's `covdata` tooling. CI includes this artifact
in the same Coveralls upload as the Go matrix and example profiles, so native
shell execution contributes to the reported library coverage.

The lighter `TestNativeShellCompletion` Go test still uses locally available
shells and can skip unavailable ones. The separate container suite is the required
CI guarantee and also runs in the dependency-update quality gate.

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
- Shell-fixture coverage: `profile.shells.cov`, including the real adapter queries and installation paths.
- `CLI4G_CI=1` enables race/verbose/coverage for Go tests and binary coverage
  for examples; Go tests then write `profile.cov.full`.

## CI

- Go linting: Linux, macOS, Windows.
- Runtime tests: minimum/stable Go on each OS, including example binaries.
- Shell E2E: a required Linux job using rootless Podman and pinned Bash/Fish/Nushell/PowerShell/Zsh versions.
- Shared checks: once on Linux; formats, module archives, workflows, example code,
  benchmark-adapter correctness, fuzzing, and vulnerabilities. Timing measurements
  are run separately on an idle machine, not in CI.
- Coverage: one Coveralls upload containing every matrix profile.
- CodeQL: separate manual-build analysis.

Tool requests are in `mise.toml`; lockfiles select exact versions and checksums.
The weekly dependency workflow verifies updates with `check:full` before opening
a draft PR. Tool updates require releases at least seven days old.
