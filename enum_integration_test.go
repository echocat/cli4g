package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type enumFormat uint8

const (
	enumJSON enumFormat = iota
	enumYAML
)

func (value enumFormat) MarshalText() ([]byte, error) {
	if value == enumJSON {
		return []byte("json"), nil
	}
	return []byte("yaml"), nil
}
func (value *enumFormat) UnmarshalText(input []byte) error {
	if string(input) == "json" {
		*value = enumJSON
	} else {
		*value = enumYAML
	}
	return nil
}

type enumLevel string

func (value *enumLevel) Set(input string) error { *value = enumLevel(input); return nil }
func (value *enumLevel) String() string         { return string(*value) }

type taggedEnumMode string

func TestEnumsIntegrateWithTagsTypedAdaptersCollectionsAndHelp(t *testing.T) {
	type options struct {
		Mode     taggedEnumMode        `flag:"mode" enum:"auto|always|never" default:"auto" env:"ENUM_MODE"`
		Format   enumFormat            `flag:"format" default:"json"`
		Formats  []enumFormat          `flag:"formats"`
		Mappings map[string]enumFormat `flag:"mapping"`
		Level    enumLevel             `flag:"level" default:"info"`
		Side     string                `argument:"side" enum:"left|right" default:"left"`
	}
	text, err := codec.NewMarshallerEnum(enumJSON, enumYAML)
	require.NoError(t, err)
	described, err := codec.NewMarshallerEnumVariants(
		codec.EnumVariant[enumFormat]{Value: enumJSON, Description: "JSON encoding."},
		codec.EnumVariant[enumFormat]{Value: enumYAML, Description: "YAML encoding."})
	require.NoError(t, err)
	levels, err := codec.NewValueEnum(enumLevel("info"), enumLevel("debug"))
	require.NoError(t, err)
	var output bytes.Buffer
	app := NewApplication[options]("enums").
		WithErrorHandlerEnabled(false).
		WithCodec(described).
		WithCodec(codec.NewSlice(text)).
		WithCodec(codec.NewMap[string](text)).
		WithCodec(levels).
		WithHelpHandler(testDefaultTemplatedHelpRequirement().
			WithOutput(&output).
			MustCreate(),
		).
		MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	ctx := context.Background()
	require.NoError(t, app.ParseAndExecute(ctx, nil, "--help"))
	assert.Contains(t, strings.Join(strings.Fields(output.String()), " "), "choices: auto, always, never")
	assert.Contains(t, output.String(), "choices: left, right")
	assert.Contains(t, output.String(), "choices: json, yaml")
	assert.Contains(t, output.String(), "JSON encoding.")
	assert.Contains(t, output.String(), "YAML encoding.")
	assert.Contains(t, output.String(), "default: auto")
	lookup := &ParseOpts{LookupEnv: func(context.Context, string) (string, bool) { return "never", true }}
	invocation, err := app.Parse(ctx, lookup, "--format", "yaml", "--formats", "json", "--formats", "yaml", "--mapping", "a=json", "--mapping", "b=yaml", "--level", "debug", "right")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, parsed := invocation.Application()
	assert.Equal(t, taggedEnumMode("never"), parsed.Mode)
	assert.Equal(t, enumYAML, parsed.Format)
	assert.Equal(t, []enumFormat{enumJSON, enumYAML}, parsed.Formats)
	assert.Equal(t, map[string]enumFormat{"a": enumJSON, "b": enumYAML}, parsed.Mappings)
	assert.Equal(t, enumLevel("debug"), parsed.Level)
	assert.Equal(t, "right", parsed.Side)
	for _, args := range [][]string{
		{"--mode", "sometimes"}, {"--format", "YAML"}, {"--formats", "toml"},
		{"--mapping", "a=toml"}, {"--level", "trace"}, {"center"},
	} {
		_, err := app.Parse(ctx, nil, args...)
		assert.ErrorIs(t, err, ErrValidation)
		assert.ErrorContains(t, err, "expected one of:")
	}
	defaults, err := app.Parse(ctx, nil)
	require.NoError(t, err)
	defer func() { assert.NoError(t, defaults.Close()) }()
	_, parsed = defaults.Application()
	assert.Equal(t, taggedEnumMode("auto"), parsed.Mode)
	assert.Equal(t, enumJSON, parsed.Format)
	assert.Equal(t, enumLevel("info"), parsed.Level)
}

type customEnumMetadata struct {
	codec.Codec[string, string]
	variants []codec.EnumVariant[string]
	failure  error
	resolves *int
}

func (this customEnumMetadata) ToRaw(ctx context.Context) codec.Codec[any, any] {
	return codec.Raw(ctx, this)
}
func (this customEnumMetadata) EnumVariants(ctx context.Context) ([]codec.EnumVariant[string], error) {
	return append([]codec.EnumVariant[string](nil), this.variants...), errors.Join(this.failure, ctx.Err())
}
func (this customEnumMetadata) Resolve(ctx context.Context, scope codec.Scope, value string) (string, error) {
	*this.resolves++
	return this.Codec.Resolve(ctx, scope, value)
}

func TestThirdPartyEnumHelpMetadataIsPreservedAndNeverResolves(t *testing.T) {
	resolves := 0
	source := customEnumMetadata{Codec: codec.NewString(), resolves: &resolves,
		variants: []codec.EnumVariant[string]{{Value: "one", Description: "First choice."}, {Value: "two"}}}
	var output bytes.Buffer
	app := NewApplication[struct {
		Choice string `flag:"choice" type:"custom"`
	}]("app").WithCodec(source, "custom").
		WithHelpHandler(testDefaultTemplatedHelpRequirement().
			WithOutput(&output).
			MustCreate(),
		).
		WithCommand(NewSimpleCommand("child")).
		MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	require.NoError(t, app.ParseAndExecute(nil, nil, "child", "--help"))
	assert.Contains(t, output.String(), "Inherited flags:")
	assert.Contains(t, output.String(), "First choice.")
	assert.Zero(t, resolves)
	definition, found := app.Flag("choice")
	require.True(t, found)
	_, isEnum := definition.Codec().(codec.Enum[any])
	assert.True(t, isEnum)
	metadataFailure := errors.New("metadata unavailable")
	for _, invalid := range []customEnumMetadata{
		{Codec: codec.NewString()},
		{Codec: codec.NewString(), variants: []codec.EnumVariant[string]{{Value: "same"}, {Value: "same"}}},
		{Codec: codec.NewString(), failure: metadataFailure},
	} {
		_, err := NewSimpleApplication("app").
			WithCodec(invalid).
			Create()
		require.Error(t, err)
		if invalid.failure != nil {
			assert.ErrorIs(t, err, metadataFailure)
		}
	}
}

func TestEnumTagsRejectMisconfigurationAndExplicitBindingWins(t *testing.T) {
	for _, build := range []func() error{
		func() error {
			_, err := NewApplication[struct {
				Value int `flag:"value" enum:"a|b"`
			}]("app").Create()
			return err
		},
		func() error {
			_, err := NewApplication[struct {
				Value string `flag:"value" enum:"a|a"`
			}]("app").Create()
			return err
		},
		func() error {
			_, err := NewApplication[struct {
				Value string `flag:"value" enum:"a|b" type:"named"`
			}]("app").Create()
			return err
		},
	} {
		err := build()
		require.Error(t, err)
		assert.NotErrorIs(t, err, ErrValidation)
	}
	type options struct {
		Value string `flag:"old" enum:"a|a"`
	}
	app := NewApplication[options]("app").
		WithStringFlag("value", func(f *FlagRequirement[string, string]) {
			f.WithField("Value")
		}).
		MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(nil, nil, "--value", "unrestricted")
	require.NoError(t, err)
	require.NoError(t, invocation.Close())
	_, parsed := invocation.Application()
	assert.Equal(t, "unrestricted", parsed.Value)
}
