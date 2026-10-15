package cli

import (
	"encoding"
	"time"

	"github.com/echocat/cli4g/codec"
)

// WithBoolFlag adds a standard bool flag. Like all typed flag helpers, it sets
// the name before applying callbacks in order. Fields bind only via WithField.
func (this *CommandRequirement[O]) WithBoolFlag(name string, configure ...func(*FlagRequirement[bool, bool])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewBool(), name, configure...))
}

// WithStringFlag adds a string flag; bind options with [FlagRequirement.WithField].
func (this *CommandRequirement[O]) WithStringFlag(name string, configure ...func(*FlagRequirement[string, string])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewString(), name, configure...))
}

// WithDurationFlag adds a flag accepting Go duration syntax; see [codec.NewDuration].
func (this *CommandRequirement[O]) WithDurationFlag(name string, configure ...func(*FlagRequirement[time.Duration, time.Duration])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewDuration(), name, configure...))
}

// WithDateFlag adds a date-only flag; see [codec.NewDate].
func (this *CommandRequirement[O]) WithDateFlag(name string, configure ...func(*FlagRequirement[time.Time, time.Time])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewDate(), name, configure...))
}

// WithDateTimeFlag adds a timestamp flag; see [codec.NewDateTime].
func (this *CommandRequirement[O]) WithDateTimeFlag(name string, configure ...func(*FlagRequirement[time.Time, time.Time])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewDateTime(), name, configure...))
}

func (this *CommandRequirement[O]) WithIntFlag(name string, configure ...func(*FlagRequirement[int, int])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewInt(), name, configure...))
}

func (this *CommandRequirement[O]) WithInt8Flag(name string, configure ...func(*FlagRequirement[int8, int8])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewInt8(), name, configure...))
}

func (this *CommandRequirement[O]) WithInt16Flag(name string, configure ...func(*FlagRequirement[int16, int16])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewInt16(), name, configure...))
}

func (this *CommandRequirement[O]) WithInt32Flag(name string, configure ...func(*FlagRequirement[int32, int32])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewInt32(), name, configure...))
}

func (this *CommandRequirement[O]) WithInt64Flag(name string, configure ...func(*FlagRequirement[int64, int64])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewInt64(), name, configure...))
}

func (this *CommandRequirement[O]) WithUintFlag(name string, configure ...func(*FlagRequirement[uint, uint])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUint(), name, configure...))
}

func (this *CommandRequirement[O]) WithUint8Flag(name string, configure ...func(*FlagRequirement[uint8, uint8])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUint8(), name, configure...))
}

func (this *CommandRequirement[O]) WithUint16Flag(name string, configure ...func(*FlagRequirement[uint16, uint16])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUint16(), name, configure...))
}

func (this *CommandRequirement[O]) WithUint32Flag(name string, configure ...func(*FlagRequirement[uint32, uint32])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUint32(), name, configure...))
}

func (this *CommandRequirement[O]) WithUint64Flag(name string, configure ...func(*FlagRequirement[uint64, uint64])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUint64(), name, configure...))
}

func (this *CommandRequirement[O]) WithUintptrFlag(name string, configure ...func(*FlagRequirement[uintptr, uintptr])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUintptr(), name, configure...))
}

func (this *CommandRequirement[O]) WithFloat32Flag(name string, configure ...func(*FlagRequirement[float32, float32])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewFloat32(), name, configure...))
}

func (this *CommandRequirement[O]) WithFloat64Flag(name string, configure ...func(*FlagRequirement[float64, float64])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewFloat64(), name, configure...))
}

func (this *CommandRequirement[O]) WithComplex64Flag(name string, configure ...func(*FlagRequirement[complex64, complex64])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewComplex64(), name, configure...))
}

func (this *CommandRequirement[O]) WithComplex128Flag(name string, configure ...func(*FlagRequirement[complex128, complex128])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewComplex128(), name, configure...))
}

func (this *CommandRequirement[O]) WithMarshallerFlag[
	T encoding.TextMarshaler,
	TP interface {
		*T
		encoding.TextUnmarshaler
	},
](name string, configure ...func(*FlagRequirement[T, string])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewMarshaller[T, TP](), name, configure...))
}

func (this *CommandRequirement[O]) WithMarshallerPFlag[
	T interface {
		encoding.TextMarshaler
		encoding.TextUnmarshaler
	},
](name string, factory func() T, configure ...func(*FlagRequirement[T, string])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlagFactory(marshallerPointerFactory(factory), name, configure...))
}

// WithBoolSliceFlag adds a standard []bool flag. Slice helpers accept one value
// per occurrence and retain the standard codec's aggregation/default semantics.
func (this *CommandRequirement[O]) WithBoolSliceFlag(name string, configure ...func(*FlagRequirement[[]bool, []bool])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewBoolSlice(), name, configure...))
}

// WithStringSliceFlag adds a repeatable string flag, one element per occurrence.
func (this *CommandRequirement[O]) WithStringSliceFlag(name string, configure ...func(*FlagRequirement[[]string, []string])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewStringSlice(), name, configure...))
}

// WithDurationSliceFlag adds a repeatable duration flag; see [codec.NewDurationSlice].
func (this *CommandRequirement[O]) WithDurationSliceFlag(name string, configure ...func(*FlagRequirement[[]time.Duration, []time.Duration])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewDurationSlice(), name, configure...))
}

// WithDateSliceFlag adds a repeatable date-only flag; see [codec.NewDateSlice].
func (this *CommandRequirement[O]) WithDateSliceFlag(name string, configure ...func(*FlagRequirement[[]time.Time, []time.Time])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewDateSlice(), name, configure...))
}

// WithDateTimeSliceFlag adds a repeatable timestamp flag; see [codec.NewDateTimeSlice].
func (this *CommandRequirement[O]) WithDateTimeSliceFlag(name string, configure ...func(*FlagRequirement[[]time.Time, []time.Time])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewDateTimeSlice(), name, configure...))
}

func (this *CommandRequirement[O]) WithIntSliceFlag(name string, configure ...func(*FlagRequirement[[]int, []int])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewIntSlice(), name, configure...))
}

func (this *CommandRequirement[O]) WithInt8SliceFlag(name string, configure ...func(*FlagRequirement[[]int8, []int8])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewInt8Slice(), name, configure...))
}

func (this *CommandRequirement[O]) WithInt16SliceFlag(name string, configure ...func(*FlagRequirement[[]int16, []int16])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewInt16Slice(), name, configure...))
}

func (this *CommandRequirement[O]) WithInt32SliceFlag(name string, configure ...func(*FlagRequirement[[]int32, []int32])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewInt32Slice(), name, configure...))
}

func (this *CommandRequirement[O]) WithInt64SliceFlag(name string, configure ...func(*FlagRequirement[[]int64, []int64])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewInt64Slice(), name, configure...))
}

func (this *CommandRequirement[O]) WithUintSliceFlag(name string, configure ...func(*FlagRequirement[[]uint, []uint])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUintSlice(), name, configure...))
}

func (this *CommandRequirement[O]) WithUint8SliceFlag(name string, configure ...func(*FlagRequirement[[]uint8, []uint8])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUint8Slice(), name, configure...))
}

func (this *CommandRequirement[O]) WithUint16SliceFlag(name string, configure ...func(*FlagRequirement[[]uint16, []uint16])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUint16Slice(), name, configure...))
}

func (this *CommandRequirement[O]) WithUint32SliceFlag(name string, configure ...func(*FlagRequirement[[]uint32, []uint32])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUint32Slice(), name, configure...))
}

func (this *CommandRequirement[O]) WithUint64SliceFlag(name string, configure ...func(*FlagRequirement[[]uint64, []uint64])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUint64Slice(), name, configure...))
}

func (this *CommandRequirement[O]) WithUintptrSliceFlag(name string, configure ...func(*FlagRequirement[[]uintptr, []uintptr])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewUintptrSlice(), name, configure...))
}

func (this *CommandRequirement[O]) WithFloat32SliceFlag(name string, configure ...func(*FlagRequirement[[]float32, []float32])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewFloat32Slice(), name, configure...))
}

func (this *CommandRequirement[O]) WithFloat64SliceFlag(name string, configure ...func(*FlagRequirement[[]float64, []float64])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewFloat64Slice(), name, configure...))
}

func (this *CommandRequirement[O]) WithComplex64SliceFlag(name string, configure ...func(*FlagRequirement[[]complex64, []complex64])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewComplex64Slice(), name, configure...))
}

func (this *CommandRequirement[O]) WithComplex128SliceFlag(name string, configure ...func(*FlagRequirement[[]complex128, []complex128])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewComplex128Slice(), name, configure...))
}

func (this *CommandRequirement[O]) WithMarshallerSliceFlag[
	T encoding.TextMarshaler,
	TP interface {
		*T
		encoding.TextUnmarshaler
	},
](name string, configure ...func(*FlagRequirement[[]T, []string])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewMarshallerSlice[T, TP](), name, configure...))
}

func (this *CommandRequirement[O]) WithMarshallerPSliceFlag[
	T interface {
		encoding.TextMarshaler
		encoding.TextUnmarshaler
	},
](name string, factory func() T, configure ...func(*FlagRequirement[[]T, []string])) *CommandRequirement[O] {
	return this.WithFlag(newConfiguredFlagFactory(marshallerPointerSliceFactory(factory), name, configure...))
}
