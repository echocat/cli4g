package codec

import (
	"context"
	"encoding"
	"fmt"
	"reflect"
)

// NewMarshaller adapts text methods on T/*T using string inputs and defaults.
// UnmarshalText runs during Resolve on a fresh value. It must mark user mistakes
// with [NewValidationError]; unmarked errors receive a neutral message with their
// original cause retained. Warning: marked messages are displayed verbatim and
// must not contain input values, passwords, or other confidential data.
func NewMarshaller[T encoding.TextMarshaler, TP interface {
	*T
	encoding.TextUnmarshaler
}]() Codec[T, string] {
	return textCodec[T]{typ: reflect.TypeFor[T](), resolve: func(ctx context.Context, scope Scope, input string) (T, error) {
		var target TP = new(T)
		ownValue(scope, target)
		if err := ctx.Err(); err != nil {
			return *target, err
		}
		err := target.UnmarshalText([]byte(input))
		if err != nil {
			return *target, err
		}
		return *target, ctx.Err()
	}}
}

// NewMarshallerP adapts fresh pointer targets from a concurrent-safe factory.
// A nil factory panics; a nil result fails with [ErrInvalid]. See [NewMarshaller].
// Warning: custom ValidationError messages are displayed verbatim; their authors
// must exclude supplied inputs and confidential data.
func NewMarshallerP[T interface {
	encoding.TextMarshaler
	encoding.TextUnmarshaler
}](factory func() T) Codec[T, string] {
	if factory == nil {
		panic("pointer value codec requires a factory")
	}
	return textCodec[T]{typ: reflect.TypeFor[T](), resolve: func(ctx context.Context, scope Scope, input string) (T, error) {
		target := factory()
		if !isNil(target) {
			ownValue(scope, target)
		}
		if err := ctx.Err(); err != nil {
			return target, err
		}
		if isNil(target) {
			var zero T
			return zero, fmt.Errorf("%w: value factory returned nil", ErrInvalid)
		}
		if err := target.UnmarshalText([]byte(input)); err != nil {
			return target, err
		}
		return target, ctx.Err()
	}}
}

type textCodec[T any] struct {
	typ     reflect.Type
	resolve func(context.Context, Scope, string) (T, error)
}

func (this textCodec[T]) ToRaw(ctx context.Context) Codec[any, any] { return Raw(ctx, this) }

func (this textCodec[T]) CanHandleType(ctx context.Context, typ reflect.Type) (bool, error) {
	return typ == this.typ, ctx.Err()
}

func (textCodec[T]) ExpectMinMax(context.Context, Usage) (int, int) { return 1, 1 }

func (this textCodec[T]) ParseRequirement(ctx context.Context, usage Usage, args ...string) (string, error) {
	if err := EnsureArgumentsMatching(ctx, usage, this, args...); err != nil {
		return "", err
	}
	return args[0], nil
}

func (textCodec[T]) FormatRequirement(ctx context.Context, input string) ([]string, error) {
	return []string{input}, ctx.Err()
}

func (textCodec[T]) CloneRequirement(ctx context.Context, input string) (string, error) {
	return input, ctx.Err()
}

func (this textCodec[T]) Resolve(ctx context.Context, scope Scope, input string) (T, error) {
	if isNil(scope) {
		var zero T
		return zero, fmt.Errorf("%w: text resolution requires a scope", ErrInvalid)
	}
	if err := ctx.Err(); err != nil {
		var zero T
		return zero, err
	}
	value, err := this.resolve(ctx, scope, input)
	return value, SanitizeError(err, "value could not be decoded as text")
}
