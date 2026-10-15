package cli

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/echocat/cli4g/codec"
)

func TestRequirementCodecFactoryIsDeferredAndLastSetterWins(t *testing.T) {
	for _, kind := range []string{"flag", "argument"} {
		for _, selected := range []string{"factory", "codec"} {
			t.Run(kind+"/"+selected, func(t *testing.T) {
				root := NewSimpleApplication().WithName("app").WithHelp(false)
				calls := 0
				factory := func() (codec.Codec[string, string], error) { calls++; return codec.NewString(), nil }
				var args []string
				if kind == "flag" {
					flag := NewFlag(codec.NewString()).WithName("value").WithCodecFactory(factory)
					if selected == "codec" {
						flag.WithCodec(codec.NewString())
					}
					root.WithFlag(flag)
					args = []string{"--value=input"}
				} else {
					argument := NewArgument(codec.NewString()).WithName("value").WithCodecFactory(factory)
					if selected == "codec" {
						argument.WithCodec(codec.NewString())
					}
					root.WithArgument(argument)
					args = []string{"input"}
				}
				assert.Zero(t, calls)
				app, err := root.Create()
				require.NoError(t, err)
				defer func() { assert.NoError(t, app.Close()) }()
				expected := 0
				if selected == "factory" {
					expected = 1
				}
				assert.Equal(t, expected, calls)
				for range 2 {
					invocation, err := app.Parse(nil, nil, args...)
					require.NoError(t, err)
					require.NoError(t, invocation.Close())
				}
				assert.Equal(t, expected, calls, "Parse must use the captured codec, not its schema factory")
			})
		}
	}
}

func TestRequirementCodecFactoryErrorsAndNilResults(t *testing.T) {
	primary := errors.New("schema factory failed")
	for _, kind := range []string{"flag", "argument"} {
		for _, failure := range []string{"error", "nil"} {
			t.Run(kind+"/"+failure, func(t *testing.T) {
				root := NewSimpleApplication().WithName("app").WithHelp(false)
				factory := func() (codec.Codec[string, string], error) {
					if failure == "error" {
						return nil, primary
					}
					return nil, nil
				}
				if kind == "flag" {
					root.WithFlag(NewFlag[string, string](nil).WithName("value").WithCodecFactory(factory))
				} else {
					root.WithArgument(NewArgument[string, string](nil).WithName("value").WithCodecFactory(factory))
				}
				var app *Application[NoOptions]
				var err error
				require.NotPanics(t, func() { app, err = root.Create() })
				assert.Nil(t, app)
				if failure == "error" {
					require.ErrorIs(t, err, primary)
				} else {
					require.ErrorIs(t, err, codec.ErrInvalid)
				}
				require.ErrorContains(t, err, "value")
			})
		}
	}
}

func TestValueFactoryCallbacksCanConfigureTraitsBeforeCodecCreation(t *testing.T) {
	for _, scope := range []string{"application", "command"} {
		for _, kind := range []string{"flag", "argument", "slice-flag", "slice-argument"} {
			t.Run(scope+"/"+kind, func(t *testing.T) {
				root := NewSimpleApplication().WithName("app").WithHelp(false)
				child := NewSimpleCommand().WithName("run")
				var events []string
				factories := 0
				factory := func() *configuredBoolFlag {
					factories++
					return &configuredBoolFlag{Value: new(bool)}
				}
				config := codec.ValueConfig[*configuredBoolFlag]{Traits: &codec.ValueTraits{BoolFlag: true}}
				createCodec := func() (codec.Codec[*configuredBoolFlag, []string], error) {
					events = append(events, "codec")
					return codec.NewValueP(factory, config)
				}
				createSliceCodec := func() (codec.Codec[[]*configuredBoolFlag, [][]string], error) {
					events = append(events, "codec")
					return codec.NewValuePSlice(factory, config)
				}
				configureFlag := func(f *FlagRequirement[*configuredBoolFlag, []string]) {
					events = append(events, "callback")
					f.WithCodecFactory(createCodec)
				}
				configureArgument := func(a *ArgumentRequirement[*configuredBoolFlag, []string]) {
					events = append(events, "callback")
					a.WithCodecFactory(createCodec)
				}
				configureSliceFlag := func(f *FlagRequirement[[]*configuredBoolFlag, [][]string]) {
					events = append(events, "callback")
					f.WithCodecFactory(createSliceCodec)
				}
				configureSliceArgument := func(a *ArgumentRequirement[[]*configuredBoolFlag, [][]string]) {
					events = append(events, "callback")
					a.WithCodecFactory(createSliceCodec)
				}
				require.NotPanics(t, func() {
					if scope == "command" {
						switch kind {
						case "flag":
							child.WithValuePFlag("value", factory, configureFlag)
						case "argument":
							child.WithValuePArgument("value", factory, configureArgument)
						case "slice-flag":
							child.WithValuePSliceFlag("value", factory, configureSliceFlag)
						case "slice-argument":
							child.WithValuePSliceArgument("value", factory, configureSliceArgument)
						}
						root.WithCommand(child)
					} else {
						switch kind {
						case "flag":
							root.WithValuePFlag("value", factory, configureFlag)
						case "argument":
							root.WithValuePArgument("value", factory, configureArgument)
						case "slice-flag":
							root.WithValuePSliceFlag("value", factory, configureSliceFlag)
						case "slice-argument":
							root.WithValuePSliceArgument("value", factory, configureSliceArgument)
						}
					}
				})
				assert.Equal(t, []string{"callback"}, events)
				app, err := root.Create()
				require.NoError(t, err, "the zero trait probe is invalid; the configured traits must be used")
				defer func() { assert.NoError(t, app.Close()) }()
				assert.Equal(t, []string{"callback", "codec"}, events)
				assert.Zero(t, factories, "schema creation must not call the value factory")
				// Both config and trait values have been captured independently.
				config.Traits.BoolFlag = false
				var args []string
				if scope == "command" {
					args = append(args, "run")
				}
				switch kind {
				case "flag":
					args = append(args, "--value")
				case "argument":
					args = append(args, "true")
				case "slice-flag":
					args = append(args, "--value=true", "--value=false")
				case "slice-argument":
					args = append(args, "true", "false")
				}
				invocation, err := app.Parse(nil, nil, args...)
				require.NoError(t, err)
				defer func() { assert.NoError(t, invocation.Close()) }()
				var value any
				if kind == "flag" || kind == "slice-flag" {
					_, value = invocation.Flag("value")
				} else {
					_, value = invocation.Argument(0)
				}
				switch parsed := value.(type) {
				case *configuredBoolFlag:
					assert.True(t, *parsed.Value)
					assert.Equal(t, 1, factories)
				case []*configuredBoolFlag:
					require.Len(t, parsed, 2)
					assert.True(t, *parsed[0].Value)
					assert.False(t, *parsed[1].Value)
					assert.Equal(t, 2, factories)
				default:
					t.Fatalf("unexpected decoded value: %T", value)
				}
				assert.Equal(t, []string{"callback", "codec"}, events)
			})
		}
	}
}

func TestValueFactoryTraitErrorsAreReturnedByCreate(t *testing.T) {
	for _, kind := range []string{"flag", "argument"} {
		t.Run(kind, func(t *testing.T) {
			root := NewSimpleApplication().WithName("app").WithHelp(false)
			factory := func() *configuredBoolFlag { return &configuredBoolFlag{Value: new(bool)} }
			require.NotPanics(t, func() {
				if kind == "flag" {
					root.WithValuePFlag("value", factory)
				} else {
					root.WithValuePArgument("value", factory)
				}
			})
			var app *Application[NoOptions]
			var err error
			require.NotPanics(t, func() { app, err = root.Create() })
			assert.Nil(t, app)
			require.ErrorContains(t, err, "zero-instance traits")
		})
	}
}

func TestValueCodecFactorySnapshotsTokenRequirements(t *testing.T) {
	inputs := []string{"configured"}
	app, err := NewSimpleApplication().WithName("app").WithErrorHandlerEnabled(false).
		WithValueFlag("value", func(f *FlagRequirement[flagReceiverValue, []string]) { f.WithDefault(inputs) }).Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, app.Close()) }()
	inputs[0] = "changed"
	invocation, err := app.Parse(nil, nil)
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, value := invocation.Flag("value")
	assert.Equal(t, "configured", value.(flagReceiverValue).Text)
}
