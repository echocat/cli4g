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

func TestFlagConvenienceStandardCodecs(t *testing.T) {
	for _, test := range []struct {
		name string
		run  func(*testing.T)
	}{
		{"Bool", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithBoolFlag, (*CommandRequirement[NoOptions]).WithBoolFlag, []string{"true"}, true)},
		{"String", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithStringFlag, (*CommandRequirement[NoOptions]).WithStringFlag, []string{""}, "")},
		{"Duration", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithDurationFlag, (*CommandRequirement[NoOptions]).WithDurationFlag, []string{"1m30s"}, 90*time.Second)},
		{"Date", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithDateFlag, (*CommandRequirement[NoOptions]).WithDateFlag, []string{"14 Oct 26"}, time.Date(2026, time.October, 14, 0, 0, 0, 0, time.UTC))},
		{"DateTime", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithDateTimeFlag, (*CommandRequirement[NoOptions]).WithDateTimeFlag, []string{"2026-10-14T12:30:45.123456789Z"}, time.Date(2026, time.October, 14, 12, 30, 45, 123456789, time.UTC))},
		{"Int", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithIntFlag, (*CommandRequirement[NoOptions]).WithIntFlag, []string{"-7"}, int(-7))},
		{"Int8", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithInt8Flag, (*CommandRequirement[NoOptions]).WithInt8Flag, []string{"-8"}, int8(-8))},
		{"Int16", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithInt16Flag, (*CommandRequirement[NoOptions]).WithInt16Flag, []string{"-16"}, int16(-16))},
		{"Int32", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithInt32Flag, (*CommandRequirement[NoOptions]).WithInt32Flag, []string{"-32"}, int32(-32))},
		{"Int64", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithInt64Flag, (*CommandRequirement[NoOptions]).WithInt64Flag, []string{"-4294967296"}, int64(-4294967296))},
		{"Uint", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithUintFlag, (*CommandRequirement[NoOptions]).WithUintFlag, []string{"7"}, uint(7))},
		{"Uint8", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithUint8Flag, (*CommandRequirement[NoOptions]).WithUint8Flag, []string{"255"}, uint8(255))},
		{"Uint16", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithUint16Flag, (*CommandRequirement[NoOptions]).WithUint16Flag, []string{"65535"}, uint16(65535))},
		{"Uint32", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithUint32Flag, (*CommandRequirement[NoOptions]).WithUint32Flag, []string{"4294967295"}, uint32(4294967295))},
		{"Uint64", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithUint64Flag, (*CommandRequirement[NoOptions]).WithUint64Flag, []string{"18446744073709551615"}, uint64(18446744073709551615))},
		{"Uintptr", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithUintptrFlag, (*CommandRequirement[NoOptions]).WithUintptrFlag, []string{"15"}, uintptr(15))},
		{"Float32", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithFloat32Flag, (*CommandRequirement[NoOptions]).WithFloat32Flag, []string{"1.25"}, float32(1.25))},
		{"Float64", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithFloat64Flag, (*CommandRequirement[NoOptions]).WithFloat64Flag, []string{"2.5"}, float64(2.5))},
		{"Complex64", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithComplex64Flag, (*CommandRequirement[NoOptions]).WithComplex64Flag, []string{"1+2i"}, complex64(1+2i))},
		{"Complex128", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithComplex128Flag, (*CommandRequirement[NoOptions]).WithComplex128Flag, []string{"2-3i"}, complex128(2-3i))},
		{"BoolSlice", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithBoolSliceFlag, (*CommandRequirement[NoOptions]).WithBoolSliceFlag, []string{"true", "false"}, []bool{true, false})},
		{"StringSlice", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithStringSliceFlag, (*CommandRequirement[NoOptions]).WithStringSliceFlag, []string{"", "two"}, []string{"", "two"})},
		{"DurationSlice", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithDurationSliceFlag, (*CommandRequirement[NoOptions]).WithDurationSliceFlag, []string{"250ms", "-2s"}, []time.Duration{250 * time.Millisecond, -2 * time.Second})},
		{"DateSlice", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithDateSliceFlag, (*CommandRequirement[NoOptions]).WithDateSliceFlag, []string{"2026-10-14", "15.10.2026"}, []time.Time{time.Date(2026, time.October, 14, 0, 0, 0, 0, time.UTC), time.Date(2026, time.October, 15, 0, 0, 0, 0, time.UTC)})},
		{"DateTimeSlice", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithDateTimeSliceFlag, (*CommandRequirement[NoOptions]).WithDateTimeSliceFlag, []string{"2026-10-14T12:30:45Z", "20261015"}, []time.Time{time.Date(2026, time.October, 14, 12, 30, 45, 0, time.UTC), time.Date(2026, time.October, 15, 0, 0, 0, 0, time.UTC)})},
		{"IntSlice", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithIntSliceFlag, (*CommandRequirement[NoOptions]).WithIntSliceFlag, []string{"-1", "2"}, []int{-1, 2})},
		{"Int8Slice", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithInt8SliceFlag, (*CommandRequirement[NoOptions]).WithInt8SliceFlag, []string{"-8", "8"}, []int8{-8, 8})},
		{"Int16Slice", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithInt16SliceFlag, (*CommandRequirement[NoOptions]).WithInt16SliceFlag, []string{"-16", "16"}, []int16{-16, 16})},
		{"Int32Slice", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithInt32SliceFlag, (*CommandRequirement[NoOptions]).WithInt32SliceFlag, []string{"-32", "32"}, []int32{-32, 32})},
		{"Int64Slice", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithInt64SliceFlag, (*CommandRequirement[NoOptions]).WithInt64SliceFlag, []string{"-4294967296", "4294967296"}, []int64{-4294967296, 4294967296})},
		{"UintSlice", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithUintSliceFlag, (*CommandRequirement[NoOptions]).WithUintSliceFlag, []string{"1", "2"}, []uint{1, 2})},
		{"Uint8Slice", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithUint8SliceFlag, (*CommandRequirement[NoOptions]).WithUint8SliceFlag, []string{"0", "255"}, []uint8{0, 255})},
		{"Uint16Slice", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithUint16SliceFlag, (*CommandRequirement[NoOptions]).WithUint16SliceFlag, []string{"0", "65535"}, []uint16{0, 65535})},
		{"Uint32Slice", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithUint32SliceFlag, (*CommandRequirement[NoOptions]).WithUint32SliceFlag, []string{"0", "4294967295"}, []uint32{0, 4294967295})},
		{"Uint64Slice", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithUint64SliceFlag, (*CommandRequirement[NoOptions]).WithUint64SliceFlag, []string{"0", "18446744073709551615"}, []uint64{0, 18446744073709551615})},
		{"UintptrSlice", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithUintptrSliceFlag, (*CommandRequirement[NoOptions]).WithUintptrSliceFlag, []string{"0", "15"}, []uintptr{0, 15})},
		{"Float32Slice", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithFloat32SliceFlag, (*CommandRequirement[NoOptions]).WithFloat32SliceFlag, []string{"1.25", "2.5"}, []float32{1.25, 2.5})},
		{"Float64Slice", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithFloat64SliceFlag, (*CommandRequirement[NoOptions]).WithFloat64SliceFlag, []string{"2.5", "3.75"}, []float64{2.5, 3.75})},
		{"Complex64Slice", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithComplex64SliceFlag, (*CommandRequirement[NoOptions]).WithComplex64SliceFlag, []string{"1+2i", "2-3i"}, []complex64{1 + 2i, 2 - 3i})},
		{"Complex128Slice", flagConvenienceCase((*ApplicationRequirement[NoOptions]).WithComplex128SliceFlag, (*CommandRequirement[NoOptions]).WithComplex128SliceFlag, []string{"2-3i", "4+5i"}, []complex128{2 - 3i, 4 + 5i})},
	} {
		t.Run(test.name, test.run)
	}
}

func flagConvenienceCase[V, R any](
	application func(*ApplicationRequirement[NoOptions], string, ...func(*FlagRequirement[V, R])) *ApplicationRequirement[NoOptions],
	command func(*CommandRequirement[NoOptions], string, ...func(*FlagRequirement[V, R])) *CommandRequirement[NoOptions],
	inputs []string, expected V,
) func(*testing.T) {
	return func(t *testing.T) {
		for _, scope := range []string{"application", "command"} {
			for _, mode := range []string{"plain", "configured", "required-default"} {
				t.Run(scope+"/"+mode, func(t *testing.T) {
					root := NewSimpleApplication().WithName("app").WithHelp(false)
					child := NewSimpleCommand().WithName("run")
					var configure []func(*FlagRequirement[V, R])
					var order []int
					var configured *FlagRequirement[V, R]
					if mode == "configured" {
						configure = []func(*FlagRequirement[V, R]){
							func(f *FlagRequirement[V, R]) {
								assert.Equal(t, []string{"value"}, f.Names, "name must precede callbacks")
								configured = f
								order = append(order, 1)
								f.WithAliases("alias").WithShort('v').WithDefaultStr(inputs...).WithDescription("first")
							},
							func(f *FlagRequirement[V, R]) {
								assert.Same(t, configured, f)
								order = append(order, 2)
								f.WithDescription("last").WithVisible(false)
							},
						}
					} else if mode == "required-default" {
						configure = []func(*FlagRequirement[V, R]){func(f *FlagRequirement[V, R]) {
							f.WithRequired(true).WithDefaultStr(inputs...)
						}}
					}
					var prefix []string
					if scope == "command" {
						require.Same(t, child, command(child, "value", configure...))
						require.Len(t, child.Flags, 1)
						root.WithCommand(child)
						prefix = []string{"run"}
					} else {
						require.Same(t, root, application(root, "value", configure...))
						require.Len(t, root.Flags, 1)
					}
					app, err := root.Create()
					if mode == "required-default" {
						require.ErrorContains(t, err, "required cannot be combined with a default")
						assert.Nil(t, app)
						return
					}
					require.NoError(t, err)
					defer func() { assert.NoError(t, app.Close()) }()
					args := append([]string(nil), prefix...)
					for index, input := range inputs {
						name := "--value="
						if mode == "configured" {
							name = "--alias="
							if index == 0 {
								name = "-v="
							}
						}
						args = append(args, name+input)
					}
					invocation, err := app.Parse(nil, nil, args...)
					require.NoError(t, err)
					defer func() { assert.NoError(t, invocation.Close()) }()
					definition, parsed := invocation.Flag("value")
					require.NotNil(t, definition)
					assert.Equal(t, reflect.TypeFor[V](), definition.ValueType())
					assert.Equal(t, expected, parsed)
					assert.Equal(t, mode == "plain", definition.IsVisible())
					if mode == "configured" {
						assert.Equal(t, []int{1, 2}, order)
						assert.Equal(t, "last", definition.Description())
						alias, value := invocation.Flag("alias")
						assert.Same(t, definition, alias)
						assert.Equal(t, expected, value)
						fallback, err := app.Parse(nil, nil, prefix...)
						require.NoError(t, err)
						defer func() { assert.NoError(t, fallback.Close()) }()
						_, value = fallback.Flag("value")
						assert.Equal(t, expected, value)
					}
				})
			}
		}
	}
}

type flagConvenienceOptions struct {
	Port    uint32
	Ports   []uint32
	Verbose bool
}

// The explicit builder form is the behavioral reference for field bindings,
// override priority, validation and duplicate errors, rather than a new parser.
func portConvenienceRequirement(scope, form string, configure ...func(*FlagRequirement[uint32, uint32])) *ApplicationRequirement[flagConvenienceOptions] {
	root := NewApplication[flagConvenienceOptions]().WithName("app").WithHelp(false).
		WithErrorHandler(func(_ context.Context, _ InvocationLike, err error) error { return err })
	child := NewCommand[flagConvenienceOptions]().WithName("run", "r")
	if form == "explicit" {
		flag := NewFlag(codec.NewUint32()).WithName("port")
		for _, callback := range configure {
			callback(flag)
		}
		if scope == "command" {
			child.WithFlag(flag)
		} else {
			root.WithFlag(flag)
		}
	} else if scope == "command" {
		child.WithUint32Flag("port", configure...)
	} else {
		root.WithUint32Flag("port", configure...)
	}
	if scope == "command" {
		root.WithCommand(child)
	}
	return root
}

func flagConvenienceParseArgs(scope string, args ...string) []string {
	if scope == "command" {
		return append([]string{"r"}, args...)
	}
	return args
}

func flagConvenienceParsedOptions(t *testing.T, scope string, invocation *Invocation[flagConvenienceOptions]) flagConvenienceOptions {
	t.Helper()
	if scope == "command" {
		child, found := FindCommand[flagConvenienceOptions](invocation.Root(), "run")
		require.True(t, found)
		options, found := child.OptionsFrom(invocation)
		require.True(t, found)
		return options
	}
	_, options := invocation.Application()
	return options
}

func TestFlagConvenienceFieldBindingDefaultsAndAliasesMatchExplicitBuilder(t *testing.T) {
	for _, scope := range []string{"application", "command"} {
		for _, form := range []string{"convenience", "explicit"} {
			for _, binding := range []string{"bound", "unbound"} {
				t.Run(scope+"/"+form+"/"+binding, func(t *testing.T) {
					root := portConvenienceRequirement(scope, form, func(f *FlagRequirement[uint32, uint32]) {
						f.WithAliases("listen").WithShort('p', 'P').WithDefault(8080).
							WithDescription("Port to listen on.").WithPlaceholder("PORT")
						if binding == "bound" {
							f.WithField("Port")
						}
					})
					app, err := root.Create()
					require.NoError(t, err)
					defer func() { assert.NoError(t, app.Close()) }()
					for _, test := range []struct {
						args     []string
						expected uint32
					}{
						{nil, 8080}, {[]string{"--port=9000"}, 9000},
						{[]string{"--listen=9001"}, 9001}, {[]string{"-P=9002"}, 9002},
					} {
						invocation, err := app.Parse(nil, nil, flagConvenienceParseArgs(scope, test.args...)...)
						require.NoError(t, err)
						options := flagConvenienceParsedOptions(t, scope, invocation)
						if binding == "bound" {
							assert.Equal(t, test.expected, options.Port)
						} else {
							assert.Zero(t, options.Port, "port must not implicitly bind Port")
						}
						definition, value := invocation.Flag("port")
						assert.Equal(t, test.expected, value)
						assert.Equal(t, []string{"port", "listen"}, definition.Names())
						assert.Equal(t, []rune{'p', 'P'}, definition.Shorts())
						assert.Equal(t, "Port to listen on.", definition.Description())
						assert.Equal(t, "PORT", definition.Placeholder())
						require.NoError(t, invocation.Close())
					}
					_, err = app.Parse(nil, nil, flagConvenienceParseArgs(scope, "--port=1", "--listen=2")...)
					require.ErrorContains(t, err, "supplied more than once")
				})
			}
		}
	}
}

func TestFlagConvenienceRequiredMatchesExplicitBuilder(t *testing.T) {
	for _, scope := range []string{"application", "command"} {
		for _, form := range []string{"convenience", "explicit"} {
			t.Run(scope+"/"+form, func(t *testing.T) {
				root := portConvenienceRequirement(scope, form, func(f *FlagRequirement[uint32, uint32]) {
					f.WithField("Port").WithAliases("listen").WithRequired(true)
				})
				app, err := root.Create()
				require.NoError(t, err)
				defer func() { assert.NoError(t, app.Close()) }()
				invocation, err := app.Parse(nil, nil, flagConvenienceParseArgs(scope)...)
				require.ErrorContains(t, err, "required flag --port is missing")
				assert.Nil(t, invocation)
				invocation, err = app.Parse(nil, nil, flagConvenienceParseArgs(scope, "--listen=0")...)
				require.NoError(t, err)
				require.NoError(t, invocation.Close())
				definition, value := invocation.Flag("port")
				assert.True(t, definition.IsRequired())
				assert.Equal(t, uint32(0), value, "an explicit zero satisfies Required")
				_, err = portConvenienceRequirement(scope, form, func(f *FlagRequirement[uint32, uint32]) {
					f.WithRequired(true).WithDefault(0)
				}).Create()
				require.ErrorContains(t, err, "required cannot be combined with a default")
			})
		}
	}
}

func TestFlagConvenienceBoolAndSliceFieldSemantics(t *testing.T) {
	for _, scope := range []string{"application", "command"} {
		t.Run(scope, func(t *testing.T) {
			root := NewApplication[flagConvenienceOptions]().WithName("app").WithHelp(false)
			child := NewCommand[flagConvenienceOptions]().WithName("run", "r")
			prototype := []uint32{8080, 8443}
			configureBool := func(f *FlagRequirement[bool, bool]) { f.WithField("Verbose").WithShort('v') }
			configureSlice := func(f *FlagRequirement[[]uint32, []uint32]) {
				f.WithField("Ports").WithAliases("listen-port").WithDefault(prototype)
			}
			if scope == "command" {
				child.WithBoolFlag("verbose", configureBool).WithUint32SliceFlag("ports", configureSlice)
				root.WithCommand(child)
			} else {
				root.WithBoolFlag("verbose", configureBool).WithUint32SliceFlag("ports", configureSlice)
			}
			app, err := root.Create()
			require.NoError(t, err)
			defer func() { assert.NoError(t, app.Close()) }()
			prototype[0] = 1
			for range 2 {
				invocation, err := app.Parse(nil, nil, flagConvenienceParseArgs(scope)...)
				require.NoError(t, err)
				options := flagConvenienceParsedOptions(t, scope, invocation)
				assert.Equal(t, []uint32{8080, 8443}, options.Ports)
				options.Ports[0] = 2
				require.NoError(t, invocation.Close())
			}
			invocation, err := app.Parse(nil, nil, flagConvenienceParseArgs(scope, "-v", "--listen-port=9000", "--ports=9001")...)
			require.NoError(t, err)
			options := flagConvenienceParsedOptions(t, scope, invocation)
			assert.True(t, options.Verbose)
			assert.Equal(t, []uint32{9000, 9001}, options.Ports, "CLI replaces defaults, then aggregates occurrences")
			require.NoError(t, invocation.Close())
		})
	}
}
