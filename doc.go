// Package cli builds command-line applications with typed options, subcommands,
// flags, arguments, environment values, automatic help, and shell completion.
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
// NewApplication defaults to seven subcommand levels. WithMaxCommandDepth sets
// an explicit uint8 limit; zero permits only the application. Create checks all
// definitions, including inactive profiles. Sibling counts are not limited.
// Parse scratch capacity follows the actual compiled depth and token count,
// rather than reserving the configured maximum for every invocation.
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
// # Shell completion
//
// [ApplicationRequirement.WithCompletion] is an opt-in shortcut. True assigns the
// current [DefaultCompletionCommand] and [DefaultCompletionRequestCommandName];
// false clears both configuration fields, including custom values. There is no
// additional enabled state and Create never fills empty values from defaults.
// The standard engine derives suggestions from the existing CLI definition.
// Users install a native adapter once, without editing their shell configuration:
//
//	tool completion install bash
//	tool completion install nu
//	tool completion install powershell
//	tool completion install zsh
//
// Bash 4.4+ uses the bash-completion package's on-demand loader. Nushell 0.114+
// uses its user autoload directory on the next interactive startup. PowerShell
// 5.1+ registers a native argument completer through static dot-source blocks in
// the installed pwsh/powershell hosts' CurrentUserAllHosts profiles. The installer
// discovers those paths without loading profiles, preserves existing profile
// content and encoding, and writes a UTF-8-with-BOM adapter below the first
// profile's cli4g-completions directory. Both Windows PowerShell and PowerShell 7
// are configured when installed; install again after adding another host. Existing
// execution policies apply. No application process runs when the adapter is loaded.
// Source the installed file (dot-source it in PowerShell) to activate it in an
// existing session. For packaging, supply a
// directory after the shell name or use [Application.CompletionScript] to obtain
// an adapter without installing it. A shell subcommand also prints it to stdout:
//
//	tool completion bash
//
// An explicit PowerShell installation directory writes only the adapter and does
// not edit profiles. PowerShell sends literal AST values and UTF-8 byte cursor
// offsets through an ASCII-safe Base64/JSON transport, including on Windows
// PowerShell 5.1's legacy native pipeline. Tokens after the cursor remain available
// for dependencies. Variables and expressions are not evaluated to collect values.
// Candidates include descriptions and use literal quoting, including legacy native
// argument escaping. Paths normally use PowerShell's filename completer; literal
// Unix colon filenames use filesystem enumeration because the provider interprets
// colons as drives. Candidate order is preserved. The user's PSReadLine binding
// determines whitespace insertion; the adapter does not change that binding.
// See [Application.InstallCompletion] and [CompletionProtocolVersion] for details.
//
// Zsh 5.8+ uses its native compinit/fpath autoloading. The default installer writes
// _Name below XDG_DATA_HOME/zsh/site-functions (otherwise
// ~/.local/share/zsh/site-functions), and
// appends a static block to .zshrc in Zsh's startup directory. That directory is
// discovered from the installed zsh, including non-exported ZDOTDIR settings in
// .zshenv. Discovery runs only during installation. The block extends fpath,
// initializes compinit if needed, and registers an autoload function; it starts
// neither the application nor the adapter at shell startup. Existing user content,
// widgets and completion styles are preserved. Explicit directories write only
// the autoload file for packaging. Source that file after compinit for an existing
// session. Run installation again after relocating the startup directory.
//
// The Zsh adapter uses native words and cursor prefix/suffix fragments, including
// unfinished quotes and arguments after the cursor. It never evaluates expressions
// while collecting dependency values. Candidates carry descriptions; compadd
// provides literal quoting and NoSpace/KeepOrder, and _files handles native paths.
// ZLE settings control whether completion operates inside a word (COMPLETE_IN_WORD)
// and how menu selection inserts spaces. The adapter retains those settings.
//
// ApplicationRequirement.CompletionCommand is an ordinary CommandRequirementLike.
// Nil means absent. Create compiles it alongside Commands without modifying the
// builder list or shared default. It uses the same parsing, inherited inputs,
// validation, initialization, execution, cleanup, profiles, help and depth limits
// as every other command. Parent required flags also apply; required positional
// parent arguments cannot coexist with subcommands. Commands/Command and typed
// FindCommand expose the actual compiled command, not a metadata substitute.
//
// [DefaultCompletionCommand] is a replaceable, preconfigured command requirement.
// [NewCompletionCommand] creates a fresh version for per-application changes:
//
//	WithCompletion(true).
//	    WithCompletionCommand(NewCompletionCommand().WithName("shells"))
//
// Supply any custom CommandRequirement through WithCompletionCommand instead.
// [CompletionApplicationFromContext] gives ordinary custom handlers access to the
// renderer, installer and direct completion API without knowing the root options
// type. Configure global defaults before use, without concurrent modification;
// already compiled applications keep independent definitions.
//
// ApplicationRequirement.CompletionRequestCommandName independently configures
// the special adapter entry. Empty disables it. WithCompletion(true) assigns the
// current replaceable DefaultCompletionRequestCommandName (initially "__complete").
// This name is not registered as a Command and never appears in Commands(), help
// or suggestions. It must not collide with any root command/alias, including
// inactive profiles. Keep it stable across upgrades, or regenerate adapters/provide
// a compatibility entry point when changing it. Public command changes do not
// affect installed adapters; the wire protocol is independent of either name.
//
// Adapters contain no flags, commands, enum snapshots, or version-specific binary
// paths. At completion time they query the currently selected executable through
// [CompletionProtocolVersion]; replacing the binary therefore updates suggestions
// automatically. Sourcing an adapter starts no application process. Each query
// starts one process; startup code and Go init functions remain the application's
// responsibility. Query callbacks receive [CompletionQueryTimeout] as a cooperative
// deadline. cli4g does not maintain a background service or persistent result cache.
//
// Commands, aliases, visible flags, enum values, bools, and file inputs are
// completed automatically. Noncumulative flags already used are omitted from name
// suggestions. Slice codecs delegate to their element codec, and map codecs
// delegate the value after key=. codec.Completer adds context-dependent values,
// such as namespaces selected by --kubeconfig and --context. Its input contains
// canonical flag occurrence maps, arguments by name and position, command scopes,
// active profiles, and the current word/cursor target. Neither name completion nor
// metadata inspection invokes value completers. Providers return unquoted tokens
// and optional descriptions; shell adapters own quoting and display.
// The regular options structs and command definitions are the single CLI schema;
// the built-in shell entry point constructs the completion context automatically.
// To attach a function to an existing codec, use codec.WithCompletionHandler:
//
//	WithCodec(codec.WithCompletionHandler(codec.NewString(), completeNamespaces), "namespace")
//
// A type:"namespace" tag selects that codec for the regular field. No custom
// codec wrapper type or ToRaw implementation is needed. For builder-defined
// inputs, [FlagRequirement.WithCompletionHandler] and
// [ArgumentRequirement.WithCompletionHandler] customize one flag/argument without
// changing its parsing codec. Their handler runs first; Handled=false delegates
// to the codec, then enum/bool/path defaults. See [CompletionHandler].
//
// [Application.Complete] is the direct Go API. It accepts a [CompletionRequest]
// and [ParseOpts], tolerates unknown and incomplete input, and never constructs
// options or runs their lifecycle. Raw CLI tokens remain available even if they
// would not decode. Known flags delimit missing required values for recovery;
// literal option-looking values can be supplied inline. Other complete arguments
// after the cursor remain visible when supplied by the adapter/caller.
// codec.CompletionInput.Effective lazily reads dependencies with CLI > environment
// > declared-default precedence and identifies their source. Factory-only values
// are unavailable. Defaults and other inputs are not resolved to runtime resources.
// Shell expressions/substitutions are not executed to discover dependency values.
//
// Direct API errors are returned without invoking the application's error handler.
// Adapter queries turn failures into empty answers so terminal editing remains
// usable. Providers doing I/O must honor cancellation and close their own request
// resources. A provider-side in-memory cache does not span per-query processes;
// persistent caches must account for all configuration/context dependencies.
//
// Only a nonempty configured request name is recognized specially in the first
// token by Parse and ParseAndExecute. Parse returns a metadata-only invocation;
// Execute serves the query using context streams, without constructing, validating
// or initializing ordinary options or running business handlers. Use -- before a
// literal request-name root argument. Standard --help retains its usual bypass
// behavior for the ordinary public command. The direct Complete API remains
// available regardless of either configuration field.
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
// ParseAndExecute prepares standard streams once for the entire managed
// operation. The invocation and profile metadata share one immutable context
// overlay; early parse contexts retain their own profile snapshot.
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
// Argument value labels preserve the declared name and case: <userName>. Flag
// values use --flag=<value>; the generated value name is the final lowercased
// word, splitting at '.', '-', '_' and case boundaries (log.colorMode -> <mode>).
// Explicit placeholder tags or WithPlaceholder text keep their case and take
// precedence; surrounding <...> are added once. Optional arguments use [<name>],
// variadic values <name>..., and optional single flag values --flag[=<value>].
// Template output is automatically aligned and word-wrapped. Tabs separate
// columns; blank lines separate sections, and nested indentation gets its own
// column widths. Terminal width is read from the selected output for each help
// rendering; files, pipes and other unsuccessful queries fall back to 80.
// [TemplatedHelpHandlerRequirement.WithWidth] selects a fixed width;
// [TemplatedHelpHandlerRequirement.WithFormatting] disables layout for exact
// template output. Width counts Unicode code points, not terminal cell widths;
// unbreakable words/labels may exceed it. CRLF is normalized to LF by formatting.
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
