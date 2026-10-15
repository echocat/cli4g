package cli

import (
	"time"

	"github.com/echocat/cli4g/codec"
)

// WithBoolArgument adds a standard bool argument. Like all typed argument
// helpers, it sets the name before applying callbacks in order. Fields bind
// only via WithField. Bool arguments consume one explicit input value.
func (this *ApplicationRequirement[O]) WithBoolArgument(name string, configure ...func(*ArgumentRequirement[bool, bool])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewBool(), name, configure...))
}

// WithStringArgument adds a positional string; bind options with [ArgumentRequirement.WithField].
func (this *ApplicationRequirement[O]) WithStringArgument(name string, configure ...func(*ArgumentRequirement[string, string])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewString(), name, configure...))
}

// WithDurationArgument adds a duration argument; use -- before negative values.
func (this *ApplicationRequirement[O]) WithDurationArgument(name string, configure ...func(*ArgumentRequirement[time.Duration, time.Duration])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewDuration(), name, configure...))
}

// WithDateArgument adds a date-only argument; see [codec.NewDate].
func (this *ApplicationRequirement[O]) WithDateArgument(name string, configure ...func(*ArgumentRequirement[time.Time, time.Time])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewDate(), name, configure...))
}

// WithDateTimeArgument adds a timestamp argument; see [codec.NewDateTime].
func (this *ApplicationRequirement[O]) WithDateTimeArgument(name string, configure ...func(*ArgumentRequirement[time.Time, time.Time])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewDateTime(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithIntArgument(name string, configure ...func(*ArgumentRequirement[int, int])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewInt(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithInt8Argument(name string, configure ...func(*ArgumentRequirement[int8, int8])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewInt8(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithInt16Argument(name string, configure ...func(*ArgumentRequirement[int16, int16])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewInt16(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithInt32Argument(name string, configure ...func(*ArgumentRequirement[int32, int32])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewInt32(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithInt64Argument(name string, configure ...func(*ArgumentRequirement[int64, int64])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewInt64(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUintArgument(name string, configure ...func(*ArgumentRequirement[uint, uint])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUint(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUint8Argument(name string, configure ...func(*ArgumentRequirement[uint8, uint8])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUint8(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUint16Argument(name string, configure ...func(*ArgumentRequirement[uint16, uint16])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUint16(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUint32Argument(name string, configure ...func(*ArgumentRequirement[uint32, uint32])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUint32(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUint64Argument(name string, configure ...func(*ArgumentRequirement[uint64, uint64])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUint64(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUintptrArgument(name string, configure ...func(*ArgumentRequirement[uintptr, uintptr])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUintptr(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithFloat32Argument(name string, configure ...func(*ArgumentRequirement[float32, float32])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewFloat32(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithFloat64Argument(name string, configure ...func(*ArgumentRequirement[float64, float64])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewFloat64(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithComplex64Argument(name string, configure ...func(*ArgumentRequirement[complex64, complex64])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewComplex64(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithComplex128Argument(name string, configure ...func(*ArgumentRequirement[complex128, complex128])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewComplex128(), name, configure...))
}

// WithBoolSliceArgument adds a standard variadic []bool argument. Like all
// standard slice arguments, it must be the final argument in its container.
func (this *ApplicationRequirement[O]) WithBoolSliceArgument(name string, configure ...func(*ArgumentRequirement[[]bool, []bool])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewBoolSlice(), name, configure...))
}

// WithStringSliceArgument adds a variadic string argument; it must be last.
func (this *ApplicationRequirement[O]) WithStringSliceArgument(name string, configure ...func(*ArgumentRequirement[[]string, []string])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewStringSlice(), name, configure...))
}

// WithDurationSliceArgument adds variadic durations; it must be last.
func (this *ApplicationRequirement[O]) WithDurationSliceArgument(name string, configure ...func(*ArgumentRequirement[[]time.Duration, []time.Duration])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewDurationSlice(), name, configure...))
}

// WithDateSliceArgument adds variadic dates; see [codec.NewDateSlice]. It must be last.
func (this *ApplicationRequirement[O]) WithDateSliceArgument(name string, configure ...func(*ArgumentRequirement[[]time.Time, []time.Time])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewDateSlice(), name, configure...))
}

// WithDateTimeSliceArgument adds variadic timestamps; see [codec.NewDateTimeSlice]. It must be last.
func (this *ApplicationRequirement[O]) WithDateTimeSliceArgument(name string, configure ...func(*ArgumentRequirement[[]time.Time, []time.Time])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewDateTimeSlice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithIntSliceArgument(name string, configure ...func(*ArgumentRequirement[[]int, []int])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewIntSlice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithInt8SliceArgument(name string, configure ...func(*ArgumentRequirement[[]int8, []int8])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewInt8Slice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithInt16SliceArgument(name string, configure ...func(*ArgumentRequirement[[]int16, []int16])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewInt16Slice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithInt32SliceArgument(name string, configure ...func(*ArgumentRequirement[[]int32, []int32])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewInt32Slice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithInt64SliceArgument(name string, configure ...func(*ArgumentRequirement[[]int64, []int64])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewInt64Slice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUintSliceArgument(name string, configure ...func(*ArgumentRequirement[[]uint, []uint])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUintSlice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUint8SliceArgument(name string, configure ...func(*ArgumentRequirement[[]uint8, []uint8])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUint8Slice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUint16SliceArgument(name string, configure ...func(*ArgumentRequirement[[]uint16, []uint16])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUint16Slice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUint32SliceArgument(name string, configure ...func(*ArgumentRequirement[[]uint32, []uint32])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUint32Slice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUint64SliceArgument(name string, configure ...func(*ArgumentRequirement[[]uint64, []uint64])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUint64Slice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUintptrSliceArgument(name string, configure ...func(*ArgumentRequirement[[]uintptr, []uintptr])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewUintptrSlice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithFloat32SliceArgument(name string, configure ...func(*ArgumentRequirement[[]float32, []float32])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewFloat32Slice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithFloat64SliceArgument(name string, configure ...func(*ArgumentRequirement[[]float64, []float64])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewFloat64Slice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithComplex64SliceArgument(name string, configure ...func(*ArgumentRequirement[[]complex64, []complex64])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewComplex64Slice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithComplex128SliceArgument(name string, configure ...func(*ArgumentRequirement[[]complex128, []complex128])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewComplex128Slice(), name, configure...))
}
