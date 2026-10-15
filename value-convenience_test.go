package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type valueConvenienceOptions struct {
	Value    flagReceiverValue
	Pointer  *flagReceiverValue
	Values   []flagReceiverValue
	Pointers []*flagReceiverValue
	Bool     flagReceiverBool
	Map      *flagReceiverMap
}

func TestValueConvenienceFlagsAndArguments(t *testing.T) {
	for _, scope := range []string{"application", "command"} {
		for _, kind := range []string{"flag", "pointer-flag", "slice-flag", "pointer-slice-flag", "argument", "pointer-argument", "slice-argument", "pointer-slice-argument"} {
			t.Run(scope+"/"+kind, func(t *testing.T) {
				root := NewApplication[valueConvenienceOptions]().WithName("app").WithHelp(false)
				child := NewCommand[valueConvenienceOptions]().WithName("run", "r")
				factories := 0
				factory := func() *flagReceiverValue { factories++; return &flagReceiverValue{Hidden: 10} }
				configureFlag := func(f *FlagRequirement[flagReceiverValue, []string]) {
					f.WithField("Value").WithAliases("alias").WithDefault([]string{"default"})
				}
				configurePFlag := func(f *FlagRequirement[*flagReceiverValue, []string]) {
					f.WithField("Pointer").WithAliases("alias")
				}
				configureSliceFlag := func(f *FlagRequirement[[]flagReceiverValue, [][]string]) {
					f.WithField("Values").WithAliases("alias")
				}
				configurePSliceFlag := func(f *FlagRequirement[[]*flagReceiverValue, [][]string]) {
					f.WithField("Pointers").WithAliases("alias")
				}
				configureArg := func(a *ArgumentRequirement[flagReceiverValue, []string]) {
					a.WithField("Value").WithDefault([]string{"default"})
				}
				configurePArg := func(a *ArgumentRequirement[*flagReceiverValue, []string]) { a.WithField("Pointer") }
				configureSliceArg := func(a *ArgumentRequirement[[]flagReceiverValue, [][]string]) { a.WithField("Values") }
				configurePSliceArg := func(a *ArgumentRequirement[[]*flagReceiverValue, [][]string]) { a.WithField("Pointers") }
				if scope == "command" {
					switch kind {
					case "flag":
						require.Same(t, child, child.WithValueFlag("value", configureFlag))
					case "pointer-flag":
						require.Same(t, child, child.WithValuePFlag("value", factory, configurePFlag))
					case "slice-flag":
						require.Same(t, child, child.WithValueSliceFlag("value", configureSliceFlag))
					case "pointer-slice-flag":
						require.Same(t, child, child.WithValuePSliceFlag("value", factory, configurePSliceFlag))
					case "argument":
						require.Same(t, child, child.WithValueArgument("value", configureArg))
					case "pointer-argument":
						require.Same(t, child, child.WithValuePArgument("value", factory, configurePArg))
					case "slice-argument":
						require.Same(t, child, child.WithValueSliceArgument("value", configureSliceArg))
					case "pointer-slice-argument":
						require.Same(t, child, child.WithValuePSliceArgument("value", factory, configurePSliceArg))
					}
					root.WithCommand(child)
				} else {
					switch kind {
					case "flag":
						require.Same(t, root, root.WithValueFlag("value", configureFlag))
					case "pointer-flag":
						require.Same(t, root, root.WithValuePFlag("value", factory, configurePFlag))
					case "slice-flag":
						require.Same(t, root, root.WithValueSliceFlag("value", configureSliceFlag))
					case "pointer-slice-flag":
						require.Same(t, root, root.WithValuePSliceFlag("value", factory, configurePSliceFlag))
					case "argument":
						require.Same(t, root, root.WithValueArgument("value", configureArg))
					case "pointer-argument":
						require.Same(t, root, root.WithValuePArgument("value", factory, configurePArg))
					case "slice-argument":
						require.Same(t, root, root.WithValueSliceArgument("value", configureSliceArg))
					case "pointer-slice-argument":
						require.Same(t, root, root.WithValuePSliceArgument("value", factory, configurePSliceArg))
					}
				}
				app, err := root.Create()
				require.NoError(t, err)
				defer func() { assert.NoError(t, app.Close()) }()
				assert.Zero(t, factories)
				var args []string
				if scope == "command" {
					args = append(args, "r")
				}
				switch kind {
				case "flag", "pointer-flag":
					args = append(args, "--alias=first")
				case "slice-flag", "pointer-slice-flag":
					args = append(args, "--value=first", "--alias=second")
				case "argument", "pointer-argument":
					args = append(args, "first")
				case "slice-argument", "pointer-slice-argument":
					args = append(args, "first", "second")
				}
				invocation, err := app.Parse(nil, nil, args...)
				require.NoError(t, err)
				defer func() { assert.NoError(t, invocation.Close()) }()
				options := valueConvenienceParsedOptions(t, scope, invocation)
				switch kind {
				case "flag", "argument":
					assert.Equal(t, flagReceiverValue{Text: "first", Hidden: 1}, options.Value)
				case "pointer-flag", "pointer-argument":
					require.NotNil(t, options.Pointer)
					assert.Equal(t, flagReceiverValue{Text: "first", Hidden: 11}, *options.Pointer)
					assert.Equal(t, 1, factories)
				case "slice-flag", "slice-argument":
					assert.Equal(t, []flagReceiverValue{{Text: "first", Hidden: 1}, {Text: "second", Hidden: 1}}, options.Values)
				case "pointer-slice-flag", "pointer-slice-argument":
					require.Len(t, options.Pointers, 2)
					assert.NotSame(t, options.Pointers[0], options.Pointers[1])
					assert.Equal(t, "first", options.Pointers[0].Text)
					assert.Equal(t, "second", options.Pointers[1].Text)
					assert.Equal(t, 2, factories)
				}
				if kind == "flag" || kind == "argument" {
					var prefix []string
					if scope == "command" {
						prefix = []string{"r"}
					}
					fallback, err := app.Parse(nil, nil, prefix...)
					require.NoError(t, err)
					defer func() { assert.NoError(t, fallback.Close()) }()
					assert.Equal(t, flagReceiverValue{Text: "default", Hidden: 1}, valueConvenienceParsedOptions(t, scope, fallback).Value)
				}
			})
		}
	}
}

func valueConvenienceParsedOptions(t *testing.T, scope string, invocation *Invocation[valueConvenienceOptions]) valueConvenienceOptions {
	t.Helper()
	if scope == "command" {
		command, found := FindCommand[valueConvenienceOptions](invocation.Root(), "run")
		require.True(t, found)
		options, found := command.OptionsFrom(invocation)
		require.True(t, found)
		return options
	}
	_, options := invocation.Application()
	return options
}

func TestValueConvenienceBoolAndCumulativeTraits(t *testing.T) {
	for _, scope := range []string{"application", "command"} {
		t.Run(scope, func(t *testing.T) {
			root := NewApplication[valueConvenienceOptions]().WithName("app").WithHelp(false)
			child := NewCommand[valueConvenienceOptions]().WithName("run", "r")
			factories := 0
			factory := func() *flagReceiverMap { factories++; return new(flagReceiverMap) }
			configureBool := func(f *FlagRequirement[flagReceiverBool, []string]) { f.WithField("Bool").WithShort('b') }
			configureMap := func(f *FlagRequirement[*flagReceiverMap, []string]) { f.WithField("Map") }
			if scope == "command" {
				child.WithValueFlag("bool", configureBool).WithValuePFlag("map", factory, configureMap)
				root.WithCommand(child)
			} else {
				root.WithValueFlag("bool", configureBool).WithValuePFlag("map", factory, configureMap)
			}
			app, err := root.Create()
			require.NoError(t, err)
			defer func() { assert.NoError(t, app.Close()) }()
			args := []string{"-b", "--map=a=one", "--map=b=two"}
			if scope == "command" {
				args = append([]string{"r"}, args...)
			}
			invocation, err := app.Parse(context.Background(), nil, args...)
			require.NoError(t, err)
			defer func() { assert.NoError(t, invocation.Close()) }()
			options := valueConvenienceParsedOptions(t, scope, invocation)
			assert.True(t, bool(options.Bool))
			require.NotNil(t, options.Map)
			assert.Equal(t, flagReceiverMap{"a": "one", "b": "two"}, *options.Map)
			assert.Equal(t, 1, factories)
		})
	}
}

func TestValueConvenienceWithoutCallbacks(t *testing.T) {
	for _, scope := range []string{"application", "command"} {
		t.Run(scope, func(t *testing.T) {
			root := NewSimpleApplication().WithName("app").WithHelp(false)
			child := NewSimpleCommand().WithName("run")
			if scope == "command" {
				child.WithValueFlag[flagReceiverValue]("value").WithValueSliceArgument[flagReceiverValue]("arguments")
				root.WithCommand(child)
			} else {
				root.WithValueFlag[flagReceiverValue]("value").WithValueSliceArgument[flagReceiverValue]("arguments")
			}
			app, err := root.Create()
			require.NoError(t, err)
			defer func() { assert.NoError(t, app.Close()) }()
			args := []string{"--value=flag", "one", "two"}
			if scope == "command" {
				args = append([]string{"run"}, args...)
			}
			invocation, err := app.Parse(nil, nil, args...)
			require.NoError(t, err)
			defer func() { assert.NoError(t, invocation.Close()) }()
			_, value := invocation.Flag("value")
			assert.Equal(t, flagReceiverValue{Text: "flag", Hidden: 1}, value)
			_, value = invocation.Argument(0)
			assert.Equal(t, []flagReceiverValue{{Text: "one", Hidden: 1}, {Text: "two", Hidden: 1}}, value)
		})
	}
}
