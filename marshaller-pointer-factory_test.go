package cli

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/echocat/cli4g/codec"
)

type pointerFactoryText struct{}

func (*pointerFactoryText) MarshalText() ([]byte, error) { return nil, nil }
func (*pointerFactoryText) UnmarshalText([]byte) error   { return nil }

func TestMarshallerPointerHelpersDeferInvalidFactoryUntilCreate(t *testing.T) {
	var factory func() *pointerFactoryText
	for _, scope := range []string{"app", "command"} {
		for _, kind := range []string{"flag", "slice-flag", "map-flag", "argument", "slice-argument", "map-argument"} {
			t.Run(scope+"/"+kind, func(t *testing.T) {
				root := NewSimpleApplication().WithName("app")
				child := NewSimpleCommand().WithName("run")
				require.NotPanics(t, func() {
					if scope == "app" {
						switch kind {
						case "flag":
							root.WithMarshallerPFlag("value", factory)
						case "slice-flag":
							root.WithMarshallerPSliceFlag("value", factory)
						case "map-flag":
							root.WithMarshallerPMapFlag("value", factory)
						case "argument":
							root.WithMarshallerPArgument("value", factory)
						case "slice-argument":
							root.WithMarshallerPSliceArgument("value", factory)
						case "map-argument":
							root.WithMarshallerPMapArgument("value", factory)
						}
					} else {
						switch kind {
						case "flag":
							child.WithMarshallerPFlag("value", factory)
						case "slice-flag":
							child.WithMarshallerPSliceFlag("value", factory)
						case "map-flag":
							child.WithMarshallerPMapFlag("value", factory)
						case "argument":
							child.WithMarshallerPArgument("value", factory)
						case "slice-argument":
							child.WithMarshallerPSliceArgument("value", factory)
						case "map-argument":
							child.WithMarshallerPMapArgument("value", factory)
						}
						root.WithCommand(child)
					}
				})
				var err error
				require.NotPanics(t, func() { _, err = root.Create() })
				require.ErrorIs(t, err, codec.ErrInvalid)
				require.ErrorContains(t, err, "value")
				require.ErrorContains(t, err, "factory")
			})
		}
	}
}

func TestMarshallerPointerCallbackCanReplaceInvalidFactory(t *testing.T) {
	var factory func() *pointerFactoryText
	root := NewSimpleApplication().WithName("app")
	require.NotPanics(t, func() {
		root.WithMarshallerPFlag("value", factory, func(f *FlagRequirement[*pointerFactoryText, string]) {
			f.WithCodec(codec.NewMarshallerP(func() *pointerFactoryText { return new(pointerFactoryText) }))
		})
	})
	app, err := root.Create()
	require.NoError(t, err)
	defer func() { require.NoError(t, app.Close()) }()
	invocation, err := app.Parse(nil, nil, "--value=text")
	require.NoError(t, err)
	_, value := invocation.Flag("value")
	assert.IsType(t, (*pointerFactoryText)(nil), value)
	require.NoError(t, invocation.Close())
}
