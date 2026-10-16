# Profiles

Compile one application and select its active flags, arguments, and commands for
each invocation using `ParseOpts.Profiles`:

```go
type options struct {
    Text    string `flag:"foo" profile:"A" default:"hello"`
    Number  int32  `flag:"foo" profile:"B"`
    Verbose bool   `flag:"verbose"`
}

app := cli.NewApplication[options]("example").MustCreate()
defer app.Close()

invocation, err := app.Parse(ctx, &cli.ParseOpts{
    Profiles: []string{"A"},
}, "--foo", "world")
// On success, close invocation after use.
```

In A, `foo` is a string with default `hello`. In B, it is an `int32` with no
declared default. With no profiles, only `verbose` and automatic help are active.
The Go options type stays static; different value types need different fields.

## Selection rules

- No assignment means always active within an active parent.
- An assignment such as `profile:"A|B"` uses OR matching. One matching selected
  profile activates the definition exactly once, even if A and B are both selected.
- Builders use `WithProfiles("A", "B")` / `SetProfiles([]string{"A", "B"})`
  on flag, argument, and command requirements. These replace the assignments;
  an empty slice removes them. Explicit field bindings override tags as usual.
- Names are exact and case-sensitive. Empty names and names containing `|` are
  invalid assignments. Unknown selected profiles are parse errors. Repeating a
  selected profile has no additional effect.
- Only `ParseOpts.Profiles` selects profiles. The library does not interpret a CLI
  flag or environment variable as a profile selector.
- Inactive inputs are absent from token recognition, help, environment lookup,
  declared defaults, required checks, decoding, and runtime resource acquisition.
  The complete options factory and options lifecycle still run normally.
- Filtered positional arguments consume tokens in declaration order without gaps.
  `Application.Argument(index)` and `Invocation.Argument(index)` retain their
  complete-definition indices, including indices belonging to inactive inputs.

The application is an immutable snapshot after `Create`. Concurrent parses can
select different profiles; user callbacks must still be concurrency-safe.

## Commands

```go
cli.NewSimpleApplication("example").WithCommand(
    cli.NewCommand[TextOptions]("convert").WithProfiles("A"),
    cli.NewCommand[NumberOptions]("convert").WithProfiles("B"),
)
```

Variants can have different options types, handlers, inputs, and subcommands.
An inactive command is not recognized as a command and is omitted from help.
Under existing positional-input rules, its name may instead be an argument value.

Each command checks its own assignments against the same global selection. Its
ancestors must also be active. A B-only child under an A-only parent therefore
requires A+B. A profile-free child only requires its parent to be active. Selecting
a child profile never implicitly selects its parent's profiles.

A handlerless application or command requires a subcommand only when it has
active children for the selected profiles. Incomplete command paths fail before
options creation; the default error handler shows the group's help and exits
with code 2. If every child is inactive, no subcommand is required. An explicit
`--help` remains a successful help request.

## Conflict detection

There are no overrides, priorities, or implicit merges. `Create` rejects:

- Conflicting definitions if either has no profile assignments.
- Conflicting definitions whose assignments share a profile.
- Structural errors even in definitions that a later parse never activates.

Disjoint-profile variants can coexist in the compiled application. Selecting a
combination that makes them conflict fails during parsing, before options are
created, even if the input does not use the conflicting definition or requests
help. Conflicts are checked along the selected command path; inactive and
unvisited command subtrees do not invalidate unrelated invocations.

Checks include long names and aliases, short flags, command names and aliases,
argument names, overlapping options-field bindings, inherited flags, variadic
argument ordering, and required parent arguments combined with active subcommands.
Existing codec/default validation timing is preserved: defaults whose syntax is
validated at parse/help time retain that behavior.

## Introspection

```go
for profile := range app.Profiles() { // iter.Seq[string], sorted and deduplicated
    // Includes profiles declared anywhere in the command tree.
}
known := app.ContainsProfile("A")

global, found := app.Flag("foo")
variant, found := app.FlagOfProfile("foo", "A")
short, found := app.ShortFlagOfProfile('f', "A")
command, found := app.CommandOfProfile("convert", "A")
typed, found := cli.FindCommandOfProfile[TextOptions](app, "A", "convert")
```

`Flag`, `ShortFlag`, and `Command` find only profile-free definitions. Their
`OfProfile` methods find explicit assignments without profile-free fallback.
Unknown names/profiles return `found == false` from these metadata lookups.
Long aliases are supported. `Flags()`, `Arguments()`, and `Commands()` expose all
variants in definition order, each once. Definition `Profiles()` methods return
independent copies of their explicit assignments.

`FindCommandOfProfile` resolves a path for one profile, traversing explicitly
assigned and profile-free commands. For mixed-profile paths, obtain the parent
handle first and resolve its child using that child's profile.

Invocation value lookups and `HelpInfo()` reflect that invocation's active schema,
including on help and parse-error responses.
`CommandFromContext[Options](ctx, "parent", "child")` resolves explicit paths
against the selected command stack, including profile-specific variants, aliases,
and paths whose command levels use different profiles. Unselected commands and
commands without created options (such as help invocations) return `found == false`.

## Selected profiles in callbacks and invocations

An invocation exposes its own selected profiles through `InvocationLike`:

```go
for profile := range invocation.Profiles() { // iter.Seq[string]
    // Selected profiles only, sorted and deduplicated.
}
selected := invocation.ContainsProfile("A")
```

This differs from `app.Profiles()`, which lists every known profile. An invocation
retains its selection after `Close`, including help/error invocations, independently
of later changes to the caller's `ParseOpts.Profiles` slice.

Callbacks can access the same selection directly through their context:

```go
profiles, found := cli.ProfilesFromContext(ctx)
if found {
    for profile := range profiles {
        // Available even in options factories and codec decoding.
    }
}
if cli.ContextContainsProfile(ctx, "A") {
    // Profile A was explicitly selected for this invocation.
}
```

The selection is available throughout parsing and in validation, initialization,
execution, help, and error callbacks. `Execute` attaches the invocation's original
selection even when passed a different context. Nested parses use their own
`ParseOpts` selection, including an empty selection, without inheriting outer
profiles or mutating the caller's context. During error handling, the metadata
includes requested unknown profiles so diagnostics can inspect the failed request.

`ProfilesFromContext` returns an empty sequence and `true` for an operation with
no selected profiles. Without CLI profile context, including a nil context, it
returns an empty sequence and `false`. `ContextContainsProfile` returns
`false` when the profile is absent or the context has no selection.

See the runnable [profiles example](../_examples/profiles).
