package codec

import (
	"context"
	"reflect"
	"strconv"
)

type floatingPoint interface{ float32 | float64 }

type floating[T floatingPoint] struct{ *numericConfig }

func (this floating[T]) ToRaw(ctx context.Context) Codec[any, any] { return Raw(ctx, this) }

var (
	float32Config = numericConfig{reflect.TypeFor[float32](), 32}
	float64Config = numericConfig{reflect.TypeFor[float64](), 64}
)

func (this floating[T]) CanHandleType(ctx context.Context, t reflect.Type) (bool, error) {
	return t == this.typ, ctx.Err()
}

func (floating[T]) ExpectMinMax(context.Context, Usage) (int, int) { return 1, 1 }

func (this floating[T]) ParseRequirement(ctx context.Context, usage Usage, args ...string) (T, error) {
	if err := EnsureArgumentsMatching(ctx, usage, this, args...); err != nil {
		return 0, err
	}
	n, err := strconv.ParseFloat(args[0], int(this.bits))
	if err != nil {
		return 0, floatParseError(args[0], int(this.bits), err)
	}
	return T(n), nil
}

func (this floating[T]) FormatRequirement(_ context.Context, value T) ([]string, error) {
	return []string{strconv.FormatFloat(float64(value), 'g', -1, int(this.bits))}, nil
}

func (floating[T]) CloneRequirement(ctx context.Context, value T) (T, error) {
	return value, ctx.Err()
}
func (floating[T]) Resolve(ctx context.Context, _ Scope, value T) (T, error) { return value, ctx.Err() }

func NewFloat32() Codec[float32, float32] {
	return floating[float32]{&float32Config}
}
func NewFloat64() Codec[float64, float64] {
	return floating[float64]{&float64Config}
}
