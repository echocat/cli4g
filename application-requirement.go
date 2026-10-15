package cli

import (
	"context"
	"reflect"
	"slices"

	"github.com/echocat/cli4g/codec"
)

// NewApplication starts a CLI builder with automatic --help/-h enabled.
// O must be a struct or pointer to a struct; exported tags define inputs.
// By default, input and execution errors use [DefaultErrorHandler].
func NewApplication[O any]() *ApplicationRequirement[O] {
	return &ApplicationRequirement[O]{Help: true}
}

// NewSimpleApplication starts a builder without an options struct.
// Read unbound values through [Invocation.Flag] and [Invocation.Argument].
func NewSimpleApplication() *ApplicationRequirement[NoOptions] {
	return NewApplication[NoOptions]()
}

// ApplicationRequirement configures options, inputs, commands, and handlers.
// Call [ApplicationRequirement.Create] to obtain an independent [Application].
// Use pointer options when callbacks need to retain field assignments.
type ApplicationRequirement[O any] struct {
	Name        string
	Description string
	// Help enables automatic --help/-h throughout the command tree. Constructors
	// enable it by default; a zero-valued requirement has it disabled.
	Help                 bool
	HelpHandler          HelpHandler
	ErrorHandler         ErrorHandler
	errorHandlerDisabled bool

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
	this.Commands = append(this.Commands, v...)
	return this
}

func (this *ApplicationRequirement[O]) SetCommands(v []CommandRequirementLike) {
	this.Commands = v
}

func (this *ApplicationRequirement[O]) OptionsType() reflect.Type {
	return reflect.TypeFor[O]()
}

// WithFlag adds explicit flags. Bind options fields with [FlagRequirement.WithField].
func (this *ApplicationRequirement[O]) WithFlag(v ...FlagRequirementLike) *ApplicationRequirement[O] {
	this.Flags = append(this.Flags, v...)
	return this
}

// WithCodec accepts an immutable codec.Codec[V, R]. Create validates and
// snapshots the codec; names bind it to matching options type tags.
func (this *ApplicationRequirement[O]) WithCodec(v any, names ...string) *ApplicationRequirement[O] {
	this.Codecs = append(this.Codecs, codec.NewRegistration(v, names...))
	return this
}

func (this *ApplicationRequirement[O]) SetCodecs(v []codec.Registration) {
	this.Codecs = slices.Clone(v)
}

func (this *ApplicationRequirement[O]) SetFlags(v []FlagRequirementLike) {
	this.Flags = v
}

// WithArgument adds positional arguments in order. Variadic arguments must be last.
// Bind options fields with [ArgumentRequirement.WithField].
func (this *ApplicationRequirement[O]) WithArgument(v ...ArgumentRequirementLike) *ApplicationRequirement[O] {
	this.Arguments = append(this.Arguments, v...)
	return this
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
