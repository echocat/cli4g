package cli

import (
	"encoding"
	"time"

	"github.com/echocat/cli4g/codec"
)

// WithBoolArgument adds a standard bool argument. Like all typed argument
// helpers, it sets the name before applying callbacks in order. Fields bind
// only via WithField. Bool arguments consume one explicit input value.
func (this *CommandRequirement[O]) WithBoolArgument(name string, configure ...func(*ArgumentRequirement[bool, bool])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewBool(), name, configure...))
}

// WithStringArgument adds a positional string; bind options with [ArgumentRequirement.WithField].
func (this *CommandRequirement[O]) WithStringArgument(name string, configure ...func(*ArgumentRequirement[string, string])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewString(), name, configure...))
}

// WithDurationArgument adds a duration argument; use -- before negative values.
func (this *CommandRequirement[O]) WithDurationArgument(name string, configure ...func(*ArgumentRequirement[time.Duration, time.Duration])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewDuration(), name, configure...))
}

// WithDateArgument adds a date-only argument; see [codec.NewDate].
func (this *CommandRequirement[O]) WithDateArgument(name string, configure ...func(*ArgumentRequirement[time.Time, time.Time])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewDate(), name, configure...))
}

// WithDateTimeArgument adds a timestamp argument; see [codec.NewDateTime].
func (this *CommandRequirement[O]) WithDateTimeArgument(name string, configure ...func(*ArgumentRequirement[time.Time, time.Time])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewDateTime(), name, configure...))
}

func (this *CommandRequirement[O]) WithIntArgument(name string, configure ...func(*ArgumentRequirement[int, int])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewInt(), name, configure...))
}

func (this *CommandRequirement[O]) WithInt8Argument(name string, configure ...func(*ArgumentRequirement[int8, int8])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewInt8(), name, configure...))
}

func (this *CommandRequirement[O]) WithInt16Argument(name string, configure ...func(*ArgumentRequirement[int16, int16])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewInt16(), name, configure...))
}

func (this *CommandRequirement[O]) WithInt32Argument(name string, configure ...func(*ArgumentRequirement[int32, int32])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewInt32(), name, configure...))
}

func (this *CommandRequirement[O]) WithInt64Argument(name string, configure ...func(*ArgumentRequirement[int64, int64])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewInt64(), name, configure...))
}

func (this *CommandRequirement[O]) WithUintArgument(name string, configure ...func(*ArgumentRequirement[uint, uint])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUint(), name, configure...))
}

func (this *CommandRequirement[O]) WithUint8Argument(name string, configure ...func(*ArgumentRequirement[uint8, uint8])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUint8(), name, configure...))
}

func (this *CommandRequirement[O]) WithUint16Argument(name string, configure ...func(*ArgumentRequirement[uint16, uint16])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUint16(), name, configure...))
}

func (this *CommandRequirement[O]) WithUint32Argument(name string, configure ...func(*ArgumentRequirement[uint32, uint32])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUint32(), name, configure...))
}

func (this *CommandRequirement[O]) WithUint64Argument(name string, configure ...func(*ArgumentRequirement[uint64, uint64])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUint64(), name, configure...))
}

func (this *CommandRequirement[O]) WithUintptrArgument(name string, configure ...func(*ArgumentRequirement[uintptr, uintptr])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUintptr(), name, configure...))
}

func (this *CommandRequirement[O]) WithFloat32Argument(name string, configure ...func(*ArgumentRequirement[float32, float32])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewFloat32(), name, configure...))
}

func (this *CommandRequirement[O]) WithFloat64Argument(name string, configure ...func(*ArgumentRequirement[float64, float64])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewFloat64(), name, configure...))
}

func (this *CommandRequirement[O]) WithComplex64Argument(name string, configure ...func(*ArgumentRequirement[complex64, complex64])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewComplex64(), name, configure...))
}

func (this *CommandRequirement[O]) WithComplex128Argument(name string, configure ...func(*ArgumentRequirement[complex128, complex128])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewComplex128(), name, configure...))
}

func (this *CommandRequirement[O]) WithMarshallerArgument[
	T encoding.TextMarshaler,
	TP interface {
		*T
		encoding.TextUnmarshaler
	},
](name string, configure ...func(*ArgumentRequirement[T, string])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewMarshaller[T, TP](), name, configure...))
}

func (this *CommandRequirement[O]) WithMarshallerPArgument[
	T interface {
		encoding.TextMarshaler
		encoding.TextUnmarshaler
	},
](name string, factory func() T, configure ...func(*ArgumentRequirement[T, string])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgumentFactory(marshallerPointerFactory(factory), name, configure...))
}

// WithBoolSliceArgument adds a standard variadic []bool argument. Like all
// standard slice arguments, it must be the final argument in its container.
func (this *CommandRequirement[O]) WithBoolSliceArgument(name string, configure ...func(*ArgumentRequirement[[]bool, []bool])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewBoolSlice(), name, configure...))
}

// WithStringSliceArgument adds a variadic string argument; it must be last.
func (this *CommandRequirement[O]) WithStringSliceArgument(name string, configure ...func(*ArgumentRequirement[[]string, []string])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewStringSlice(), name, configure...))
}

// WithDurationSliceArgument adds variadic durations; it must be last.
func (this *CommandRequirement[O]) WithDurationSliceArgument(name string, configure ...func(*ArgumentRequirement[[]time.Duration, []time.Duration])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewDurationSlice(), name, configure...))
}

// WithDateSliceArgument adds variadic dates; see [codec.NewDateSlice]. It must be last.
func (this *CommandRequirement[O]) WithDateSliceArgument(name string, configure ...func(*ArgumentRequirement[[]time.Time, []time.Time])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewDateSlice(), name, configure...))
}

// WithDateTimeSliceArgument adds variadic timestamps; see [codec.NewDateTimeSlice]. It must be last.
func (this *CommandRequirement[O]) WithDateTimeSliceArgument(name string, configure ...func(*ArgumentRequirement[[]time.Time, []time.Time])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewDateTimeSlice(), name, configure...))
}

func (this *CommandRequirement[O]) WithIntSliceArgument(name string, configure ...func(*ArgumentRequirement[[]int, []int])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewIntSlice(), name, configure...))
}

func (this *CommandRequirement[O]) WithInt8SliceArgument(name string, configure ...func(*ArgumentRequirement[[]int8, []int8])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewInt8Slice(), name, configure...))
}

func (this *CommandRequirement[O]) WithInt16SliceArgument(name string, configure ...func(*ArgumentRequirement[[]int16, []int16])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewInt16Slice(), name, configure...))
}

func (this *CommandRequirement[O]) WithInt32SliceArgument(name string, configure ...func(*ArgumentRequirement[[]int32, []int32])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewInt32Slice(), name, configure...))
}

func (this *CommandRequirement[O]) WithInt64SliceArgument(name string, configure ...func(*ArgumentRequirement[[]int64, []int64])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewInt64Slice(), name, configure...))
}

func (this *CommandRequirement[O]) WithUintSliceArgument(name string, configure ...func(*ArgumentRequirement[[]uint, []uint])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUintSlice(), name, configure...))
}

func (this *CommandRequirement[O]) WithUint8SliceArgument(name string, configure ...func(*ArgumentRequirement[[]uint8, []uint8])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUint8Slice(), name, configure...))
}

func (this *CommandRequirement[O]) WithUint16SliceArgument(name string, configure ...func(*ArgumentRequirement[[]uint16, []uint16])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUint16Slice(), name, configure...))
}

func (this *CommandRequirement[O]) WithUint32SliceArgument(name string, configure ...func(*ArgumentRequirement[[]uint32, []uint32])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUint32Slice(), name, configure...))
}

func (this *CommandRequirement[O]) WithUint64SliceArgument(name string, configure ...func(*ArgumentRequirement[[]uint64, []uint64])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUint64Slice(), name, configure...))
}

func (this *CommandRequirement[O]) WithUintptrSliceArgument(name string, configure ...func(*ArgumentRequirement[[]uintptr, []uintptr])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUintptrSlice(), name, configure...))
}

func (this *CommandRequirement[O]) WithFloat32SliceArgument(name string, configure ...func(*ArgumentRequirement[[]float32, []float32])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewFloat32Slice(), name, configure...))
}

func (this *CommandRequirement[O]) WithFloat64SliceArgument(name string, configure ...func(*ArgumentRequirement[[]float64, []float64])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewFloat64Slice(), name, configure...))
}

func (this *CommandRequirement[O]) WithComplex64SliceArgument(name string, configure ...func(*ArgumentRequirement[[]complex64, []complex64])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewComplex64Slice(), name, configure...))
}

func (this *CommandRequirement[O]) WithComplex128SliceArgument(name string, configure ...func(*ArgumentRequirement[[]complex128, []complex128])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewComplex128Slice(), name, configure...))
}

func (this *CommandRequirement[O]) WithMarshallerSliceArgument[
	T encoding.TextMarshaler,
	TP interface {
		*T
		encoding.TextUnmarshaler
	},
](name string, configure ...func(*ArgumentRequirement[[]T, []string])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewMarshallerSlice[T, TP](), name, configure...))
}

func (this *CommandRequirement[O]) WithMarshallerPSliceArgument[
	T interface {
		encoding.TextMarshaler
		encoding.TextUnmarshaler
	},
](name string, factory func() T, configure ...func(*ArgumentRequirement[[]T, []string])) *CommandRequirement[O] {
	return this.WithArgument(newConfiguredArgumentFactory(marshallerPointerSliceFactory(factory), name, configure...))
}
