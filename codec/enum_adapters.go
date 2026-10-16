package codec

import (
	"encoding"
	"fmt"
)

// NewMarshallerEnum restricts a text adapter using typed runtime alternatives.
// MarshalText is called once per supplied value during construction, never
// during help or parsing. UnmarshalText still runs on fresh targets in Resolve.
// Warning: custom ValidationError messages are trusted; see NewMarshaller.
func NewMarshallerEnum[T encoding.TextMarshaler, TP interface {
	*T
	encoding.TextUnmarshaler
}](values ...T) (Codec[T, string], error) {
	return NewMarshallerEnumVariants[T, TP](enumVariants(values)...)
}

// NewMarshallerEnumVariants adds descriptions to typed text alternatives.
// Warning: custom ValidationError messages are trusted; see NewMarshaller.
func NewMarshallerEnumVariants[T encoding.TextMarshaler, TP interface {
	*T
	encoding.TextUnmarshaler
}](variants ...EnumVariant[T]) (Codec[T, string], error) {
	values := make([]T, len(variants))
	for i, variant := range variants {
		values[i] = variant.Value
	}
	requirements, err := enumTextRequirements(values)
	if err != nil {
		return nil, err
	}
	result := enumVariants(requirements)
	for i := range result {
		result[i].Description = variants[i].Description
	}
	return NewEnumVariants(NewMarshaller[T, TP](), result...)
}

// NewMarshallerPEnum is NewMarshallerEnum for pointer targets created by a
// caller-supplied factory. The factory is not called during construction/help.
// Warning: custom ValidationError messages are trusted; see NewMarshallerP.
func NewMarshallerPEnum[T interface {
	encoding.TextMarshaler
	encoding.TextUnmarshaler
}](factory func() T, values ...T) (Codec[T, string], error) {
	if factory == nil {
		return nil, fmt.Errorf("%w: nil enum text factory", ErrInvalid)
	}
	requirements, err := enumTextRequirements(values)
	if err != nil {
		return nil, err
	}
	return NewEnum(NewMarshallerP(factory), requirements...)
}

func enumTextRequirements[T encoding.TextMarshaler](values []T) ([]string, error) {
	result := make([]string, len(values))
	for i, value := range values {
		if isNil(value) {
			return nil, fmt.Errorf("%w: nil enum variant %d", ErrInvalid, i)
		}
		text, err := value.MarshalText()
		if err != nil {
			return nil, prefixError(fmt.Sprintf("enum variant %d", i), SanitizeError(err, "enum choice could not be formatted"))
		}
		result[i] = string(text)
	}
	return result, nil
}

// NewValueEnum restricts a Set/String adapter using typed runtime alternatives.
// For this opt-in adapter, String must produce a canonical input accepted by
// Set. Distinct alternatives must have distinct strings. Runtime String is not
// used as serialization: only the declared strings are retained as requirements.
// Warning: custom ValidationError messages are trusted; see NewValue.
func NewValueEnum[T any, TP interface {
	*T
	Value
}](values ...T) (Codec[T, []string], error) {
	return NewValueEnumVariants[T, TP](enumVariants(values)...)
}

// NewValueEnumVariants adds descriptions to typed Set/String alternatives.
// Warning: custom ValidationError messages are trusted; see NewValue.
func NewValueEnumVariants[T any, TP interface {
	*T
	Value
}](variants ...EnumVariant[T]) (Codec[T, []string], error) {
	source, err := NewValue[T, TP]()
	if err != nil {
		return nil, err
	}
	requirements := make([]EnumVariant[[]string], len(variants))
	for i, variant := range variants {
		value := variant.Value
		requirements[i] = EnumVariant[[]string]{Value: []string{TP(&value).String()}, Description: variant.Description}
	}
	return NewEnumVariants(source, requirements...)
}

// NewValuePEnum is NewValueEnum for factory-created pointers. Supplied values
// are borrowed configuration inputs; the adapter never retains or closes them.
// Warning: custom ValidationError messages are trusted; see NewValueP.
func NewValuePEnum[T Value](factory func() T, values ...T) (Codec[T, []string], error) {
	source, err := NewValueP(factory)
	if err != nil {
		return nil, err
	}
	requirements := make([][]string, len(values))
	for i, value := range values {
		if isNil(value) {
			return nil, fmt.Errorf("%w: nil enum variant %d", ErrInvalid, i)
		}
		requirements[i] = []string{value.String()}
	}
	return NewEnum(source, requirements...)
}
