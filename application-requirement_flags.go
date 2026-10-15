package cli

import (
	"time"

	"github.com/echocat/cli4g/codec"
)

// WithBoolFlag adds a standard bool flag. Like all typed flag helpers, it sets
// the name before applying callbacks in order. Fields bind only via WithField.
func (this *ApplicationRequirement[O]) WithBoolFlag(name string, configure ...func(*FlagRequirement[bool, bool])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewBool(), name, configure...))
}

// WithStringFlag adds a string flag; bind options with [FlagRequirement.WithField].
func (this *ApplicationRequirement[O]) WithStringFlag(name string, configure ...func(*FlagRequirement[string, string])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewString(), name, configure...))
}

// WithDurationFlag adds a flag accepting Go duration syntax; see [codec.NewDuration].
func (this *ApplicationRequirement[O]) WithDurationFlag(name string, configure ...func(*FlagRequirement[time.Duration, time.Duration])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewDuration(), name, configure...))
}

// WithDateFlag adds a date-only flag; see [codec.NewDate].
func (this *ApplicationRequirement[O]) WithDateFlag(name string, configure ...func(*FlagRequirement[time.Time, time.Time])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewDate(), name, configure...))
}

// WithDateTimeFlag adds a timestamp flag; see [codec.NewDateTime].
func (this *ApplicationRequirement[O]) WithDateTimeFlag(name string, configure ...func(*FlagRequirement[time.Time, time.Time])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewDateTime(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithIntFlag(name string, configure ...func(*FlagRequirement[int, int])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewInt(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithInt8Flag(name string, configure ...func(*FlagRequirement[int8, int8])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewInt8(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithInt16Flag(name string, configure ...func(*FlagRequirement[int16, int16])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewInt16(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithInt32Flag(name string, configure ...func(*FlagRequirement[int32, int32])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewInt32(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithInt64Flag(name string, configure ...func(*FlagRequirement[int64, int64])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewInt64(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUintFlag(name string, configure ...func(*FlagRequirement[uint, uint])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUint(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUint8Flag(name string, configure ...func(*FlagRequirement[uint8, uint8])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUint8(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUint16Flag(name string, configure ...func(*FlagRequirement[uint16, uint16])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUint16(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUint32Flag(name string, configure ...func(*FlagRequirement[uint32, uint32])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUint32(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUint64Flag(name string, configure ...func(*FlagRequirement[uint64, uint64])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUint64(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUintptrFlag(name string, configure ...func(*FlagRequirement[uintptr, uintptr])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUintptr(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithFloat32Flag(name string, configure ...func(*FlagRequirement[float32, float32])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewFloat32(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithFloat64Flag(name string, configure ...func(*FlagRequirement[float64, float64])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewFloat64(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithComplex64Flag(name string, configure ...func(*FlagRequirement[complex64, complex64])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewComplex64(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithComplex128Flag(name string, configure ...func(*FlagRequirement[complex128, complex128])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewComplex128(), name, configure...))
}

// WithBoolSliceFlag adds a standard []bool flag. Slice helpers accept one value
// per occurrence and retain the standard codec's aggregation/default semantics.
func (this *ApplicationRequirement[O]) WithBoolSliceFlag(name string, configure ...func(*FlagRequirement[[]bool, []bool])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewBoolSlice(), name, configure...))
}

// WithStringSliceFlag adds a repeatable string flag, one element per occurrence.
func (this *ApplicationRequirement[O]) WithStringSliceFlag(name string, configure ...func(*FlagRequirement[[]string, []string])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewStringSlice(), name, configure...))
}

// WithDurationSliceFlag adds a repeatable duration flag; see [codec.NewDurationSlice].
func (this *ApplicationRequirement[O]) WithDurationSliceFlag(name string, configure ...func(*FlagRequirement[[]time.Duration, []time.Duration])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewDurationSlice(), name, configure...))
}

// WithDateSliceFlag adds a repeatable date-only flag; see [codec.NewDateSlice].
func (this *ApplicationRequirement[O]) WithDateSliceFlag(name string, configure ...func(*FlagRequirement[[]time.Time, []time.Time])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewDateSlice(), name, configure...))
}

// WithDateTimeSliceFlag adds a repeatable timestamp flag; see [codec.NewDateTimeSlice].
func (this *ApplicationRequirement[O]) WithDateTimeSliceFlag(name string, configure ...func(*FlagRequirement[[]time.Time, []time.Time])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewDateTimeSlice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithIntSliceFlag(name string, configure ...func(*FlagRequirement[[]int, []int])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewIntSlice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithInt8SliceFlag(name string, configure ...func(*FlagRequirement[[]int8, []int8])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewInt8Slice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithInt16SliceFlag(name string, configure ...func(*FlagRequirement[[]int16, []int16])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewInt16Slice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithInt32SliceFlag(name string, configure ...func(*FlagRequirement[[]int32, []int32])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewInt32Slice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithInt64SliceFlag(name string, configure ...func(*FlagRequirement[[]int64, []int64])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewInt64Slice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUintSliceFlag(name string, configure ...func(*FlagRequirement[[]uint, []uint])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUintSlice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUint8SliceFlag(name string, configure ...func(*FlagRequirement[[]uint8, []uint8])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUint8Slice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUint16SliceFlag(name string, configure ...func(*FlagRequirement[[]uint16, []uint16])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUint16Slice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUint32SliceFlag(name string, configure ...func(*FlagRequirement[[]uint32, []uint32])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUint32Slice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUint64SliceFlag(name string, configure ...func(*FlagRequirement[[]uint64, []uint64])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUint64Slice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithUintptrSliceFlag(name string, configure ...func(*FlagRequirement[[]uintptr, []uintptr])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUintptrSlice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithFloat32SliceFlag(name string, configure ...func(*FlagRequirement[[]float32, []float32])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewFloat32Slice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithFloat64SliceFlag(name string, configure ...func(*FlagRequirement[[]float64, []float64])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewFloat64Slice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithComplex64SliceFlag(name string, configure ...func(*FlagRequirement[[]complex64, []complex64])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewComplex64Slice(), name, configure...))
}

func (this *ApplicationRequirement[O]) WithComplex128SliceFlag(name string, configure ...func(*FlagRequirement[[]complex128, []complex128])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewComplex128Slice(), name, configure...))
}
