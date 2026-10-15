package cli

import "testing"

type marshallerConvenienceValue struct{ Text string }

func (value marshallerConvenienceValue) MarshalText() ([]byte, error) { return []byte(value.Text), nil }
func (value *marshallerConvenienceValue) UnmarshalText(input []byte) error {
	value.Text = string(input)
	return nil
}

func TestMarshallerConvenienceOnBothBuilders(t *testing.T) {
	value := marshallerConvenienceValue{Text: "one"}
	values := []marshallerConvenienceValue{{Text: "one"}, {Text: "two"}}
	factory := func() *marshallerConvenienceValue { return new(marshallerConvenienceValue) }
	t.Run("value-flag", flagConvenienceCase(
		(*ApplicationRequirement[NoOptions]).WithMarshallerFlag[marshallerConvenienceValue],
		(*CommandRequirement[NoOptions]).WithMarshallerFlag[marshallerConvenienceValue], []string{"one"}, value))
	t.Run("value-argument", argumentConvenienceCase(
		(*ApplicationRequirement[NoOptions]).WithMarshallerArgument[marshallerConvenienceValue],
		(*CommandRequirement[NoOptions]).WithMarshallerArgument[marshallerConvenienceValue], []string{"one"}, value))
	t.Run("slice-flag", flagConvenienceCase(
		(*ApplicationRequirement[NoOptions]).WithMarshallerSliceFlag[marshallerConvenienceValue],
		(*CommandRequirement[NoOptions]).WithMarshallerSliceFlag[marshallerConvenienceValue], []string{"one", "two"}, values))
	t.Run("slice-argument", argumentConvenienceCase(
		(*ApplicationRequirement[NoOptions]).WithMarshallerSliceArgument[marshallerConvenienceValue],
		(*CommandRequirement[NoOptions]).WithMarshallerSliceArgument[marshallerConvenienceValue], []string{"one", "two"}, values))
	t.Run("pointer-flag", flagConvenienceCase(
		func(root *ApplicationRequirement[NoOptions], name string, configure ...func(*FlagRequirement[*marshallerConvenienceValue, string])) *ApplicationRequirement[NoOptions] {
			return root.WithMarshallerPFlag(name, factory, configure...)
		},
		func(child *CommandRequirement[NoOptions], name string, configure ...func(*FlagRequirement[*marshallerConvenienceValue, string])) *CommandRequirement[NoOptions] {
			return child.WithMarshallerPFlag(name, factory, configure...)
		}, []string{"one"}, &value))
	t.Run("pointer-argument", argumentConvenienceCase(
		func(root *ApplicationRequirement[NoOptions], name string, configure ...func(*ArgumentRequirement[*marshallerConvenienceValue, string])) *ApplicationRequirement[NoOptions] {
			return root.WithMarshallerPArgument(name, factory, configure...)
		},
		func(child *CommandRequirement[NoOptions], name string, configure ...func(*ArgumentRequirement[*marshallerConvenienceValue, string])) *CommandRequirement[NoOptions] {
			return child.WithMarshallerPArgument(name, factory, configure...)
		}, []string{"one"}, &value))
	pointers := []*marshallerConvenienceValue{{Text: "one"}, {Text: "two"}}
	t.Run("pointer-slice-flag", flagConvenienceCase(
		func(root *ApplicationRequirement[NoOptions], name string, configure ...func(*FlagRequirement[[]*marshallerConvenienceValue, []string])) *ApplicationRequirement[NoOptions] {
			return root.WithMarshallerPSliceFlag(name, factory, configure...)
		},
		func(child *CommandRequirement[NoOptions], name string, configure ...func(*FlagRequirement[[]*marshallerConvenienceValue, []string])) *CommandRequirement[NoOptions] {
			return child.WithMarshallerPSliceFlag(name, factory, configure...)
		}, []string{"one", "two"}, pointers))
	t.Run("pointer-slice-argument", argumentConvenienceCase(
		func(root *ApplicationRequirement[NoOptions], name string, configure ...func(*ArgumentRequirement[[]*marshallerConvenienceValue, []string])) *ApplicationRequirement[NoOptions] {
			return root.WithMarshallerPSliceArgument(name, factory, configure...)
		},
		func(child *CommandRequirement[NoOptions], name string, configure ...func(*ArgumentRequirement[[]*marshallerConvenienceValue, []string])) *CommandRequirement[NoOptions] {
			return child.WithMarshallerPSliceArgument(name, factory, configure...)
		}, []string{"one", "two"}, pointers))
}
