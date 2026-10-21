package cli

import (
	"context"
	"reflect"
	"slices"

	"github.com/echocat/cli4g/codec"
)

// Argument describes a positional input. Use [Invocation.Argument] for parsed V
// and [Argument.Default] for its resource-free default R.
type Argument[V, R any] struct {
	name              string
	profiles          []string
	description       string
	required          bool
	placeholder       string
	codec             codec.Codec[V, R]
	completionHandler CompletionHandler
	valueType         reflect.Type
	fieldPath         []string
	binding           *fieldBinding
	arity             valueArity
	defaultValue      *valueDefault[R]
	index             int // position in the compiled container's argument definitions
}

// Profiles returns a copy of the explicit profile assignments; empty means always active.
func (this *Argument[V, R]) Profiles() []string { return slices.Clone(this.profiles) }

func (this *Argument[V, R]) Name() string {
	return this.name
}

func (this *Argument[V, R]) Description() string {
	return this.description
}

func (this *Argument[V, R]) IsRequired() bool {
	return this.required
}

func (this *Argument[V, R]) Placeholder() string {
	return this.placeholder
}

func (this *Argument[V, R]) ValueType() reflect.Type { return this.valueType }

func (this *Argument[V, R]) Codec() codec.Codec[V, R] {
	return this.codec
}

// CompletionHandler returns the explicitly configured argument handler, or nil
// when value completion uses only the codec and standard handler. Codec returns
// the effective codec, including the captured argument override.
func (this *Argument[V, R]) CompletionHandler() CompletionHandler { return this.completionHandler }

func (this *Argument[V, R]) HasDefault() bool { return this.defaultValue != nil }

// Default returns an independent resource-free requirement R, never a runtime V.
func (this *Argument[V, R]) Default(ctx context.Context) (value R, found bool, err error) {
	return resolveDefault(ctx, this.defaultValue, this.codec, this.valueType)
}

// FormatDefault returns help-safe default text without resolving resources.
func (this *Argument[V, R]) FormatDefault(ctx context.Context) ([]string, bool, error) {
	return formatDefault(ctx, this.defaultValue, this.codec, this.valueType)
}

type argumentEntry[V any] struct {
	argument    *Argument[V, V]
	requirement V
	value       V
	resolved    bool
}
