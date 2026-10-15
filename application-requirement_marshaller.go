package cli

import (
	"encoding"

	"github.com/echocat/cli4g/codec"
)

func (this *ApplicationRequirement[O]) WithMarshallerFlag[T encoding.TextMarshaler, TP interface {
	*T
	encoding.TextUnmarshaler
}](name string, configure ...func(*FlagRequirement[T, string])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewMarshaller[T, TP](), name, configure...))
}

func (this *ApplicationRequirement[O]) WithMarshallerPFlag[T interface {
	encoding.TextMarshaler
	encoding.TextUnmarshaler
}](name string, factory func() T, configure ...func(*FlagRequirement[T, string])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlagFactory(marshallerPointerFactory(factory), name, configure...))
}

func (this *ApplicationRequirement[O]) WithMarshallerSliceFlag[T encoding.TextMarshaler, TP interface {
	*T
	encoding.TextUnmarshaler
}](name string, configure ...func(*FlagRequirement[[]T, []string])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlag(codec.NewMarshallerSlice[T, TP](), name, configure...))
}

func (this *ApplicationRequirement[O]) WithMarshallerPSliceFlag[T interface {
	encoding.TextMarshaler
	encoding.TextUnmarshaler
}](name string, factory func() T, configure ...func(*FlagRequirement[[]T, []string])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlagFactory(marshallerPointerSliceFactory(factory), name, configure...))
}

func (this *ApplicationRequirement[O]) WithMarshallerArgument[T encoding.TextMarshaler, TP interface {
	*T
	encoding.TextUnmarshaler
}](name string, configure ...func(*ArgumentRequirement[T, string])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewMarshaller[T, TP](), name, configure...))
}

func (this *ApplicationRequirement[O]) WithMarshallerPArgument[T interface {
	encoding.TextMarshaler
	encoding.TextUnmarshaler
}](name string, factory func() T, configure ...func(*ArgumentRequirement[T, string])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgumentFactory(marshallerPointerFactory(factory), name, configure...))
}

func (this *ApplicationRequirement[O]) WithMarshallerSliceArgument[T encoding.TextMarshaler, TP interface {
	*T
	encoding.TextUnmarshaler
}](name string, configure ...func(*ArgumentRequirement[[]T, []string])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgument(codec.NewMarshallerSlice[T, TP](), name, configure...))
}

func (this *ApplicationRequirement[O]) WithMarshallerPSliceArgument[T interface {
	encoding.TextMarshaler
	encoding.TextUnmarshaler
}](name string, factory func() T, configure ...func(*ArgumentRequirement[[]T, []string])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgumentFactory(marshallerPointerSliceFactory(factory), name, configure...))
}
