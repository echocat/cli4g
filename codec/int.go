package codec

import (
	"context"
	"reflect"
	"strconv"
)

type signedInteger interface {
	int | int8 | int16 | int32 | int64
}

type signed[T signedInteger] struct{ *numericConfig }

func (this signed[T]) ToRaw(ctx context.Context) Codec[any, any] { return Raw(ctx, this) }

var (
	intConfig   = numericConfig{reflect.TypeFor[int](), strconv.IntSize}
	int8Config  = numericConfig{reflect.TypeFor[int8](), 8}
	int16Config = numericConfig{reflect.TypeFor[int16](), 16}
	int32Config = numericConfig{reflect.TypeFor[int32](), 32}
	int64Config = numericConfig{reflect.TypeFor[int64](), 64}
)

func (this signed[T]) CanHandleType(ctx context.Context, t reflect.Type) (bool, error) {
	return t == this.typ, ctx.Err()
}

func (signed[T]) ExpectMinMax(context.Context, Usage) (int, int) { return 1, 1 }

func (this signed[T]) ParseRequirement(ctx context.Context, usage Usage, args ...string) (T, error) {
	if err := EnsureArgumentsMatching(ctx, usage, this, args...); err != nil {
		return 0, err
	}
	n, err := strconv.ParseInt(args[0], 0, int(this.bits))
	if err != nil {
		return 0, signedParseError(args[0], int(this.bits), err)
	}
	return T(n), nil
}

func (signed[T]) FormatRequirement(_ context.Context, value T) ([]string, error) {
	return []string{strconv.FormatInt(int64(value), 10)}, nil
}

func (signed[T]) CloneRequirement(ctx context.Context, value T) (T, error) {
	return value, ctx.Err()
}
func (signed[T]) Resolve(ctx context.Context, _ Scope, value T) (T, error) { return value, ctx.Err() }

func NewInt() Codec[int, int] {
	return signed[int]{&intConfig}
}
func NewInt8() Codec[int8, int8] {
	return signed[int8]{&int8Config}
}
func NewInt16() Codec[int16, int16] {
	return signed[int16]{&int16Config}
}
func NewInt32() Codec[int32, int32] {
	return signed[int32]{&int32Config}
}
func NewInt64() Codec[int64, int64] {
	return signed[int64]{&int64Config}
}
