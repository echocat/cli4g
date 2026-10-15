package cli

import (
	"context"
	"reflect"
	"slices"

	"github.com/echocat/cli4g/codec"
)

// NewCommand starts a visible command builder with struct or pointer-to-struct options O.
// Attach it using [ApplicationRequirement.WithCommand] or [CommandRequirement.WithCommand].
func NewCommand[O any]() *CommandRequirement[O] {
	// Unlike a zero-valued requirement, constructed commands are visible by default.
	return &CommandRequirement[O]{Visible: true}
}

// NewSimpleCommand starts a command without an options struct.
func NewSimpleCommand() *CommandRequirement[NoOptions] {
	return NewCommand[NoOptions]()
}

// CommandRequirement configures a command's names, options, inputs, and handlers.
// Struct callbacks receive copies; use pointer options to retain assignments.
type CommandRequirement[O any] struct {
	// Names contains the primary name followed by aliases.
	Names []string

	Description string
	Visible     bool

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

// WithName replaces the primary name and optional aliases. WithAliases appends aliases.
func (this *CommandRequirement[O]) WithName(v ...string) *CommandRequirement[O] {
	this.SetNames(v)
	return this
}

func (this *CommandRequirement[O]) WithAliases(v ...string) *CommandRequirement[O] {
	this.Names = append(this.Names, v...)
	return this
}

func (this *CommandRequirement[O]) SetNames(v []string) {
	this.Names = v
}

func (this *CommandRequirement[O]) WithDescription(v string) *CommandRequirement[O] {
	this.SetDescription(v)
	return this
}

func (this *CommandRequirement[O]) SetDescription(v string) {
	this.Description = v
}

// WithVisible controls whether the command appears in help listings.
func (this *CommandRequirement[O]) WithVisible(v bool) *CommandRequirement[O] {
	this.SetVisible(v)
	return this
}

func (this *CommandRequirement[O]) SetVisible(v bool) {
	this.Visible = v
}

func (this *CommandRequirement[O]) WithCommand(v ...CommandRequirementLike) *CommandRequirement[O] {
	this.AddCommand(v...)
	return this
}

func (this *CommandRequirement[O]) AddCommand(v ...CommandRequirementLike) {
	this.Commands = append(this.Commands, v...)
}

func (this *CommandRequirement[O]) SetCommands(v []CommandRequirementLike) {
	this.Commands = v
}

func (this *CommandRequirement[O]) OptionsType() reflect.Type {
	return reflect.TypeFor[O]()
}

func (this *CommandRequirement[O]) WithFlag(v ...FlagRequirementLike) *CommandRequirement[O] {
	this.AddFlag(v...)
	return this
}

func (this *CommandRequirement[O]) AddFlag(v ...FlagRequirementLike) {
	this.Flags = append(this.Flags, v...)
}

func (this *CommandRequirement[O]) SetFlags(v []FlagRequirementLike) {
	this.Flags = v
}

// WithCodec accepts an immutable codec.Codec[V, R]; Create validates it.
func (this *CommandRequirement[O]) WithCodec(v any, names ...string) *CommandRequirement[O] {
	this.AddCodec(v, names...)
	return this
}

func (this *CommandRequirement[O]) AddCodec(v any, names ...string) {
	this.Codecs = append(this.Codecs, codec.NewRegistration(v, names...))
}

func (this *CommandRequirement[O]) SetCodecs(v []codec.Registration) {
	this.Codecs = slices.Clone(v)
}

func (this *CommandRequirement[O]) WithArgument(v ...ArgumentRequirementLike) *CommandRequirement[O] {
	this.AddArgument(v...)
	return this
}

func (this *CommandRequirement[O]) AddArgument(v ...ArgumentRequirementLike) {
	this.Arguments = append(this.Arguments, v...)
}

func (this *CommandRequirement[O]) SetArguments(v []ArgumentRequirementLike) {
	this.Arguments = v
}

// WithCreateOptionsHandler supplies this command's invocation-owned options
// before decoding. It runs only for selected commands and must return fresh
// mutable state. Non-nil returned options are owned even on error; nil pointers
// are invalid. Factory values do not satisfy Required. Resources not returned
// remain the factory's responsibility; acquire resources in Initialize.
func (this *CommandRequirement[O]) WithCreateOptionsHandler(v func(context.Context) (O, error)) *CommandRequirement[O] {
	this.SetCreateOptionsHandler(v)
	return this
}

func (this *CommandRequirement[O]) SetCreateOptionsHandler(v func(context.Context) (O, error)) {
	this.CreateOptionsHandler = v
}

// WithInitializeHandler runs after every selected scope has been decoded and
// validated. Resources retained by owned options are closed on any later error;
// other partial resources remain the handler's responsibility.
// Use pointer options when the handler needs to attach resources or change
// fields on the owned instance; assignments to struct copies are not retained.
func (this *CommandRequirement[O]) WithInitializeHandler(v func(context.Context, O) error) *CommandRequirement[O] {
	this.SetInitializeHandler(v)
	return this
}

func (this *CommandRequirement[O]) SetInitializeHandler(v func(context.Context, O) error) {
	this.InitializeHandler = v
}

// WithHandler receives this command's typed options. During invocation execution,
// InvocationFromContext and ContainerFromContext expose the complete invocation
// and the currently executing container.
func (this *CommandRequirement[O]) WithHandler(v func(context.Context, O) error) *CommandRequirement[O] {
	this.SetHandler(v)
	return this
}

func (this *CommandRequirement[O]) SetHandler(v func(context.Context, O) error) {
	this.Handler = v
}

// WithCloseHandler provides cleanup for every created options instance, even
// after failed decoding, validation, or partial initialization. It must tolerate
// partially populated options. Options implementing io.Closer are also closed.
func (this *CommandRequirement[O]) WithCloseHandler(v func(O) error) *CommandRequirement[O] {
	this.SetCloseHandler(v)
	return this
}

func (this *CommandRequirement[O]) SetCloseHandler(v func(O) error) {
	this.CloseHandler = v
}

// WithValidateHandler checks fully decoded options after their Validate method
// and before any initializer runs. It must not mutate options or acquire resources.
// Mark invalid input with NewValidationErrorf; unmarked failures remain operational.
func (this *CommandRequirement[O]) WithValidateHandler(v func(context.Context, O) error) *CommandRequirement[O] {
	this.SetValidateHandler(v)
	return this
}

func (this *CommandRequirement[O]) SetValidateHandler(v func(context.Context, O) error) {
	this.ValidateHandler = v
}
