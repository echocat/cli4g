package cli

import (
	"context"
	"fmt"
	"reflect"
	"slices"

	"github.com/echocat/cli4g/codec"
)

// FlagRequirementLike allows flags with different value types in one builder.
// It is implemented by [FlagRequirement], not by callers.
type FlagRequirementLike interface {
	createRaw() (*Flag[any, any], error)
}

// NewFlag starts a visible flag using codec. The first optional string sets the
// name; further strings form the description, joined with spaces. Add aliases
// with [FlagRequirement.WithAliases]. With no strings, configure the name later;
// Create validates the completed builder.
func NewFlag[V, R any](codec codec.Codec[V, R], nameAndDescription ...string) *FlagRequirement[V, R] {
	// Unlike a zero-valued requirement, constructed flags are visible by default.
	name, description := parseNameAndDescription(nameAndDescription)
	result := &FlagRequirement[V, R]{Codec: codec, Visible: true, Description: description}
	if len(nameAndDescription) > 0 {
		result.Names = []string{name}
	}
	return result
}

// FlagRequirement configures a flag producing V from input requirements R.
// Use [FlagRequirement.WithDefault] for typed R or [FlagRequirement.WithDefaultStr]
// for input tokens. Required flags cannot declare defaults, including zero/nil.
// Repetition requires [codec.Cumulative]; standard slice flags accept one element
// per occurrence. Bind fields explicitly with [FlagRequirement.WithField].
type FlagRequirement[V, R any] struct {
	// Names contains the primary name followed by aliases.
	Names []string
	// Profiles activates this definition when any listed profile is selected.
	// An empty list makes it active for every parse.
	Profiles []string
	// Shorts contains optional single-rune aliases.
	Shorts []rune

	// EnvVars contains environment fallback names in lookup order.
	EnvVars []string

	Description string
	Required    bool
	Visible     bool
	Placeholder string
	Codec       codec.Codec[V, R]
	// CompletionHandler optionally overrides value suggestions for this flag.
	// Returning Handled=false delegates to the codec and automatic completion.
	CompletionHandler CompletionHandler
	// CodecFactory configures a codec during Create, without acquiring resources.
	CodecFactory func() (codec.Codec[V, R], error)
	Field        []string
	defaultValue *valueDefault[R]
}

func (this FlagRequirement[V, R]) createRaw() (*Flag[any, any], error) {
	selected := this.Codec
	if selected == nil && this.CodecFactory != nil {
		var err error
		selected, err = this.CodecFactory()
		if err != nil {
			return nil, fmt.Errorf("flag %q: create codec: %w", this.Names, err)
		}
	}
	snapshot, err := codec.Snapshot(context.Background(), selected)
	if err != nil {
		return nil, fmt.Errorf("flag %q: %w", this.Names, err)
	}
	return &Flag[any, any]{
		names:             slices.Clone(this.Names),
		profiles:          slices.Clone(this.Profiles),
		shorts:            slices.Clone(this.Shorts),
		envVars:           slices.Clone(this.EnvVars),
		description:       this.Description,
		required:          this.Required,
		visible:           this.Visible,
		placeholder:       this.Placeholder,
		codec:             snapshot,
		completionHandler: this.CompletionHandler,
		valueType:         reflect.TypeFor[V](),
		fieldPath:         slices.Clone(this.Field),
		defaultValue:      rawDefault(this.defaultValue),
	}, nil
}

// WithProfiles replaces the profile selection. No profiles means always active.
func (this *FlagRequirement[V, R]) WithProfiles(v ...string) *FlagRequirement[V, R] {
	this.SetProfiles(v)
	return this
}

func (this *FlagRequirement[V, R]) SetProfiles(v []string) {
	this.Profiles = slices.Clone(v)
}

// WithDefault sets a resource-free typed default R, independently copied for each parse.
// R can differ from V, for example [codec.FileRequirement] instead of *os.File.
func (this *FlagRequirement[V, R]) WithDefault(v R) *FlagRequirement[V, R] {
	this.SetDefault(v)
	return this
}

func (this *FlagRequirement[V, R]) SetDefault(v R) {
	this.defaultValue = &valueDefault[R]{value: v}
}

// WithDefaultStr sets default input tokens without shell/CSV splitting.
// It replaces any previous typed or string default.
func (this *FlagRequirement[V, R]) WithDefaultStr(v ...string) *FlagRequirement[V, R] {
	this.SetDefaultStr(v...)
	return this
}

func (this *FlagRequirement[V, R]) SetDefaultStr(v ...string) {
	this.defaultValue = &valueDefault[R]{strings: slices.Clone(v), isString: true}
}

// WithField binds V to an exported options field; v is its nested field path.
// The binding overrides tags on that field. Omit it to keep the flag unbound.
func (this *FlagRequirement[V, R]) WithField(v ...string) *FlagRequirement[V, R] {
	this.SetField(v)
	return this
}

func (this *FlagRequirement[V, R]) SetField(v []string) *FlagRequirement[V, R] {
	this.Field = v
	return this
}

// WithName replaces the primary name and optional aliases. WithAliases appends aliases.
func (this *FlagRequirement[V, R]) WithName(v ...string) *FlagRequirement[V, R] {
	this.SetNames(v)
	return this
}

func (this *FlagRequirement[V, R]) WithAliases(v ...string) *FlagRequirement[V, R] {
	this.Names = append(this.Names, v...)
	return this
}

func (this *FlagRequirement[V, R]) SetNames(v []string) {
	this.Names = v
}

func (this *FlagRequirement[V, R]) WithShort(v ...rune) *FlagRequirement[V, R] {
	this.Shorts = append(this.Shorts, v...)
	return this
}

func (this *FlagRequirement[V, R]) SetShorts(v []rune) {
	this.Shorts = v
}

// WithEnvVar adds environment fallback names. CLI input takes precedence.
func (this *FlagRequirement[V, R]) WithEnvVar(v ...string) *FlagRequirement[V, R] {
	this.EnvVars = append(this.EnvVars, v...)
	return this
}

func (this *FlagRequirement[V, R]) SetEnvVars(v []string) {
	this.EnvVars = v
}

func (this *FlagRequirement[V, R]) WithDescription(v string) *FlagRequirement[V, R] {
	this.SetDescription(v)
	return this
}

func (this *FlagRequirement[V, R]) SetDescription(v string) {
	this.Description = v
}

// WithRequired demands CLI or environment input. Defaults are incompatible;
// options-factory values do not satisfy this requirement.
func (this *FlagRequirement[V, R]) WithRequired(v bool) *FlagRequirement[V, R] {
	this.SetRequired(v)
	return this
}

func (this *FlagRequirement[V, R]) SetRequired(v bool) {
	this.Required = v
}

func (this *FlagRequirement[V, R]) WithVisible(v bool) *FlagRequirement[V, R] {
	this.SetVisible(v)
	return this
}

func (this *FlagRequirement[V, R]) SetVisible(v bool) {
	this.Visible = v
}

// WithPlaceholder sets help value text, preserving its case. The default uses the
// last word of the primary flag name, split at '.', '-', '_' and case boundaries,
// and lowercased. Help displays --flag=<text>, enclosing text once in <...>.
func (this *FlagRequirement[V, R]) WithPlaceholder(v string) *FlagRequirement[V, R] {
	this.SetPlaceholder(v)
	return this
}

func (this *FlagRequirement[V, R]) SetPlaceholder(v string) {
	this.Placeholder = v
}

// WithCompletionHandler sets a context-aware value handler for this flag without
// changing its parsing codec or CLI syntax. It runs before the codec completer;
// Handled=false delegates, and Handled=true with no candidates suppresses further
// suggestions. A nil handler restores codec/automatic completion. Name completion
// never invokes it. Create captures the callback; it must be concurrent-safe.
func (this *FlagRequirement[V, R]) WithCompletionHandler(v CompletionHandler) *FlagRequirement[V, R] {
	this.SetCompletionHandler(v)
	return this
}

// SetCompletionHandler is the setter counterpart of WithCompletionHandler.
func (this *FlagRequirement[V, R]) SetCompletionHandler(v CompletionHandler) {
	this.CompletionHandler = v
}

// WithCodec replaces the decoder and clears any [FlagRequirement.WithCodecFactory].
func (this *FlagRequirement[V, R]) WithCodec(v codec.Codec[V, R]) *FlagRequirement[V, R] {
	this.SetCodec(v)
	return this
}

func (this *FlagRequirement[V, R]) SetCodec(v codec.Codec[V, R]) {
	this.Codec = v
	this.CodecFactory = nil
}

// WithCodecFactory configures the decoder during Create and replaces [FlagRequirement.WithCodec].
// The factory must not acquire runtime resources; configuration errors fail Create.
func (this *FlagRequirement[V, R]) WithCodecFactory(v func() (codec.Codec[V, R], error)) *FlagRequirement[V, R] {
	this.SetCodecFactory(v)
	return this
}

func (this *FlagRequirement[V, R]) SetCodecFactory(v func() (codec.Codec[V, R], error)) {
	this.Codec = nil
	this.CodecFactory = v
}
