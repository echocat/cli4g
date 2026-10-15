package cli

import "github.com/echocat/cli4g/codec"

// WithValueFlag adds a Set/String value flag, keeping T as the field/value type.
func (this *ApplicationRequirement[O]) WithValueFlag[T any, TP interface {
	*T
	codec.Value
}](name string, configure ...func(*FlagRequirement[T, []string])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlagFactory(func() (codec.Codec[T, []string], error) {
		return codec.NewValue[T, TP]()
	}, name, configure...))
}

// WithValuePFlag adds a Set/String flag with a fresh pointer factory.
func (this *ApplicationRequirement[O]) WithValuePFlag[T codec.Value](name string, factory func() T, configure ...func(*FlagRequirement[T, []string])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlagFactory(func() (codec.Codec[T, []string], error) {
		return codec.NewValueP(factory)
	}, name, configure...))
}

// WithValueSliceFlag adds a standard repeated slice flag of Set/String values.
func (this *ApplicationRequirement[O]) WithValueSliceFlag[T any, TP interface {
	*T
	codec.Value
}](name string, configure ...func(*FlagRequirement[[]T, [][]string])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlagFactory(func() (codec.Codec[[]T, [][]string], error) {
		return codec.NewValueSlice[T, TP]()
	}, name, configure...))
}

// WithValuePSliceFlag adds a repeated slice flag of factory-created pointers.
func (this *ApplicationRequirement[O]) WithValuePSliceFlag[T codec.Value](name string, factory func() T, configure ...func(*FlagRequirement[[]T, [][]string])) *ApplicationRequirement[O] {
	return this.WithFlag(newConfiguredFlagFactory(func() (codec.Codec[[]T, [][]string], error) {
		return codec.NewValuePSlice(factory)
	}, name, configure...))
}

// WithValueArgument adds a positional Set/String value argument.
func (this *ApplicationRequirement[O]) WithValueArgument[T any, TP interface {
	*T
	codec.Value
}](name string, configure ...func(*ArgumentRequirement[T, []string])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgumentFactory(func() (codec.Codec[T, []string], error) {
		return codec.NewValue[T, TP]()
	}, name, configure...))
}

// WithValuePArgument adds a positional value with a fresh pointer factory.
func (this *ApplicationRequirement[O]) WithValuePArgument[T codec.Value](name string, factory func() T, configure ...func(*ArgumentRequirement[T, []string])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgumentFactory(func() (codec.Codec[T, []string], error) {
		return codec.NewValueP(factory)
	}, name, configure...))
}

// WithValueSliceArgument adds a final variadic slice of Set/String values.
func (this *ApplicationRequirement[O]) WithValueSliceArgument[T any, TP interface {
	*T
	codec.Value
}](name string, configure ...func(*ArgumentRequirement[[]T, [][]string])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgumentFactory(func() (codec.Codec[[]T, [][]string], error) {
		return codec.NewValueSlice[T, TP]()
	}, name, configure...))
}

// WithValuePSliceArgument adds a final variadic slice of factory-created pointers.
func (this *ApplicationRequirement[O]) WithValuePSliceArgument[T codec.Value](name string, factory func() T, configure ...func(*ArgumentRequirement[[]T, [][]string])) *ApplicationRequirement[O] {
	return this.WithArgument(newConfiguredArgumentFactory(func() (codec.Codec[[]T, [][]string], error) {
		return codec.NewValuePSlice(factory)
	}, name, configure...))
}
