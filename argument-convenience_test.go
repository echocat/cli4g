package cli

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/echocat/cli4g/codec"
)

func TestArgumentConvenienceStandardCodecs(t *testing.T) {
	for _, test := range []struct {
		name string
		run  func(*testing.T)
	}{
		{"Bool", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithBoolArgument, (*CommandRequirement[NoOptions]).WithBoolArgument, []string{"true"}, true)},
		{"String", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithStringArgument, (*CommandRequirement[NoOptions]).WithStringArgument, []string{""}, "")},
		{"Duration", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithDurationArgument, (*CommandRequirement[NoOptions]).WithDurationArgument, []string{"1m30s"}, 90*time.Second)},
		{"Date", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithDateArgument, (*CommandRequirement[NoOptions]).WithDateArgument, []string{"14 Oct 26"}, time.Date(2026, time.October, 14, 0, 0, 0, 0, time.UTC))},
		{"DateTime", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithDateTimeArgument, (*CommandRequirement[NoOptions]).WithDateTimeArgument, []string{"2026-10-14T12:30:45.123456789Z"}, time.Date(2026, time.October, 14, 12, 30, 45, 123456789, time.UTC))},
		{"Int", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithIntArgument, (*CommandRequirement[NoOptions]).WithIntArgument, []string{"-7"}, int(-7))},
		{"Int8", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithInt8Argument, (*CommandRequirement[NoOptions]).WithInt8Argument, []string{"-8"}, int8(-8))},
		{"Int16", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithInt16Argument, (*CommandRequirement[NoOptions]).WithInt16Argument, []string{"-16"}, int16(-16))},
		{"Int32", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithInt32Argument, (*CommandRequirement[NoOptions]).WithInt32Argument, []string{"-32"}, int32(-32))},
		{"Int64", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithInt64Argument, (*CommandRequirement[NoOptions]).WithInt64Argument, []string{"-4294967296"}, int64(-4294967296))},
		{"Uint", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithUintArgument, (*CommandRequirement[NoOptions]).WithUintArgument, []string{"7"}, uint(7))},
		{"Uint8", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithUint8Argument, (*CommandRequirement[NoOptions]).WithUint8Argument, []string{"255"}, uint8(255))},
		{"Uint16", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithUint16Argument, (*CommandRequirement[NoOptions]).WithUint16Argument, []string{"65535"}, uint16(65535))},
		{"Uint32", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithUint32Argument, (*CommandRequirement[NoOptions]).WithUint32Argument, []string{"4294967295"}, uint32(4294967295))},
		{"Uint64", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithUint64Argument, (*CommandRequirement[NoOptions]).WithUint64Argument, []string{"18446744073709551615"}, uint64(18446744073709551615))},
		{"Uintptr", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithUintptrArgument, (*CommandRequirement[NoOptions]).WithUintptrArgument, []string{"15"}, uintptr(15))},
		{"Float32", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithFloat32Argument, (*CommandRequirement[NoOptions]).WithFloat32Argument, []string{"1.25"}, float32(1.25))},
		{"Float64", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithFloat64Argument, (*CommandRequirement[NoOptions]).WithFloat64Argument, []string{"2.5"}, float64(2.5))},
		{"Complex64", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithComplex64Argument, (*CommandRequirement[NoOptions]).WithComplex64Argument, []string{"1+2i"}, complex64(1+2i))},
		{"Complex128", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithComplex128Argument, (*CommandRequirement[NoOptions]).WithComplex128Argument, []string{"2-3i"}, complex128(2-3i))},
		{"BoolSlice", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithBoolSliceArgument, (*CommandRequirement[NoOptions]).WithBoolSliceArgument, []string{"true", "false"}, []bool{true, false})},
		{"StringSlice", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithStringSliceArgument, (*CommandRequirement[NoOptions]).WithStringSliceArgument, []string{"", "two"}, []string{"", "two"})},
		{"DurationSlice", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithDurationSliceArgument, (*CommandRequirement[NoOptions]).WithDurationSliceArgument, []string{"250ms", "-2s"}, []time.Duration{250 * time.Millisecond, -2 * time.Second})},
		{"DateSlice", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithDateSliceArgument, (*CommandRequirement[NoOptions]).WithDateSliceArgument, []string{"2026-10-14", "15.10.2026"}, []time.Time{time.Date(2026, time.October, 14, 0, 0, 0, 0, time.UTC), time.Date(2026, time.October, 15, 0, 0, 0, 0, time.UTC)})},
		{"DateTimeSlice", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithDateTimeSliceArgument, (*CommandRequirement[NoOptions]).WithDateTimeSliceArgument, []string{"2026-10-14T12:30:45Z", "20261015"}, []time.Time{time.Date(2026, time.October, 14, 12, 30, 45, 0, time.UTC), time.Date(2026, time.October, 15, 0, 0, 0, 0, time.UTC)})},
		{"IntSlice", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithIntSliceArgument, (*CommandRequirement[NoOptions]).WithIntSliceArgument, []string{"-1", "2"}, []int{-1, 2})},
		{"Int8Slice", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithInt8SliceArgument, (*CommandRequirement[NoOptions]).WithInt8SliceArgument, []string{"-8", "8"}, []int8{-8, 8})},
		{"Int16Slice", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithInt16SliceArgument, (*CommandRequirement[NoOptions]).WithInt16SliceArgument, []string{"-16", "16"}, []int16{-16, 16})},
		{"Int32Slice", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithInt32SliceArgument, (*CommandRequirement[NoOptions]).WithInt32SliceArgument, []string{"-32", "32"}, []int32{-32, 32})},
		{"Int64Slice", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithInt64SliceArgument, (*CommandRequirement[NoOptions]).WithInt64SliceArgument, []string{"-4294967296", "4294967296"}, []int64{-4294967296, 4294967296})},
		{"UintSlice", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithUintSliceArgument, (*CommandRequirement[NoOptions]).WithUintSliceArgument, []string{"1", "2"}, []uint{1, 2})},
		{"Uint8Slice", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithUint8SliceArgument, (*CommandRequirement[NoOptions]).WithUint8SliceArgument, []string{"0", "255"}, []uint8{0, 255})},
		{"Uint16Slice", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithUint16SliceArgument, (*CommandRequirement[NoOptions]).WithUint16SliceArgument, []string{"0", "65535"}, []uint16{0, 65535})},
		{"Uint32Slice", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithUint32SliceArgument, (*CommandRequirement[NoOptions]).WithUint32SliceArgument, []string{"0", "4294967295"}, []uint32{0, 4294967295})},
		{"Uint64Slice", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithUint64SliceArgument, (*CommandRequirement[NoOptions]).WithUint64SliceArgument, []string{"0", "18446744073709551615"}, []uint64{0, 18446744073709551615})},
		{"UintptrSlice", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithUintptrSliceArgument, (*CommandRequirement[NoOptions]).WithUintptrSliceArgument, []string{"0", "15"}, []uintptr{0, 15})},
		{"Float32Slice", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithFloat32SliceArgument, (*CommandRequirement[NoOptions]).WithFloat32SliceArgument, []string{"1.25", "2.5"}, []float32{1.25, 2.5})},
		{"Float64Slice", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithFloat64SliceArgument, (*CommandRequirement[NoOptions]).WithFloat64SliceArgument, []string{"2.5", "3.75"}, []float64{2.5, 3.75})},
		{"Complex64Slice", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithComplex64SliceArgument, (*CommandRequirement[NoOptions]).WithComplex64SliceArgument, []string{"1+2i", "2-3i"}, []complex64{1 + 2i, 2 - 3i})},
		{"Complex128Slice", argumentConvenienceCase((*ApplicationRequirement[NoOptions]).WithComplex128SliceArgument, (*CommandRequirement[NoOptions]).WithComplex128SliceArgument, []string{"2-3i", "4+5i"}, []complex128{2 - 3i, 4 + 5i})},
	} {
		t.Run(test.name, test.run)
	}
}

func argumentConvenienceCase[V, R any](
	application func(*ApplicationRequirement[NoOptions], string, ...func(*ArgumentRequirement[V, R])) *ApplicationRequirement[NoOptions],
	command func(*CommandRequirement[NoOptions], string, ...func(*ArgumentRequirement[V, R])) *CommandRequirement[NoOptions],
	inputs []string, expected V,
) func(*testing.T) {
	return func(t *testing.T) {
		for _, scope := range []string{"application", "command"} {
			for _, mode := range []string{"plain", "configured", "required-default"} {
				t.Run(scope+"/"+mode, func(t *testing.T) {
					root := NewSimpleApplication().WithName("app").WithHelp(false)
					child := NewSimpleCommand().WithName("run")
					var configure []func(*ArgumentRequirement[V, R])
					var order []int
					var configured *ArgumentRequirement[V, R]
					if mode == "configured" {
						configure = []func(*ArgumentRequirement[V, R]){
							func(a *ArgumentRequirement[V, R]) {
								assert.Equal(t, "value", a.Name, "name must precede callbacks")
								configured = a
								order = append(order, 1)
								a.WithDefaultStr(inputs...).WithDescription("first")
							},
							func(a *ArgumentRequirement[V, R]) {
								assert.Same(t, configured, a)
								order = append(order, 2)
								a.WithName("renamed").WithDescription("last").WithPlaceholder("VALUE")
							},
						}
					} else if mode == "required-default" {
						configure = []func(*ArgumentRequirement[V, R]){func(a *ArgumentRequirement[V, R]) {
							a.WithRequired(true).WithDefaultStr(inputs...)
						}}
					}
					var prefix []string
					if scope == "command" {
						require.Same(t, child, command(child, "value", configure...))
						require.Len(t, child.Arguments, 1)
						root.WithCommand(child)
						prefix = []string{"run"}
					} else {
						require.Same(t, root, application(root, "value", configure...))
						require.Len(t, root.Arguments, 1)
					}
					app, err := root.Create()
					if mode == "required-default" {
						require.ErrorContains(t, err, "required cannot be combined with a default")
						assert.Nil(t, app)
						return
					}
					require.NoError(t, err)
					defer func() { assert.NoError(t, app.Close()) }()
					args := append(append([]string(nil), prefix...), "--")
					args = append(args, inputs...)
					invocation, err := app.Parse(nil, nil, args...)
					require.NoError(t, err)
					defer func() { assert.NoError(t, invocation.Close()) }()
					definition, parsed := invocation.Argument(0)
					require.NotNil(t, definition)
					assert.Equal(t, reflect.TypeFor[V](), definition.ValueType())
					assert.Equal(t, expected, parsed)
					count := 0
					for range invocation.Arguments() {
						count++
					}
					assert.Equal(t, 1, count, "one entry per definition, including variadic arguments")
					if mode == "configured" {
						assert.Equal(t, []int{1, 2}, order)
						assert.Equal(t, "renamed", definition.Name())
						assert.Equal(t, "last", definition.Description())
						assert.Equal(t, "VALUE", definition.Placeholder())
					} else {
						assert.Equal(t, "value", definition.Name())
					}
					fallback, err := app.Parse(nil, nil, prefix...)
					require.NoError(t, err)
					defer func() { assert.NoError(t, fallback.Close()) }()
					omitted, value := fallback.Argument(0)
					if mode == "configured" {
						assert.Same(t, definition, omitted)
						assert.Equal(t, expected, value)
					} else {
						assert.Nil(t, omitted)
						assert.Nil(t, value)
					}
				})
			}
		}
	}
}

type argumentConvenienceOptions struct {
	Port    uint32
	Ports   []uint32
	Enabled bool
}

func argumentConvenienceRequirements() (*ApplicationRequirement[argumentConvenienceOptions], *CommandRequirement[argumentConvenienceOptions]) {
	root := NewApplication[argumentConvenienceOptions]().WithName("app").WithHelp(false).
		WithErrorHandler(func(_ context.Context, _ InvocationLike, err error) error { return err })
	return root, NewCommand[argumentConvenienceOptions]().WithName("run", "r")
}

func argumentConvenienceArgs(scope string, inputs ...string) []string {
	if scope == "command" {
		return append([]string{"r", "--"}, inputs...)
	}
	return append([]string{"--"}, inputs...)
}

func argumentConvenienceOptionsFrom(t *testing.T, scope string, invocation *Invocation[argumentConvenienceOptions]) argumentConvenienceOptions {
	t.Helper()
	if scope == "command" {
		child, found := FindCommand[argumentConvenienceOptions](invocation.Root(), "run")
		require.True(t, found)
		options, found := child.OptionsFrom(invocation)
		require.True(t, found)
		return options
	}
	_, options := invocation.Application()
	return options
}

func TestArgumentConvenienceBindingAndDefaultsMatchExplicitBuilder(t *testing.T) {
	for _, scope := range []string{"application", "command"} {
		for _, form := range []string{"convenience", "explicit"} {
			for _, binding := range []string{"bound", "unbound"} {
				t.Run(scope+"/"+form+"/"+binding, func(t *testing.T) {
					root, child := argumentConvenienceRequirements()
					configure := func(a *ArgumentRequirement[uint32, uint32]) {
						a.WithDefault(8080).WithDescription("Port to listen on.").WithPlaceholder("PORT")
						if binding == "bound" {
							a.WithField("Port")
						}
					}
					if form == "explicit" {
						argument := NewArgument(codec.NewUint32()).WithName("port")
						configure(argument)
						if scope == "command" {
							child.WithArgument(argument)
						} else {
							root.WithArgument(argument)
						}
					} else if scope == "command" {
						child.WithUint32Argument("port", configure)
					} else {
						root.WithUint32Argument("port", configure)
					}
					if scope == "command" {
						root.WithCommand(child)
					}
					app, err := root.Create()
					require.NoError(t, err)
					defer func() { assert.NoError(t, app.Close()) }()
					for _, test := range []struct {
						inputs   []string
						expected uint32
					}{
						{nil, 8080}, {[]string{"9000"}, 9000}, {[]string{"0"}, 0},
					} {
						invocation, err := app.Parse(nil, nil, argumentConvenienceArgs(scope, test.inputs...)...)
						require.NoError(t, err)
						options := argumentConvenienceOptionsFrom(t, scope, invocation)
						if binding == "bound" {
							assert.Equal(t, test.expected, options.Port)
						} else {
							assert.Zero(t, options.Port, "port must not implicitly bind Port")
						}
						definition, parsed := invocation.Argument(0)
						assert.Equal(t, test.expected, parsed)
						assert.Equal(t, "Port to listen on.", definition.Description())
						assert.Equal(t, "PORT", definition.Placeholder())
						require.NoError(t, invocation.Close())
					}
					_, err = app.Parse(nil, nil, argumentConvenienceArgs(scope, "1", "2")...)
					require.ErrorIs(t, err, codec.ErrTooManyArguments)
				})
			}
		}
	}
}

func TestArgumentConvenienceRequiredBoolNeedsExplicitValue(t *testing.T) {
	for _, scope := range []string{"application", "command"} {
		t.Run(scope, func(t *testing.T) {
			root, child := argumentConvenienceRequirements()
			configure := func(a *ArgumentRequirement[bool, bool]) { a.WithField("Enabled").WithRequired(true) }
			if scope == "command" {
				child.WithBoolArgument("enabled", configure)
				root.WithCommand(child)
			} else {
				root.WithBoolArgument("enabled", configure)
			}
			app, err := root.Create()
			require.NoError(t, err)
			defer func() { assert.NoError(t, app.Close()) }()
			invocation, err := app.Parse(nil, nil, argumentConvenienceArgs(scope)...)
			require.ErrorContains(t, err, "required argument enabled is missing")
			assert.Nil(t, invocation)
			for _, input := range []string{"true", "false"} {
				invocation, err := app.Parse(nil, nil, argumentConvenienceArgs(scope, input)...)
				require.NoError(t, err)
				definition, parsed := invocation.Argument(0)
				assert.True(t, definition.IsRequired())
				assert.Equal(t, input == "true", parsed)
				assert.Equal(t, input == "true", argumentConvenienceOptionsFrom(t, scope, invocation).Enabled)
				require.NoError(t, invocation.Close())
			}
		})
	}
}

func TestArgumentConvenienceVariadicDefaultsAndIndices(t *testing.T) {
	for _, scope := range []string{"application", "command"} {
		t.Run(scope, func(t *testing.T) {
			root, child := argumentConvenienceRequirements()
			prototype := []uint32{8080, 8443}
			configure := func(a *ArgumentRequirement[[]uint32, []uint32]) { a.WithField("Ports").WithDefault(prototype) }
			if scope == "command" {
				child.WithStringArgument("unused").WithUint32SliceArgument("ports", configure)
				root.WithCommand(child)
			} else {
				root.WithStringArgument("unused").WithUint32SliceArgument("ports", configure)
			}
			app, err := root.Create()
			require.NoError(t, err)
			defer func() { assert.NoError(t, app.Close()) }()
			prototype[0] = 1
			for range 2 {
				invocation, err := app.Parse(nil, nil, argumentConvenienceArgs(scope)...)
				require.NoError(t, err)
				definition, value := invocation.Argument(0)
				assert.Nil(t, definition)
				assert.Nil(t, value)
				definition, value = invocation.Argument(1)
				require.NotNil(t, definition)
				assert.Equal(t, "ports", definition.Name())
				assert.Equal(t, []uint32{8080, 8443}, value)
				options := argumentConvenienceOptionsFrom(t, scope, invocation)
				assert.Equal(t, []uint32{8080, 8443}, options.Ports)
				options.Ports[0] = 2
				require.NoError(t, invocation.Close())
			}
			invocation, err := app.Parse(nil, nil, argumentConvenienceArgs(scope, "supplied", "9000", "9001", "9002")...)
			require.NoError(t, err)
			assert.Equal(t, []uint32{9000, 9001, 9002}, argumentConvenienceOptionsFrom(t, scope, invocation).Ports)
			_, value := invocation.Argument(1)
			assert.Equal(t, []uint32{9000, 9001, 9002}, value)
			require.NoError(t, invocation.Close())
		})
	}
}

func TestArgumentConvenienceRetainsVariadicAndParentRequiredConstraints(t *testing.T) {
	for _, scope := range []string{"application", "command"} {
		t.Run(scope, func(t *testing.T) {
			root := NewSimpleApplication().WithName("app").WithHelp(false)
			child := NewSimpleCommand().WithName("run")
			if scope == "command" {
				child.WithStringSliceArgument("items").WithStringArgument("last")
				root.WithCommand(child)
			} else {
				root.WithStringSliceArgument("items").WithStringArgument("last")
			}
			_, err := root.Create()
			require.ErrorContains(t, err, "only the final argument may be variadic")
			root = NewSimpleApplication().WithName("app").WithHelp(false)
			child = NewSimpleCommand().WithName("run")
			configure := func(a *ArgumentRequirement[uint32, uint32]) { a.WithRequired(true) }
			if scope == "command" {
				child.WithUint32Argument("value", configure).WithCommand(NewSimpleCommand().WithName("leaf"))
				root.WithCommand(child)
			} else {
				root.WithUint32Argument("value", configure).WithCommand(child)
			}
			_, err = root.Create()
			require.ErrorContains(t, err, "required positional argument")
		})
	}
}
