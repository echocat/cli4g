package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type automaticText struct {
	Text    string
	History []string
}

func (automaticText) MarshalText() ([]byte, error) {
	panic("requirements and help must use input strings, not live MarshalText calls")
}

func (value *automaticText) UnmarshalText(input []byte) error {
	value.Text = string(input)
	value.History = append(value.History, string(input))
	return nil
}

func TestAutomaticMarshallerTagsSupportValuePointerFlagsArgumentsAndDefaults(t *testing.T) {
	type options struct {
		Flag            automaticText  `flag:"text" default:"default-flag"`
		PointerFlag     *automaticText `flag:"pointer" default:"default-pointer"`
		Argument        automaticText  `argument:"argument" default:"default-argument"`
		PointerArgument *automaticText `argument:"pointer-argument" default:"default-pointer-argument"`
	}
	var output bytes.Buffer
	app := NewApplication[options]().WithName("text").WithErrorHandlerEnabled(false).
		WithHelpHandler(testDefaultTemplatedHelpRequirement().WithOutput(&output).MustCreate()).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	require.NoError(t, app.ParseAndExecute(nil, nil, "--help"))
	assert.Contains(t, output.String(), "default-flag")
	assert.Contains(t, output.String(), "default-pointer")
	definition, found := app.Flag("text")
	require.True(t, found)
	formatted, found, err := definition.FormatDefault(context.Background())
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, []string{"default-flag"}, formatted)
	first, err := app.Parse(nil, nil)
	require.NoError(t, err)
	defer func() { assert.NoError(t, first.Close()) }()
	_, defaults := first.Application()
	assert.Equal(t, "default-flag", defaults.Flag.Text)
	assert.Equal(t, "default-pointer", defaults.PointerFlag.Text)
	assert.Equal(t, "default-argument", defaults.Argument.Text)
	assert.Equal(t, "default-pointer-argument", defaults.PointerArgument.Text)
	defaults.Flag.History[0] = "changed"
	defaults.PointerFlag.Text = "changed"
	second, err := app.Parse(nil, nil)
	require.NoError(t, err)
	defer func() { assert.NoError(t, second.Close()) }()
	_, independent := second.Application()
	assert.NotSame(t, defaults.PointerFlag, independent.PointerFlag)
	assert.Equal(t, []string{"default-flag"}, independent.Flag.History)
	assert.Equal(t, "default-pointer", independent.PointerFlag.Text)
	explicit, err := app.Parse(nil, nil, "--text", "flag", "--pointer", "pointer", "argument", "pointer-argument")
	require.NoError(t, err)
	defer func() { assert.NoError(t, explicit.Close()) }()
	_, parsed := explicit.Application()
	assert.Equal(t, "flag", parsed.Flag.Text)
	assert.Equal(t, "pointer", parsed.PointerFlag.Text)
	assert.Equal(t, "argument", parsed.Argument.Text)
	assert.Equal(t, "pointer-argument", parsed.PointerArgument.Text)
}

type automaticHybridText struct{ automaticText }

func (value *automaticHybridText) Set(input string) error { value.Text = "value:" + input; return nil }
func (value automaticHybridText) String() string          { return value.Text }

type overridingTextCodec struct {
	codec.Codec[automaticText, string]
	prefix string
}

func (this overridingTextCodec) ToRaw(ctx context.Context) codec.Codec[any, any] {
	return codec.Raw(ctx, this)
}

func (this overridingTextCodec) Resolve(ctx context.Context, scope codec.Scope, input string) (automaticText, error) {
	return this.Codec.Resolve(ctx, scope, this.prefix+input)
}

func TestAutomaticMarshallerRetainsCodecAndValuePriorities(t *testing.T) {
	type options struct {
		Registered automaticText       `flag:"registered"`
		Named      automaticText       `flag:"named" type:"named"`
		Explicit   automaticText       `flag:"ignored" type:"missing"`
		Hybrid     automaticHybridText `flag:"hybrid"`
	}
	source := func(prefix string) overridingTextCodec {
		return overridingTextCodec{Codec: codec.NewMarshaller[automaticText](), prefix: prefix}
	}
	app := NewApplication[options]().WithName("text").WithErrorHandlerEnabled(false).
		WithCodec(source("registered:")).WithCodec(source("named:"), "named").
		WithFlag(NewFlag(source("explicit:")).WithName("explicit").WithField("Explicit")).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(nil, nil, "--registered", "a", "--named", "b", "--explicit", "c", "--hybrid", "d")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, parsed := invocation.Application()
	assert.Equal(t, "registered:a", parsed.Registered.Text)
	assert.Equal(t, "named:b", parsed.Named.Text)
	assert.Equal(t, "explicit:c", parsed.Explicit.Text)
	assert.Equal(t, "value:d", parsed.Hybrid.Text)
	_, err = NewApplication[struct {
		Text automaticText `flag:"text" type:"missing"`
	}]().WithName("text").Create()
	assert.ErrorContains(t, err, "with name \"missing\"")
}
