package cli

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	"github.com/echocat/cli4g/codec"
)

// ArgumentRequirementLike allows differently typed arguments in one builder.
// It is implemented by [ArgumentRequirement], not by callers.
type ArgumentRequirementLike interface {
	createRaw() (*Argument[any, any], error)
}

// NewArgument starts a positional argument using codec. The first optional string
// sets the name; further strings form the description, joined with spaces. With
// no strings, configure the name later; Create validates the completed builder.
func NewArgument[V, R any](codec codec.Codec[V, R], nameAndDescription ...string) *ArgumentRequirement[V, R] {
	name, description := parseNameAndDescription(nameAndDescription)
	return &ArgumentRequirement[V, R]{Codec: codec, Name: name, Description: description}
}

// ArgumentRequirement configures a positional value V with input requirements R.
// Slice arguments must be last. Required arguments cannot have defaults or
// belong to a container with subcommands. Bool arguments require explicit input.
// See [ArgumentRequirement.WithField] and [ArgumentRequirement.WithDefault].
// Explicit arguments retain registration order; field bindings replacing tags
// keep their tagged slots in mixed definitions. Other builder arguments come first.
type ArgumentRequirement[V, R any] struct {
	Name string
	// Profiles activates this definition when any listed profile is selected.
	// An empty list makes it active for every parse.
	Profiles    []string
	Description string
	Required    bool
	Placeholder string
	Codec       codec.Codec[V, R]
	// CodecFactory supplies a codec at Create when Codec is nil. It configures
	// schema only; it must not acquire invocation resources.
	CodecFactory func() (codec.Codec[V, R], error)
	Field        []string
	defaultValue *valueDefault[R]
}

func (this ArgumentRequirement[V, R]) createRaw() (*Argument[any, any], error) {
	selected := this.Codec
	if selected == nil && this.CodecFactory != nil {
		var err error
		selected, err = this.CodecFactory()
		if err != nil {
			return nil, fmt.Errorf("argument %q: create codec: %w", this.Name, err)
		}
	}
	snapshot, err := codec.Snapshot(context.Background(), selected)
	if err != nil {
		return nil, fmt.Errorf("argument %q: %w", this.Name, err)
	}
	return &Argument[any, any]{
		name:         this.Name,
		profiles:     slices.Clone(this.Profiles),
		description:  this.Description,
		required:     this.Required,
		placeholder:  this.Placeholder,
		codec:        snapshot,
		valueType:    reflect.TypeFor[V](),
		fieldPath:    slices.Clone(this.Field),
		defaultValue: rawDefault(this.defaultValue),
	}, nil
}

// WithProfiles replaces the profile selection. No profiles means always active.
func (this *ArgumentRequirement[V, R]) WithProfiles(v ...string) *ArgumentRequirement[V, R] {
	this.SetProfiles(v)
	return this
}

func (this *ArgumentRequirement[V, R]) SetProfiles(v []string) {
	this.Profiles = slices.Clone(v)
}

// WithDefault sets a resource-free typed default R for an omitted optional argument.
// R may differ from V; each parse receives an independent copy.
func (this *ArgumentRequirement[V, R]) WithDefault(v R) *ArgumentRequirement[V, R] {
	this.SetDefault(v)
	return this
}

func (this *ArgumentRequirement[V, R]) SetDefault(v R) {
	this.defaultValue = &valueDefault[R]{value: v}
}

// WithDefaultStr sets a complete codec input without shell or CSV splitting.
// Either default setter replaces the previously configured default.
func (this *ArgumentRequirement[V, R]) WithDefaultStr(v ...string) *ArgumentRequirement[V, R] {
	this.SetDefaultStr(v...)
	return this
}

func (this *ArgumentRequirement[V, R]) SetDefaultStr(v ...string) {
	this.defaultValue = &valueDefault[R]{strings: slices.Clone(v), isString: true}
}

// WithField binds V to an exported options field; v is its nested field path.
// Explicit binding overrides that field's tags; an input name alone does not bind.
func (this *ArgumentRequirement[V, R]) WithField(v ...string) *ArgumentRequirement[V, R] {
	this.SetField(v)
	return this
}

func (this *ArgumentRequirement[V, R]) SetField(v []string) *ArgumentRequirement[V, R] {
	this.Field = v
	return this
}

func (this *ArgumentRequirement[V, R]) WithName(v string) *ArgumentRequirement[V, R] {
	this.SetName(v)
	return this
}

func (this *ArgumentRequirement[V, R]) SetName(v string) {
	this.Name = v
}

func (this *ArgumentRequirement[V, R]) WithDescription(v string) *ArgumentRequirement[V, R] {
	this.SetDescription(v)
	return this
}

func (this *ArgumentRequirement[V, R]) SetDescription(v string) {
	this.Description = v
}

// WithRequired demands a positional value. Defaults and factory values cannot
// satisfy it; containers with subcommands cannot declare required arguments.
func (this *ArgumentRequirement[V, R]) WithRequired(v bool) *ArgumentRequirement[V, R] {
	this.SetRequired(v)
	return this
}

func (this *ArgumentRequirement[V, R]) SetRequired(v bool) {
	this.Required = v
}

// WithPlaceholder sets help value text, preserving its case. The default uses the
// complete argument name. Help encloses the text in <...> unless already enclosed;
// optionality and arity are shown outside those brackets.
func (this *ArgumentRequirement[V, R]) WithPlaceholder(v string) *ArgumentRequirement[V, R] {
	this.SetPlaceholder(v)
	return this
}

func (this *ArgumentRequirement[V, R]) SetPlaceholder(v string) {
	this.Placeholder = v
}

// WithCodec replaces the decoder and clears [ArgumentRequirement.WithCodecFactory].
func (this *ArgumentRequirement[V, R]) WithCodec(v codec.Codec[V, R]) *ArgumentRequirement[V, R] {
	this.SetCodec(v)
	return this
}

func (this *ArgumentRequirement[V, R]) SetCodec(v codec.Codec[V, R]) {
	this.Codec = v
	this.CodecFactory = nil
}

// WithCodecFactory configures the decoder during Create and replaces [ArgumentRequirement.WithCodec].
// The factory must not acquire runtime resources; configuration errors fail Create.
func (this *ArgumentRequirement[V, R]) WithCodecFactory(v func() (codec.Codec[V, R], error)) *ArgumentRequirement[V, R] {
	this.SetCodecFactory(v)
	return this
}

func (this *ArgumentRequirement[V, R]) SetCodecFactory(v func() (codec.Codec[V, R], error)) {
	this.Codec = nil
	this.CodecFactory = v
}
