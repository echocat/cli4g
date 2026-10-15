package codec_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type enumLevel string

func (value enumLevel) MarshalText() ([]byte, error) {
	if value == "broken" {
		return nil, io.ErrUnexpectedEOF
	}
	return []byte(value), nil
}
func (value *enumLevel) UnmarshalText(input []byte) error {
	*value = enumLevel(strings.ToLower(string(input)))
	return nil
}

type enumValueLevel string

func (value *enumValueLevel) String() string { return string(*value) }
func (value *enumValueLevel) Set(input string) error {
	*value = enumValueLevel(strings.ToLower(input))
	return nil
}

func TestTypedEnumAdaptersRestrictTokensBeforeDecodingAndExposeInputRequirements(t *testing.T) {
	ctx := context.Background()
	text, err := codec.NewMarshallerEnum(enumLevel("info"), enumLevel("debug"))
	require.NoError(t, err)
	value, err := codec.NewValueEnum(enumValueLevel("info"), enumValueLevel("debug"))
	require.NoError(t, err)
	for _, source := range []codec.Codec[any, any]{text.ToRaw(ctx), value.ToRaw(ctx)} {
		_, err := source.ParseRequirement(ctx, codec.UsageFlag, "INFO")
		assert.ErrorIs(t, err, codec.ErrValidation, "base adapters accept case-insensitive text, but only declared enum tokens are allowed")
		_, err = source.ParseRequirement(ctx, codec.UsageDefault, "trace")
		assert.ErrorContains(t, err, `expected one of: "info", "debug"`)
		requirement, err := source.ParseRequirement(ctx, codec.UsageFlag, "debug")
		require.NoError(t, err)
		plain, err := source.FormatRequirement(ctx, requirement)
		require.NoError(t, err)
		assert.Equal(t, []string{"debug"}, plain)
		resolved, err := source.Resolve(ctx, new(codec.ResourceScope), requirement)
		require.NoError(t, err)
		assert.Contains(t, []any{enumLevel("debug"), enumValueLevel("debug")}, resolved)
		choices, err := codec.DisplayEnum(ctx, source)
		require.NoError(t, err)
		assert.Equal(t, []codec.EnumChoice{{Values: []string{"info"}}, {Values: []string{"debug"}}}, choices)
	}
	_, err = codec.NewMarshallerEnum(enumLevel("info"), enumLevel("info"))
	assert.ErrorIs(t, err, codec.ErrInvalid)
	_, err = codec.NewMarshallerEnum(enumLevel("broken"))
	assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
	_, err = codec.NewValueEnum(enumValueLevel("info"), enumValueLevel("info"))
	assert.ErrorIs(t, err, codec.ErrInvalid)
	described, err := codec.NewValueEnumVariants(codec.EnumVariant[enumValueLevel]{Value: "info", Description: "Normal output."})
	require.NoError(t, err)
	choices, err := codec.DisplayEnum(ctx, described)
	require.NoError(t, err)
	assert.Equal(t, []codec.EnumChoice{{Values: []string{"info"}, Description: "Normal output."}}, choices)
}

func TestTypedPointerEnumFactoriesAreDeferredAndNeverReuseDeclaredValues(t *testing.T) {
	ctx := context.Background()
	textValue, valueValue := enumLevel("info"), enumValueLevel("info")
	textFactories, valueFactories := 0, 0
	text, err := codec.NewMarshallerPEnum(func() *enumLevel { textFactories++; return new(enumLevel) }, &textValue)
	require.NoError(t, err)
	value, err := codec.NewValuePEnum(func() *enumValueLevel { valueFactories++; return new(enumValueLevel) }, &valueValue)
	require.NoError(t, err)
	for _, source := range []codec.Codec[any, any]{text.ToRaw(ctx), value.ToRaw(ctx)} {
		_, err := codec.DisplayEnum(ctx, source)
		require.NoError(t, err)
	}
	assert.Zero(t, textFactories)
	assert.Zero(t, valueFactories)
	textValue, valueValue = "mutated", "mutated"
	textResult, err := text.Resolve(ctx, new(codec.ResourceScope), "info")
	require.NoError(t, err)
	assert.Equal(t, enumLevel("info"), *textResult)
	assert.NotSame(t, &textValue, textResult)
	valueResult, err := value.Resolve(ctx, new(codec.ResourceScope), []string{"info"})
	require.NoError(t, err)
	assert.Equal(t, enumValueLevel("info"), *valueResult)
	assert.NotSame(t, &valueValue, valueResult)
	assert.Equal(t, 1, textFactories)
	assert.Equal(t, 1, valueFactories)
	_, err = codec.NewMarshallerPEnum[*enumLevel](nil, &textValue)
	assert.ErrorIs(t, err, codec.ErrInvalid)
	_, err = codec.NewMarshallerPEnum(func() *enumLevel { return new(enumLevel) }, (*enumLevel)(nil))
	assert.ErrorIs(t, err, codec.ErrInvalid)
	_, err = codec.NewValuePEnum[*enumValueLevel](nil, &valueValue)
	assert.ErrorIs(t, err, codec.ErrInvalid)
	_, err = codec.NewValuePEnum(func() *enumValueLevel { return new(enumValueLevel) }, (*enumValueLevel)(nil))
	assert.ErrorIs(t, err, codec.ErrInvalid)
}
