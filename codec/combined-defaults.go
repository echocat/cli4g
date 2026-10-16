package codec

import (
	"context"
	"os"
	"reflect"
	"time"
)

// CombinedDefaults resolves standard concrete value types at compilation time.
// It has no mutable type selection and can be shared by concurrent compilations.
// BindType returns the existing typed codec's raw adapter; runtime parsing does
// not dispatch through this resolver. Defined types are not matched by Kind, so
// custom codecs and automatic Value/text adapters retain their semantics.
type CombinedDefaults struct{}

// NewCombinedDefaults creates a standard type resolver. Use it through Registry
// or BindType before accessing parsing, formatting, resolution, or arity methods.
func NewCombinedDefaults() Codec[any, any] { return CombinedDefaults{} }

func (c CombinedDefaults) BindType(ctx context.Context, typ reflect.Type) (Codec[any, any], bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	var selected Codec[any, any]
	switch typ {
	case reflect.TypeFor[bool]():
		selected = NewBool().ToRaw(ctx)
	case reflect.TypeFor[string]():
		selected = NewString().ToRaw(ctx)
	case reflect.TypeFor[time.Duration]():
		selected = NewDuration().ToRaw(ctx)
	case reflect.TypeFor[time.Time]():
		selected = NewDateTime().ToRaw(ctx)
	case reflect.TypeFor[int]():
		selected = NewInt().ToRaw(ctx)
	case reflect.TypeFor[int8]():
		selected = NewInt8().ToRaw(ctx)
	case reflect.TypeFor[int16]():
		selected = NewInt16().ToRaw(ctx)
	case reflect.TypeFor[int32]():
		selected = NewInt32().ToRaw(ctx)
	case reflect.TypeFor[int64]():
		selected = NewInt64().ToRaw(ctx)
	case reflect.TypeFor[uint]():
		selected = NewUint().ToRaw(ctx)
	case reflect.TypeFor[uint8]():
		selected = NewUint8().ToRaw(ctx)
	case reflect.TypeFor[uint16]():
		selected = NewUint16().ToRaw(ctx)
	case reflect.TypeFor[uint32]():
		selected = NewUint32().ToRaw(ctx)
	case reflect.TypeFor[uint64]():
		selected = NewUint64().ToRaw(ctx)
	case reflect.TypeFor[uintptr]():
		selected = NewUintptr().ToRaw(ctx)
	case reflect.TypeFor[float32]():
		selected = NewFloat32().ToRaw(ctx)
	case reflect.TypeFor[float64]():
		selected = NewFloat64().ToRaw(ctx)
	case reflect.TypeFor[complex64]():
		selected = NewComplex64().ToRaw(ctx)
	case reflect.TypeFor[complex128]():
		selected = NewComplex128().ToRaw(ctx)
	case reflect.TypeFor[[]bool]():
		selected = NewBoolSlice().ToRaw(ctx)
	case reflect.TypeFor[[]string]():
		selected = NewStringSlice().ToRaw(ctx)
	case reflect.TypeFor[[]time.Duration]():
		selected = NewDurationSlice().ToRaw(ctx)
	case reflect.TypeFor[[]time.Time]():
		selected = NewDateTimeSlice().ToRaw(ctx)
	case reflect.TypeFor[[]int]():
		selected = NewIntSlice().ToRaw(ctx)
	case reflect.TypeFor[[]int8]():
		selected = NewInt8Slice().ToRaw(ctx)
	case reflect.TypeFor[[]int16]():
		selected = NewInt16Slice().ToRaw(ctx)
	case reflect.TypeFor[[]int32]():
		selected = NewInt32Slice().ToRaw(ctx)
	case reflect.TypeFor[[]int64]():
		selected = NewInt64Slice().ToRaw(ctx)
	case reflect.TypeFor[[]uint]():
		selected = NewUintSlice().ToRaw(ctx)
	case reflect.TypeFor[[]uint8]():
		selected = NewUint8Slice().ToRaw(ctx)
	case reflect.TypeFor[[]uint16]():
		selected = NewUint16Slice().ToRaw(ctx)
	case reflect.TypeFor[[]uint32]():
		selected = NewUint32Slice().ToRaw(ctx)
	case reflect.TypeFor[[]uint64]():
		selected = NewUint64Slice().ToRaw(ctx)
	case reflect.TypeFor[[]uintptr]():
		selected = NewUintptrSlice().ToRaw(ctx)
	case reflect.TypeFor[[]float32]():
		selected = NewFloat32Slice().ToRaw(ctx)
	case reflect.TypeFor[[]float64]():
		selected = NewFloat64Slice().ToRaw(ctx)
	case reflect.TypeFor[[]complex64]():
		selected = NewComplex64Slice().ToRaw(ctx)
	case reflect.TypeFor[[]complex128]():
		selected = NewComplex128Slice().ToRaw(ctx)
	case reflect.TypeFor[map[string]bool]():
		selected = NewBoolMap().ToRaw(ctx)
	case reflect.TypeFor[map[string]string]():
		selected = NewStringMap().ToRaw(ctx)
	case reflect.TypeFor[map[string]time.Duration]():
		selected = NewDurationMap().ToRaw(ctx)
	case reflect.TypeFor[map[string]time.Time]():
		selected = NewDateTimeMap().ToRaw(ctx)
	case reflect.TypeFor[map[string]int]():
		selected = NewIntMap().ToRaw(ctx)
	case reflect.TypeFor[map[string]int8]():
		selected = NewInt8Map().ToRaw(ctx)
	case reflect.TypeFor[map[string]int16]():
		selected = NewInt16Map().ToRaw(ctx)
	case reflect.TypeFor[map[string]int32]():
		selected = NewInt32Map().ToRaw(ctx)
	case reflect.TypeFor[map[string]int64]():
		selected = NewInt64Map().ToRaw(ctx)
	case reflect.TypeFor[map[string]uint]():
		selected = NewUintMap().ToRaw(ctx)
	case reflect.TypeFor[map[string]uint8]():
		selected = NewUint8Map().ToRaw(ctx)
	case reflect.TypeFor[map[string]uint16]():
		selected = NewUint16Map().ToRaw(ctx)
	case reflect.TypeFor[map[string]uint32]():
		selected = NewUint32Map().ToRaw(ctx)
	case reflect.TypeFor[map[string]uint64]():
		selected = NewUint64Map().ToRaw(ctx)
	case reflect.TypeFor[map[string]uintptr]():
		selected = NewUintptrMap().ToRaw(ctx)
	case reflect.TypeFor[map[string]float32]():
		selected = NewFloat32Map().ToRaw(ctx)
	case reflect.TypeFor[map[string]float64]():
		selected = NewFloat64Map().ToRaw(ctx)
	case reflect.TypeFor[map[string]complex64]():
		selected = NewComplex64Map().ToRaw(ctx)
	case reflect.TypeFor[map[string]complex128]():
		selected = NewComplex128Map().ToRaw(ctx)
	case reflect.TypeFor[*os.File]():
		selected = NewFile().ToRaw(ctx)
	case reflect.TypeFor[[]*os.File]():
		selected = NewSlice(NewFile()).ToRaw(ctx)
	}
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	return selected, selected != nil, nil
}

func (c CombinedDefaults) CanHandleType(ctx context.Context, typ reflect.Type) (bool, error) {
	_, matched, err := c.BindType(ctx, typ)
	return matched, err
}

func (c CombinedDefaults) ToRaw(ctx context.Context) Codec[any, any] {
	if ctx.Err() != nil {
		return nil
	}
	return c
}

// ExpectMinMax returns invalid metadata until a concrete type is bound.
func (CombinedDefaults) ExpectMinMax(context.Context, Usage) (int, int) { return -1, -1 }

func unboundDefaultsError(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return NewMessageError("default codec must be bound to a value type", ErrInvalid)
}

func (CombinedDefaults) ParseRequirement(ctx context.Context, _ Usage, _ ...string) (any, error) {
	return nil, unboundDefaultsError(ctx)
}
func (CombinedDefaults) FormatRequirement(ctx context.Context, _ any) ([]string, error) {
	return nil, unboundDefaultsError(ctx)
}
func (CombinedDefaults) DisplayRequirement(ctx context.Context, _ any) ([]string, error) {
	return nil, unboundDefaultsError(ctx)
}
func (CombinedDefaults) Resolve(ctx context.Context, _ Scope, _ any) (any, error) {
	return nil, unboundDefaultsError(ctx)
}
