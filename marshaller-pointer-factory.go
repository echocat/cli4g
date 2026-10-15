package cli

import (
	"encoding"
	"fmt"
	"github.com/echocat/cli4g/codec"
)

// Defer schema validation until Create, after typed builder callbacks have had
// the opportunity to replace the codec or its factory. Never acquire a value.
func marshallerPointerFactory[T interface {
	encoding.TextMarshaler
	encoding.TextUnmarshaler
}](factory func() T) func() (codec.Codec[T, string], error) {
	return func() (codec.Codec[T, string], error) {
		if factory == nil {
			return nil, fmt.Errorf("%w: nil marshaller pointer factory", codec.ErrInvalid)
		}
		return codec.NewMarshallerP(factory), nil
	}
}
func marshallerPointerSliceFactory[T interface {
	encoding.TextMarshaler
	encoding.TextUnmarshaler
}](factory func() T) func() (codec.Codec[[]T, []string], error) {
	return func() (codec.Codec[[]T, []string], error) {
		element, err := marshallerPointerFactory(factory)()
		if err != nil {
			return nil, err
		}
		return codec.NewSlice(element), nil
	}
}

func marshallerPointerMapFactory[T interface {
	encoding.TextMarshaler
	encoding.TextUnmarshaler
}](factory func() T) func() (codec.Codec[map[string]T, map[string]string], error) {
	return func() (codec.Codec[map[string]T, map[string]string], error) {
		element, err := marshallerPointerFactory(factory)()
		if err != nil {
			return nil, err
		}
		return codec.NewMap[string](element), nil
	}
}
