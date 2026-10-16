package cli

import (
	"context"
	"reflect"
	"slices"

	"github.com/echocat/cli4g/codec"
)

// Flag describes a CLI flag. Metadata and defaults remain separate from parsed
// values; use [Invocation.Flag] to read V and [Flag.Default] to read a default R.
type Flag[V, R any] struct {
	names        []string
	profiles     []string
	shorts       []rune
	envVars      []string
	description  string
	required     bool
	visible      bool
	placeholder  string
	codec        codec.Codec[V, R]
	valueType    reflect.Type
	fieldPath    []string
	binding      *fieldBinding
	arity        valueArity
	defaultValue *valueDefault[R]
	isHelp       bool // framework-owned flag, never set by user requirements
	cumulative   bool
}

// Profiles returns a copy of the explicit profile assignments; empty means always active.
func (this *Flag[V, R]) Profiles() []string { return slices.Clone(this.profiles) }

func (this *Flag[V, R]) Name() string {
	return this.names[0]
}

func (this *Flag[V, R]) Names() []string {
	return slices.Clone(this.names)
}

func (this *Flag[V, R]) Shorts() []rune {
	return slices.Clone(this.shorts)
}

func (this *Flag[V, R]) EnvVars() []string { return slices.Clone(this.envVars) }

func (this *Flag[V, R]) Description() string {
	return this.description
}

func (this *Flag[V, R]) IsRequired() bool {
	return this.required
}

func (this *Flag[V, R]) IsCumulative() bool { return this.cumulative }

func (this *Flag[V, R]) IsBoolFlag() bool {
	if this.arity.min != 0 || this.arity.max != 1 {
		return false
	}
	if this.valueType != nil && this.valueType.Kind() == reflect.Bool {
		return true
	}
	value, ok := this.codec.(interface{ IsBoolFlag() bool })
	return ok && value.IsBoolFlag()
}

func (this *Flag[V, R]) IsVisible() bool {
	return this.visible
}

func (this *Flag[V, R]) Placeholder() string {
	return this.placeholder
}

func (this *Flag[V, R]) ValueType() reflect.Type { return this.valueType }

func (this *Flag[V, R]) Codec() codec.Codec[V, R] {
	return this.codec
}

func (this *Flag[V, R]) HasDefault() bool { return this.defaultValue != nil }

// Default returns an independent default R, or found=false if none is declared.
// It does not acquire runtime resources.
func (this *Flag[V, R]) Default(ctx context.Context) (value R, found bool, err error) {
	return resolveDefault(ctx, this.defaultValue, this.codec, this.valueType)
}

// FormatDefault returns help-safe default text without opening resources.
// Invalid declared defaults return an error, even if CLI input could override them.
func (this *Flag[V, R]) FormatDefault(ctx context.Context) ([]string, bool, error) {
	return formatDefault(ctx, this.defaultValue, this.codec, this.valueType)
}

type flagEntry[V any] struct {
	flag        *Flag[V, V]
	requirement V
	value       V
	resolved    bool
}
