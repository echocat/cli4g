package codec

import (
	"context"
	"reflect"
	"strconv"
)

type unsignedInteger interface {
	uint | uint8 | uint16 | uint32 | uint64 | uintptr
}

type unsigned[T unsignedInteger] struct{ *numericConfig }

func (this unsigned[T]) ToRaw(ctx context.Context) Codec[any, any] { return Raw(ctx, this) }

var (
	uintConfig    = numericConfig{reflect.TypeFor[uint](), strconv.IntSize}
	uint8Config   = numericConfig{reflect.TypeFor[uint8](), 8}
	uint16Config  = numericConfig{reflect.TypeFor[uint16](), 16}
	uint32Config  = numericConfig{reflect.TypeFor[uint32](), 32}
	uint64Config  = numericConfig{reflect.TypeFor[uint64](), 64}
	uintptrConfig = numericConfig{reflect.TypeFor[uintptr](), strconv.IntSize}
)

func (this unsigned[T]) CanHandleType(ctx context.Context, t reflect.Type) (bool, error) {
	return t == this.typ, ctx.Err()
}

func (unsigned[T]) ExpectMinMax(context.Context, Usage) (int, int) { return 1, 1 }

func (this unsigned[T]) ParseRequirement(ctx context.Context, usage Usage, args ...string) (T, error) {
	if err := EnsureArgumentsMatching(ctx, usage, this, args...); err != nil {
		return 0, err
	}
	n, err := strconv.ParseUint(args[0], 0, int(this.bits))
	if err != nil {
		return 0, unsignedParseError(int(this.bits), err)
	}
	return T(n), nil
}

func (unsigned[T]) FormatRequirement(_ context.Context, value T) ([]string, error) {
	return []string{strconv.FormatUint(uint64(value), 10)}, nil
}

func (unsigned[T]) CloneRequirement(ctx context.Context, value T) (T, error) {
	return value, ctx.Err()
}
func (unsigned[T]) Resolve(ctx context.Context, _ Scope, value T) (T, error) { return value, ctx.Err() }

func NewUint() Codec[uint, uint] {
	return unsigned[uint]{&uintConfig}
}
func NewUint8() Codec[uint8, uint8] {
	return unsigned[uint8]{&uint8Config}
}
func NewUint16() Codec[uint16, uint16] {
	return unsigned[uint16]{&uint16Config}
}
func NewUint32() Codec[uint32, uint32] {
	return unsigned[uint32]{&uint32Config}
}
func NewUint64() Codec[uint64, uint64] {
	return unsigned[uint64]{&uint64Config}
}
func NewUintptr() Codec[uintptr, uintptr] {
	return unsigned[uintptr]{&uintptrConfig}
}
