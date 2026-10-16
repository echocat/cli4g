package codec

import (
	"context"
	"reflect"
	"strconv"
)

type complexNumber interface{ complex64 | complex128 }

type complexValue[T complexNumber] struct{ *numericConfig }

func (this complexValue[T]) ToRaw(ctx context.Context) Codec[any, any] { return Raw(ctx, this) }

var (
	complex64Config  = numericConfig{reflect.TypeFor[complex64](), 64}
	complex128Config = numericConfig{reflect.TypeFor[complex128](), 128}
)

func (this complexValue[T]) CanHandleType(ctx context.Context, t reflect.Type) (bool, error) {
	return t == this.typ, ctx.Err()
}

func (complexValue[T]) ExpectMinMax(context.Context, Usage) (int, int) { return 1, 1 }

func (this complexValue[T]) ParseRequirement(ctx context.Context, usage Usage, args ...string) (T, error) {
	if err := EnsureArgumentsMatching(ctx, usage, this, args...); err != nil {
		return 0, err
	}
	n, err := strconv.ParseComplex(args[0], int(this.bits))
	if err != nil {
		return 0, complexParseError(int(this.bits), err)
	}
	return T(n), nil
}

func (this complexValue[T]) FormatRequirement(_ context.Context, value T) ([]string, error) {
	return []string{strconv.FormatComplex(complex128(value), 'g', -1, int(this.bits))}, nil
}

func (complexValue[T]) CloneRequirement(ctx context.Context, value T) (T, error) {
	return value, ctx.Err()
}
func (complexValue[T]) Resolve(ctx context.Context, _ Scope, value T) (T, error) {
	return value, ctx.Err()
}

func NewComplex64() Codec[complex64, complex64] {
	return complexValue[complex64]{&complex64Config}
}
func NewComplex128() Codec[complex128, complex128] {
	return complexValue[complex128]{&complex128Config}
}
