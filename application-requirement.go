package cli

import (
	"context"
	"reflect"
	"slices"

	"github.com/echocat/cli4g/codec"
)

// NewApplication starts a CLI builder with automatic --help/-h enabled.
// O must be a struct or pointer to a struct; exported tags define inputs.
// By default, input and execution errors use [DefaultErrorHandler], and missing
// context streams fall back to process stdin/stdout/stderr. MaxCommandDepth is 7.
// Completion is opt-in through [ApplicationRequirement.WithCompletion].
// The first optional string sets the name; further strings form the description,
// joined with spaces. With no strings, configure the name later; Create validates
// the completed builder.
func NewApplication[O any](nameAndDescription ...string) *ApplicationRequirement[O] {
	name, description := parseNameAndDescription(nameAndDescription)
	return &ApplicationRequirement[O]{Name: name, Description: description, Help: true, FallbackToStdio: true, MaxCommandDepth: 7}
}

// NewSimpleApplication starts a builder without an options struct.
// Read unbound values through [Invocation.Flag] and [Invocation.Argument].
// Optional strings set the name and description as in [NewApplication].
func NewSimpleApplication(nameAndDescription ...string) *ApplicationRequirement[NoOptions] {
	return NewApplication[NoOptions](nameAndDescription...)
}

// ApplicationRequirement configures options, inputs, commands, and handlers.
// Call [ApplicationRequirement.Create] to obtain an independent [Application].
// Use pointer options when callbacks need to retain field assignments.
// Without a Handler, active commands require selecting a command or --help.
type ApplicationRequirement[O any] struct {
	Name        string
	Description string
	// Help enables automatic --help/-h throughout the command tree. Constructors
	// enable it by default; a zero-valued requirement has it disabled.
	Help bool
	// CompletionCommand is an optional ordinary root command, compiled alongside
	// Commands. Nil means absent; Create never substitutes a default. Configure it
	// directly or use WithCompletion(true) to select DefaultCompletionCommand.
	CompletionCommand CommandRequirementLike
	// CompletionRequestCommandName independently enables the special adapter entry
	// point. Empty disables it; Create never substitutes a default. Keep a configured
	// name stable across binary updates. This entry bypasses the options lifecycle.
	CompletionRequestCommandName string
	HelpHandler                  HelpHandler
	ErrorHandler                 ErrorHandler
	errorHandlerDisabled         bool
	// FallbackToStdio supplies process stdin/stdout/stderr only when no stream
	// configuration exists in the operation's context. Constructors enable it by default;
	// a zero-valued requirement has it disabled.
	FallbackToStdio bool
	// MaxCommandDepth limits user-defined subcommand levels below the application (depth 0).
	// Zero permits only the application; constructors default to 7. Create checks
	// every definition, including inactive profiles, and snapshots the limit.
	MaxCommandDepth uint8

	Commands  []CommandRequirementLike
	Codecs    []codec.Registration
	Flags     []FlagRequirementLike
	Arguments []ArgumentRequirementLike

	CreateOptionsHandler func(context.Context) (O, error)
	ValidateHandler      func(context.Context, O) error
	InitializeHandler    func(context.Context, O) error
	Handler              func(context.Context, O) error
	CloseHandler         func(O) error
}

func (this *ApplicationRequirement[O]) WithName(v string) *ApplicationRequirement[O] {
	this.SetName(v)
	return this
}

func (this *ApplicationRequirement[O]) SetName(v string) {
	this.Name = v
}

func (this *ApplicationRequirement[O]) WithDescription(v string) *ApplicationRequirement[O] {
	this.SetDescription(v)
	return this
}

func (this *ApplicationRequirement[O]) SetDescription(v string) {
	this.Description = v
}

// WithHelp enables or completely disables automatic help recognition and name
// reservation throughout the application, including every subcommand.
func (this *ApplicationRequirement[O]) WithHelp(v bool) *ApplicationRequirement[O] {
	this.SetHelp(v)
	return this
}

func (this *ApplicationRequirement[O]) SetHelp(v bool) { this.Help = v }

// WithCompletion is a shortcut for configuring the public command and special
// request entry. True assigns the current DefaultCompletionCommand and
// DefaultCompletionRequestCommandName; false clears both, including custom values.
// Call this before individual overrides. There is no separate enabled state, and
// neither value is implicitly restored during Create. The direct Complete API
// remains available independently of both settings.
//
// The public command is an ordinary command: parent input requirements, lifecycle,
// profiles, help and depth limits apply normally. Only the request entry bypasses
// parsing/validation/initialization. The standard handler derives suggestions from
// the existing CLI schema and optional flag/argument/codec completion handlers.
func (this *ApplicationRequirement[O]) WithCompletion(v bool) *ApplicationRequirement[O] {
	this.SetCompletion(v)
	return this
}

// SetCompletion is the setter counterpart of WithCompletion.
func (this *ApplicationRequirement[O]) SetCompletion(v bool) {
	if v {
		this.CompletionCommand = DefaultCompletionCommand
		this.CompletionRequestCommandName = DefaultCompletionRequestCommandName
	} else {
		this.CompletionCommand = nil
		this.CompletionRequestCommandName = ""
	}
}

// WithCompletionCommand configures an ordinary public management command.
// Nil removes it. Use NewCompletionCommand().WithName("shells") to customize the
// standard command independently, or supply any CommandRequirement with your own
// options and handlers. Its name belongs to that command, not a separate setting.
// The request entry is unaffected. Create compiles the command normally without
// modifying the requirement or adding it permanently to Commands.
func (this *ApplicationRequirement[O]) WithCompletionCommand(v CommandRequirementLike) *ApplicationRequirement[O] {
	this.SetCompletionCommand(v)
	return this
}

// SetCompletionCommand is the setter counterpart of WithCompletionCommand.
func (this *ApplicationRequirement[O]) SetCompletionCommand(v CommandRequirementLike) {
	this.CompletionCommand = v
}

// WithCompletionRequestCommandName names the hidden adapter request command.
// Empty disables it. True in WithCompletion selects the current default variable.
// Generated adapters use the captured name, so choose it before installation and
// retain it across upgrades. It must be a single command word, distinct from all
// root commands/aliases, including inactive profiles. This out-of-band entry is
// not registered as a command and is absent from Commands(), help and suggestions.
func (this *ApplicationRequirement[O]) WithCompletionRequestCommandName(v string) *ApplicationRequirement[O] {
	this.SetCompletionRequestCommandName(v)
	return this
}

// SetCompletionRequestCommandName is the setter counterpart of WithCompletionRequestCommandName.
func (this *ApplicationRequirement[O]) SetCompletionRequestCommandName(v string) {
	this.CompletionRequestCommandName = v
}

// WithFallbackToStdio controls fallback to process stdin/stdout/stderr when the
// context has no streams. NewApplication enables it by default. Explicit streams
// from [codec.ContextWithStdio], including nil streams, always take precedence.
func (this *ApplicationRequirement[O]) WithFallbackToStdio(v bool) *ApplicationRequirement[O] {
	this.SetFallbackToStdio(v)
	return this
}

func (this *ApplicationRequirement[O]) SetFallbackToStdio(v bool) { this.FallbackToStdio = v }

// WithMaxCommandDepth limits subcommand nesting. Zero allows only the application.
// The number of sibling commands is not limited. Create rejects deeper paths.
func (this *ApplicationRequirement[O]) WithMaxCommandDepth(v uint8) *ApplicationRequirement[O] {
	this.SetMaxCommandDepth(v)
	return this
}

func (this *ApplicationRequirement[O]) SetMaxCommandDepth(v uint8) { this.MaxCommandDepth = v }

// WithHelpHandler replaces DefaultHelpHandler. Execute invokes it only for a
// help invocation; normal handlers and the options lifecycle are skipped.
func (this *ApplicationRequirement[O]) WithHelpHandler(v HelpHandler) *ApplicationRequirement[O] {
	this.SetHelpHandler(v)
	return this
}

func (this *ApplicationRequirement[O]) SetHelpHandler(v HelpHandler) { this.HelpHandler = v }

// WithErrorHandler replaces DefaultErrorHandler for input and operational errors.
// The handler runs after cleanup. A non-nil return replaces the error; nil
// preserves it. ErrValidation identifies input failures regardless of their phase.
func (this *ApplicationRequirement[O]) WithErrorHandler(v ErrorHandler) *ApplicationRequirement[O] {
	this.SetErrorHandler(v)
	return this
}

func (this *ApplicationRequirement[O]) SetErrorHandler(v ErrorHandler) {
	this.ErrorHandler = v
}

// WithErrorHandlerEnabled controls this application's error response independently
// of the global default. Enabled is the default, including on zero requirements.
// A nil handler selects DefaultErrorHandler; disabling returns errors normally.
func (this *ApplicationRequirement[O]) WithErrorHandlerEnabled(v bool) *ApplicationRequirement[O] {
	this.SetErrorHandlerEnabled(v)
	return this
}

func (this *ApplicationRequirement[O]) SetErrorHandlerEnabled(v bool) {
	this.errorHandlerDisabled = !v
}

// WithCommand adds subcommands. Different commands may use different options types.
func (this *ApplicationRequirement[O]) WithCommand(v ...CommandRequirementLike) *ApplicationRequirement[O] {
	this.AddCommand(v...)
	return this
}

func (this *ApplicationRequirement[O]) AddCommand(v ...CommandRequirementLike) {
	this.Commands = append(this.Commands, v...)
}

func (this *ApplicationRequirement[O]) SetCommands(v []CommandRequirementLike) {
	this.Commands = v
}

func (this *ApplicationRequirement[O]) OptionsType() reflect.Type {
	return reflect.TypeFor[O]()
}

// WithFlag adds explicit flags. Bind options fields with [FlagRequirement.WithField].
func (this *ApplicationRequirement[O]) WithFlag(v ...FlagRequirementLike) *ApplicationRequirement[O] {
	this.AddFlag(v...)
	return this
}

func (this *ApplicationRequirement[O]) AddFlag(v ...FlagRequirementLike) {
	this.Flags = append(this.Flags, v...)
}

func (this *ApplicationRequirement[O]) SetFlags(v []FlagRequirementLike) {
	this.Flags = v
}

// WithCodec accepts an immutable codec.Codec[V, R]. Create validates and
// snapshots the codec; names bind it to matching options type tags.
func (this *ApplicationRequirement[O]) WithCodec(v any, names ...string) *ApplicationRequirement[O] {
	this.AddCodec(v, names...)
	return this
}

func (this *ApplicationRequirement[O]) AddCodec(v any, names ...string) {
	this.Codecs = append(this.Codecs, codec.NewRegistration(v, names...))
}

func (this *ApplicationRequirement[O]) SetCodecs(v []codec.Registration) {
	this.Codecs = slices.Clone(v)
}

// WithArgument adds positional arguments in order. Variadic arguments must be last.
// Bind options fields with [ArgumentRequirement.WithField].
func (this *ApplicationRequirement[O]) WithArgument(v ...ArgumentRequirementLike) *ApplicationRequirement[O] {
	this.AddArgument(v...)
	return this
}

func (this *ApplicationRequirement[O]) AddArgument(v ...ArgumentRequirementLike) {
	this.Arguments = append(this.Arguments, v...)
}

func (this *ApplicationRequirement[O]) SetArguments(v []ArgumentRequirementLike) {
	this.Arguments = v
}

// WithCreateOptionsHandler supplies fresh options before input decoding.
// The concurrent-safe factory must return non-nil pointer options. Returned
// options are cleaned up even on error; unreturned resources remain caller-owned.
// Factory values do not satisfy required inputs. Initialize resources separately.
func (this *ApplicationRequirement[O]) WithCreateOptionsHandler(v func(context.Context) (O, error)) *ApplicationRequirement[O] {
	this.SetCreateOptionsHandler(v)
	return this
}

func (this *ApplicationRequirement[O]) SetCreateOptionsHandler(v func(context.Context) (O, error)) {
	this.CreateOptionsHandler = v
}

// WithInitializeHandler prepares validated options after their Initialize method.
// Use pointer options to retain assignments. Release retained resources through
// [ApplicationRequirement.WithCloseHandler] or options' Close, including partial failures.
func (this *ApplicationRequirement[O]) WithInitializeHandler(v func(context.Context, O) error) *ApplicationRequirement[O] {
	this.SetInitializeHandler(v)
	return this
}

func (this *ApplicationRequirement[O]) SetInitializeHandler(v func(context.Context, O) error) {
	this.InitializeHandler = v
}

// WithHandler receives this application's typed options. During invocation
// execution, InvocationFromContext and ContainerFromContext expose the complete
// invocation and the currently executing container.
func (this *ApplicationRequirement[O]) WithHandler(v func(context.Context, O) error) *ApplicationRequirement[O] {
	this.SetHandler(v)
	return this
}

func (this *ApplicationRequirement[O]) SetHandler(v func(context.Context, O) error) {
	this.Handler = v
}

// WithCloseHandler provides cleanup for every created options instance, even
// after failed decoding, validation, or partial initialization. It must tolerate
// partially populated options. Options implementing io.Closer are also closed.
func (this *ApplicationRequirement[O]) WithCloseHandler(v func(O) error) *ApplicationRequirement[O] {
	this.SetCloseHandler(v)
	return this
}

func (this *ApplicationRequirement[O]) SetCloseHandler(v func(O) error) {
	this.CloseHandler = v
}

// WithValidateHandler checks fully decoded options after their Validate method
// and before any initializer runs. It must not mutate options or acquire resources.
// Mark invalid input with NewValidationErrorf; unmarked failures remain operational.
func (this *ApplicationRequirement[O]) WithValidateHandler(v func(context.Context, O) error) *ApplicationRequirement[O] {
	this.SetValidateHandler(v)
	return this
}

func (this *ApplicationRequirement[O]) SetValidateHandler(v func(context.Context, O) error) {
	this.ValidateHandler = v
}
