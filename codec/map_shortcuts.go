package codec

import (
	"encoding"
	"time"
)

// NewBoolMap constructs a map[string]bool codec. All map shortcuts use string keys.
func NewBoolMap() Codec[map[string]bool, map[string]bool] { return NewMap[string](NewBool()) }

// NewStringMap constructs the standard map[string]string codec.
func NewStringMap() Codec[map[string]string, map[string]string] { return NewMap[string](NewString()) }

// NewDurationMap constructs the standard map[string]time.Duration codec.
func NewDurationMap() Codec[map[string]time.Duration, map[string]time.Duration] {
	return NewMap[string](NewDuration())
}

// NewDateMap constructs a map[string]time.Time codec for date-only inputs.
func NewDateMap() Codec[map[string]time.Time, map[string]time.Time] { return NewMap[string](NewDate()) }

// NewDateTimeMap constructs the standard map[string]time.Time codec.
func NewDateTimeMap() Codec[map[string]time.Time, map[string]time.Time] {
	return NewMap[string](NewDateTime())
}

// NewIntMap constructs the standard map[string]int codec.
func NewIntMap() Codec[map[string]int, map[string]int] { return NewMap[string](NewInt()) }

// NewInt8Map constructs the standard map[string]int8 codec.
func NewInt8Map() Codec[map[string]int8, map[string]int8] { return NewMap[string](NewInt8()) }

// NewInt16Map constructs the standard map[string]int16 codec.
func NewInt16Map() Codec[map[string]int16, map[string]int16] { return NewMap[string](NewInt16()) }

// NewInt32Map constructs the standard map[string]int32 codec.
func NewInt32Map() Codec[map[string]int32, map[string]int32] { return NewMap[string](NewInt32()) }

// NewInt64Map constructs the standard map[string]int64 codec.
func NewInt64Map() Codec[map[string]int64, map[string]int64] { return NewMap[string](NewInt64()) }

// NewUintMap constructs the standard map[string]uint codec.
func NewUintMap() Codec[map[string]uint, map[string]uint] { return NewMap[string](NewUint()) }

// NewUint8Map constructs the standard map[string]uint8 codec.
func NewUint8Map() Codec[map[string]uint8, map[string]uint8] { return NewMap[string](NewUint8()) }

// NewUint16Map constructs the standard map[string]uint16 codec.
func NewUint16Map() Codec[map[string]uint16, map[string]uint16] { return NewMap[string](NewUint16()) }

// NewUint32Map constructs the standard map[string]uint32 codec.
func NewUint32Map() Codec[map[string]uint32, map[string]uint32] { return NewMap[string](NewUint32()) }

// NewUint64Map constructs the standard map[string]uint64 codec.
func NewUint64Map() Codec[map[string]uint64, map[string]uint64] { return NewMap[string](NewUint64()) }

// NewUintptrMap constructs the standard map[string]uintptr codec.
func NewUintptrMap() Codec[map[string]uintptr, map[string]uintptr] {
	return NewMap[string](NewUintptr())
}

// NewFloat32Map constructs the standard map[string]float32 codec.
func NewFloat32Map() Codec[map[string]float32, map[string]float32] {
	return NewMap[string](NewFloat32())
}

// NewFloat64Map constructs the standard map[string]float64 codec.
func NewFloat64Map() Codec[map[string]float64, map[string]float64] {
	return NewMap[string](NewFloat64())
}

// NewComplex64Map constructs the standard map[string]complex64 codec.
func NewComplex64Map() Codec[map[string]complex64, map[string]complex64] {
	return NewMap[string](NewComplex64())
}

// NewComplex128Map constructs the standard map[string]complex128 codec.
func NewComplex128Map() Codec[map[string]complex128, map[string]complex128] {
	return NewMap[string](NewComplex128())
}

// NewMarshallerMap constructs a string-keyed map of text-marshaled values.
func NewMarshallerMap[T encoding.TextMarshaler, TP interface {
	*T
	encoding.TextUnmarshaler
}]() Codec[map[string]T, map[string]string] {
	return NewMap[string](NewMarshaller[T, TP]())
}

// NewMarshallerPMap constructs a string-keyed map of factory-created pointers.
func NewMarshallerPMap[T interface {
	encoding.TextMarshaler
	encoding.TextUnmarshaler
}](factory func() T) Codec[map[string]T, map[string]string] {
	return NewMap[string](NewMarshallerP[T](factory))
}

// NewValueMap constructs a string-keyed map with NewValue elements.
func NewValueMap[T any, TP interface {
	*T
	Value
}](configs ...ValueConfig[T]) (Codec[map[string]T, map[string][]string], error) {
	element, err := NewValue[T, TP](configs...)
	if err != nil {
		return nil, err
	}
	return NewMap[string](element), nil
}

// NewValuePMap constructs a string-keyed map with factory-created pointers.
func NewValuePMap[T Value](factory func() T, configs ...ValueConfig[T]) (Codec[map[string]T, map[string][]string], error) {
	element, err := NewValueP(factory, configs...)
	if err != nil {
		return nil, err
	}
	return NewMap[string](element), nil
}
