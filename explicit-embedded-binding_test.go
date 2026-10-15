package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/echocat/cli4g/codec"
)

type ExplicitEmbeddedValue struct {
	Text string `flag:"internal" default:"tag-default"`
}

func (value *ExplicitEmbeddedValue) String() string         { return value.Text }
func (value *ExplicitEmbeddedValue) Set(input string) error { value.Text = input; return nil }

func TestExplicitEmbeddedLeafBindingOverridesOnlyItsTag(t *testing.T) {
	type options struct {
		*ExplicitEmbeddedValue
		Other string `flag:"other" default:"other-default"`
	}
	app, err := NewApplication[*options]().WithName("app").WithHelp(false).WithErrorHandlerEnabled(false).
		WithStringFlag("external", func(f *FlagRequirement[string, string]) { f.WithField("Text") }).Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, app.Close()) }()
	_, found := app.Flag("internal")
	assert.False(t, found)
	invocation, err := app.Parse(nil, nil, "--external=cli")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, parsed := invocation.Application()
	require.NotNil(t, parsed.ExplicitEmbeddedValue)
	assert.Equal(t, "cli", parsed.Text)
	assert.Equal(t, "other-default", parsed.Other)
}

func TestExplicitWholeEmbeddedBindingDoesNotCollectDescendantTags(t *testing.T) {
	for _, kind := range []string{"flag", "argument"} {
		t.Run(kind, func(t *testing.T) {
			type options struct {
				*ExplicitEmbeddedValue
				Other string `flag:"other" default:"other-default"`
			}
			decoder, err := codec.NewValueP(func() *ExplicitEmbeddedValue { return new(ExplicitEmbeddedValue) })
			require.NoError(t, err)
			root := NewApplication[*options]().WithName("app").WithHelp(false).WithErrorHandlerEnabled(false)
			var args []string
			if kind == "flag" {
				root.WithFlag(NewFlag(decoder).WithName("whole").WithField("ExplicitEmbeddedValue"))
				args = []string{"--whole=cli"}
			} else {
				root.WithArgument(NewArgument(decoder).WithName("whole").WithField("ExplicitEmbeddedValue"))
				args = []string{"cli"}
			}
			app, err := root.Create()
			require.NoError(t, err)
			defer func() { assert.NoError(t, app.Close()) }()
			_, found := app.Flag("internal")
			assert.False(t, found, "the codec owns the whole explicitly bound field")
			invocation, err := app.Parse(nil, nil, args...)
			require.NoError(t, err)
			defer func() { assert.NoError(t, invocation.Close()) }()
			_, parsed := invocation.Application()
			require.NotNil(t, parsed.ExplicitEmbeddedValue)
			assert.Equal(t, "cli", parsed.Text, "descendant defaults must not overwrite a codec's parsed value")
			assert.Equal(t, "other-default", parsed.Other)
		})
	}
}

type RecursiveBoundValue struct {
	*RecursiveBoundValue
	Text string `flag:"internal"`
}

func (value *RecursiveBoundValue) String() string         { return value.Text }
func (value *RecursiveBoundValue) Set(input string) error { value.Text = input; return nil }

func TestExplicitRecursiveValueBindingIsOpaqueToTagCollection(t *testing.T) {
	type options struct{ *RecursiveBoundValue }
	decoder, err := codec.NewValueP(func() *RecursiveBoundValue { return new(RecursiveBoundValue) })
	require.NoError(t, err)
	app, err := NewApplication[*options]().WithName("app").WithHelp(false).WithErrorHandlerEnabled(false).
		WithFlag(NewFlag(decoder).WithName("whole").WithField("RecursiveBoundValue")).Create()
	require.NoError(t, err, "tag discovery must not recurse into an explicitly bound codec value")
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(nil, nil, "--whole=cli")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, parsed := invocation.Application()
	require.NotNil(t, parsed.RecursiveBoundValue)
	assert.Equal(t, "cli", parsed.Text)
}
