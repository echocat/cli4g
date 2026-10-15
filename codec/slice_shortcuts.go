package codec

import (
	"encoding"
	"time"
)

// NewBoolSlice constructs the standard []bool codec.
func NewBoolSlice() Codec[[]bool, []bool] { return NewSlice(NewBool()) }

// NewStringSlice constructs the standard []string codec.
func NewStringSlice() Codec[[]string, []string] { return NewSlice(NewString()) }

// NewDurationSlice constructs the standard []time.Duration codec.
func NewDurationSlice() Codec[[]time.Duration, []time.Duration] { return NewSlice(NewDuration()) }

// NewDateSlice constructs a []time.Time codec for date-only inputs.
func NewDateSlice() Codec[[]time.Time, []time.Time] { return NewSlice(NewDate()) }

// NewDateTimeSlice constructs the standard []time.Time codec.
func NewDateTimeSlice() Codec[[]time.Time, []time.Time] { return NewSlice(NewDateTime()) }

// NewIntSlice constructs the standard []int codec.
func NewIntSlice() Codec[[]int, []int] { return NewSlice(NewInt()) }

// NewInt8Slice constructs the standard []int8 codec.
func NewInt8Slice() Codec[[]int8, []int8] { return NewSlice(NewInt8()) }

// NewInt16Slice constructs the standard []int16 codec.
func NewInt16Slice() Codec[[]int16, []int16] { return NewSlice(NewInt16()) }

// NewInt32Slice constructs the standard []int32 codec.
func NewInt32Slice() Codec[[]int32, []int32] { return NewSlice(NewInt32()) }

// NewInt64Slice constructs the standard []int64 codec.
func NewInt64Slice() Codec[[]int64, []int64] { return NewSlice(NewInt64()) }

// NewUintSlice constructs the standard []uint codec.
func NewUintSlice() Codec[[]uint, []uint] { return NewSlice(NewUint()) }

// NewUint8Slice constructs the standard []uint8 codec.
func NewUint8Slice() Codec[[]uint8, []uint8] { return NewSlice(NewUint8()) }

// NewUint16Slice constructs the standard []uint16 codec.
func NewUint16Slice() Codec[[]uint16, []uint16] { return NewSlice(NewUint16()) }

// NewUint32Slice constructs the standard []uint32 codec.
func NewUint32Slice() Codec[[]uint32, []uint32] { return NewSlice(NewUint32()) }

// NewUint64Slice constructs the standard []uint64 codec.
func NewUint64Slice() Codec[[]uint64, []uint64] { return NewSlice(NewUint64()) }

// NewUintptrSlice constructs the standard []uintptr codec.
func NewUintptrSlice() Codec[[]uintptr, []uintptr] { return NewSlice(NewUintptr()) }

// NewFloat32Slice constructs the standard []float32 codec.
func NewFloat32Slice() Codec[[]float32, []float32] { return NewSlice(NewFloat32()) }

// NewFloat64Slice constructs the standard []float64 codec.
func NewFloat64Slice() Codec[[]float64, []float64] { return NewSlice(NewFloat64()) }

// NewComplex64Slice constructs the standard []complex64 codec.
func NewComplex64Slice() Codec[[]complex64, []complex64] { return NewSlice(NewComplex64()) }

// NewComplex128Slice constructs the standard []complex128 codec.
func NewComplex128Slice() Codec[[]complex128, []complex128] { return NewSlice(NewComplex128()) }

// NewMarshallerSlice constructs the standard []Marshaller codec.
func NewMarshallerSlice[T encoding.TextMarshaler, TP interface {
	*T
	encoding.TextUnmarshaler
}]() Codec[[]T, []string] {
	return NewSlice(NewMarshaller[T, TP]())
}

// NewMarshallerPSlice constructs the standard []MarshallerP codec.
func NewMarshallerPSlice[T interface {
	encoding.TextMarshaler
	encoding.TextUnmarshaler
}](factory func() T) Codec[[]T, []string] {
	return NewSlice(NewMarshallerP[T](factory))
}

// NewValueSlice constructs a standard slice codec with NewValue elements.
func NewValueSlice[T any, TP interface {
	*T
	Value
}](configs ...ValueConfig[T]) (Codec[[]T, [][]string], error) {
	element, err := NewValue[T, TP](configs...)
	if err != nil {
		return nil, err
	}
	return NewSlice(element), nil
}

// NewValuePSlice constructs a standard slice codec with factory-created pointers.
func NewValuePSlice[T Value](factory func() T, configs ...ValueConfig[T]) (Codec[[]T, [][]string], error) {
	element, err := NewValueP(factory, configs...)
	if err != nil {
		return nil, err
	}
	return NewSlice(element), nil
}
