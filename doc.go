// Package cli builds command-line applications with typed options, subcommands,
// flags, arguments, environment values, and automatic help.
//
// Start with [NewApplication], add a handler with [ApplicationRequirement.WithHandler],
// and run [ApplicationRequirement.MustCreateParseAndExecute] from main.
// For caller-managed errors, use [ApplicationRequirement.WithErrorHandlerEnabled]
// and [ApplicationRequirement.CreateParseAndExecute].
//
// # Builder constructors
//
// [NewApplication], [NewSimpleApplication], [NewCommand], and [NewSimpleCommand]
// accept optional name and description strings:
//
//	app := NewSimpleApplication("app", "Example application.")
//	command := NewSimpleCommand("run", "Run", "the application.")
//	app.WithCommand(command)
//
// [NewFlag] and [NewArgument] accept the same optional strings after their codec.
// The first string is the primary name; all subsequent strings form the
// description, joined with one space between each part. Additional strings are
// never aliases: add flag/command aliases with WithAliases. WithName and
// WithDescription can still override constructor metadata. With no strings,
// configure the name later; Create validates the completed definition.
//
// # Options and bindings
//
// Use exported fields in a struct or pointer-to-struct options type:
//
//	type Options struct {
//	    Name    string   `argument:"name" required:"true"`
//	    Port    uint32   `flag:"port" short:"p" env:"PORT" default:"8080"`
//	    Labels  []string `flag:"label"`
//	    Verbose bool     `flag:"verbose" short:"v"`
//	}
//
// Supported tags:
//
//   - flag|argument=<NAME>: name an input; `|` separates flag aliases
//   - short=<RUNE>: names single-rune aliases
//   - description=<STRING>: displays more information for help
//   - placeholder=<STRING>: define how the placeholder is displayed in help
//   - env=<NAME>: names environment sources; `|` separates fallback names, in order
//   - default=<VALUE>: supplies a single input token
//   - required=<true|false>: demands explicit input
//   - type=<NAME>: selects a named codec registered with [ApplicationRequirement.WithCodec]
//   - visible=<true|false>: If `false`, hides a flag from default help
//   - profile=<NAME>: selects profiles for an input; `|` separates alternatives
//
// Builders offer the same inputs. Bind them explicitly with
// [FlagRequirement.WithField] or [ArgumentRequirement.WithField]; names alone do
// not bind fields. Explicit bindings override tags on that field.
// Tagged T or *T fields also automatically adapt codec.Value (Set/String) and
// encoding.TextMarshaler/TextUnmarshaler implementations on *T. Their zero value
// must be ready for Set/UnmarshalText. Explicit/registered codecs take precedence;
// codec.Value wins when both interfaces apply. Text defaults remain input strings,
// and UnmarshalText runs only when resolving an invocation's fresh value.
//
// The enum:"auto|always|never" tag restricts a string/string-based field to literal
// choices and publishes them in help. It selects a field-local codec and cannot
// be combined with type. For typed alternatives, descriptions, or custom parsing,
// register codec.NewEnum, codec.NewMarshallerEnum, or codec.NewValueEnum instead.
// Custom codecs can publish their complete valid set through codec.Enum[R].
//
// # Input rules
//
//   - Flags accept `--name=value`, `--name value`, short flags, and short clusters.
//   - Bare bool flags mean true; use `--verbose=false` for false.
//   - Slice flags collect one element per occurrence, preserving order and duplicates.
//   - Slice arguments collect remaining values and must be last.
//   - Map flags collect one key=value pair per occurrence; duplicate keys are errors.
//   - Map arguments collect remaining pairs and must be last. Map helpers use string keys.
//   - Use `--` before positional values starting with `-`, including negative numbers.
//   - Subcommands are selected before the first positional value; flags are inherited.
//
// Flag precedence is CLI, environment, declared default, options factory, zero.
// Arguments omit the environment step. CLI slices and maps replace defaults/factory values.
// Inputs are not shell-split or CSV-split. Required inputs cannot have defaults;
// factory values do not satisfy them. Required parent arguments cannot precede
// subcommands. See [FlagRequirement], [ArgumentRequirement], and [ParseOpts].
//
// # Commands and parent options
//
// Add typed commands with [NewCommand] and [ApplicationRequirement.WithCommand].
// An application or command without its own handler requires a subcommand when
// it has active children. An incomplete path is a validation error before options
// creation; the default error handler shows the target's help and exits with 2.
// Explicit --help exits successfully. A node with its own handler can be invoked
// directly, and a node without active children can be used for parsing alone.
// A child can borrow an exported struct pointer using parent:"application" or
// parent:"matchingType". MatchingType picks the nearest selected ancestor of the
// exact Go type. Borrowed options remain parent-owned; never close them in the child.
// Anonymous parent fields must not promote Validate, Initialize, or Close methods.
// Use [ApplicationFromContext], [CommandFromContext], or [Command.OptionsFrom]
// to read the current typed options.
//
// # Profiles
//
// Assign inputs with profile:"A|B" or [FlagRequirement.WithProfiles] and
// [ArgumentRequirement.WithProfiles]; use [CommandRequirement.WithProfiles] for
// commands. [ParseOpts.Profiles] is the only profile-selection mechanism. A
// definition without assignments is always active within its active parent;
// otherwise any selected matching profile activates it once. Child commands do
// not activate ancestors: a B child under an A parent requires both A and B.
//
// Create rejects conflicting profile-free or overlapping-profile definitions.
// Disjoint variants may use the same name with different types and defaults;
// selecting a conflicting combination is a parse error, even with --help or no
// input for that definition. Unknown profiles also fail before scanning tokens.
// Inactive definitions do not participate in help, defaults, environment lookup,
// required checks, decoding, or resource acquisition. Active positional arguments
// consume inputs in declaration order without gaps; Argument(index) retains the
// stable complete-definition index, not the filtered input position.
//
// [Application.Profiles] lists all known profiles alphabetically, including
// commands and descendants; [Application.ContainsProfile] checks exact names.
// Container Flags and Commands list every variant. Flag, ShortFlag and Command
// find only profile-free definitions; their OfProfile counterparts find explicit
// assignments without fallback. [FindCommandOfProfile] provides typed path access.
// Invocations expose the values and help metadata of their own selected schema,
// allowing concurrent parses with different selections on the same application.
// [Invocation.Profiles] and [Invocation.ContainsProfile] expose the invocation's
// selected profiles. [ProfilesFromContext] and [ContextContainsProfile] make
// this selection available throughout parsing and execution, including factories,
// decoding, help, and error callbacks.
//
// # Lifecycle and errors
//
// [Application.Parse] resolves inputs, then validates and initializes options.
// [Invocation.Execute] runs handlers from application to selected leaf.
// Always close a successful [Invocation], then its [Application].
// [Application.ParseAndExecute] manages invocation cleanup for you.
// Pointer options retain field assignments; struct callbacks receive copies.
// See [Validatable], [Initializable], and [ApplicationRequirement.WithCloseHandler].
//
// Mark invalid user input with [NewValidationError] or [NewValidationErrorf].
// Other errors remain operational. [DefaultErrorHandler] writes to context stderr and
// exits with 2 for invalid input, 130 for cancellation, or 1 for other errors.
// Disable responses when embedding the library: os.Exit does not run caller defers.
// Built-in codec failures describe the required value without echoing the input.
// Use NewValidationError("an integer expected", cause) to retain a diagnostic
// cause without appending its text. Error displays only the message; errors.Is
// and errors.As still traverse the causes. Unmarked codec/validation failures
// receive neutral messages and retain their original classification.
// Warning: author-supplied ValidationError messages are displayed verbatim and
// must not contain input or confidential data. NewValidationErrorf with %w does
// include the cause text; prefer separate messages and causes for sensitive input.
//
// # Context streams
//
// Supply borrowed streams with [ContextWithStdio](ctx, stdin, stdout, stderr).
// Consumers use [StdioOfContext], [StdinOfContext], [StdoutOfContext] or [StderrOfContext].
// [Print], [Printf] and [Println] write to context stdout; their Err* counterparts
// write to context stderr. Missing output is a no-op.
// NewApplication enables process-stdio fallback when no streams are set;
// disable it with [ApplicationRequirement.WithFallbackToStdio]. Explicit streams
// always take precedence, including nil stdin/stdout, and are never closed by
// the library. Nil stderr uses stdout; pass io.Discard to suppress stderr.
// Without context streams or fallback, the getters return nil.
// Pass the same context to Parse and Execute when calling them separately:
// file aliases resolve during Parse and cannot be replaced later by Execute.
// Built-in help/error handlers use context stderr, or io.Discard when missing.
// Their WithOutput methods can select an explicit writer instead.
//
// # Help and files
//
// --help/-h is enabled by default and reserves those names. Help stops scanning
// immediately, skips options creation, and runs only the help handler. Earlier
// syntax errors and invalid displayed defaults still fail. Customize with
// [ApplicationRequirement.WithHelpHandler] and [NewTemplatedHelpHandler].
//
// File tags fileMode:"read|write|readWrite" and allowSpecial:"true" configure
// *os.File inputs. Invocations own codec-opened handles. Write modes create missing
// files but do not truncate existing files. With allowStdio:"true", "-"/"stdin"
// in read mode borrow context stdin, and "-"/"stdout" in write mode borrow context
// stdout. These file inputs require non-nil *os.File streams; arbitrary readers
// and writers can be used directly through the context helpers.
// Invocations never close these streams. Use ./stdin, ./stdout or ./- for literal
// paths with opt-in enabled. Use the codec package for explicit append/truncate
// policies, dates, adapters, and custom value types.
package cli
