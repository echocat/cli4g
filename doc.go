// Package cli builds command-line applications with typed options, subcommands,
// flags, arguments, environment values, and automatic help.
//
// Start with [NewApplication], add a handler with [ApplicationRequirement.WithHandler],
// and run [ApplicationRequirement.MustCreateParseAndExecute] from main.
// For caller-managed errors, use [ApplicationRequirement.WithErrorHandlerEnabled]
// and [ApplicationRequirement.CreateParseAndExecute].
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
//
// Builders offer the same inputs. Bind them explicitly with
// [FlagRequirement.WithField] or [ArgumentRequirement.WithField]; names alone do
// not bind fields. Explicit bindings override tags on that field.
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
// A child can borrow an exported struct pointer using parent:"application" or
// parent:"matchingType". MatchingType picks the nearest selected ancestor of the
// exact Go type. Borrowed options remain parent-owned; never close them in the child.
// Anonymous parent fields must not promote Validate, Initialize, or Close methods.
// Use [ApplicationFromContext], [CommandFromContext], or [Command.OptionsFrom]
// to read the current typed options.
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
// Other errors remain operational. [DefaultErrorHandler] writes to stderr and
// exits with 2 for invalid input, 130 for cancellation, or 1 for other errors.
// Disable responses when embedding the library: os.Exit does not run caller defers.
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
// files but do not truncate existing files. Use the codec package for explicit
// append/truncate policies, dates, adapters, and custom value types.
package cli
