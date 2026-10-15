package codec_test

import (
	"context"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnumValidatesEveryRequirementPathAndPublishesDescriptions(t *testing.T) {
	ctx := context.Background()
	source, err := codec.NewEnumVariants(codec.NewString(),
		codec.EnumVariant[string]{Value: "info", Description: "Normal output."},
		codec.EnumVariant[string]{Value: "debug", Description: "Detailed output."})
	require.NoError(t, err)
	choices, err := codec.DisplayEnum(ctx, source)
	require.NoError(t, err)
	assert.Equal(t, []codec.EnumChoice{
		{Values: []string{"info"}, Description: "Normal output."},
		{Values: []string{"debug"}, Description: "Detailed output."},
	}, choices)
	assert.False(t, codec.IsCumulative(source))
	for _, usage := range []codec.Usage{codec.UsageFlag, codec.UsageArgument, codec.UsageDefault} {
		value, err := source.ParseRequirement(ctx, usage, "debug")
		require.NoError(t, err)
		assert.Equal(t, "debug", value)
		_, err = source.ParseRequirement(ctx, usage, "trace")
		require.ErrorIs(t, err, codec.ErrIllegalArgument)
		assert.ErrorContains(t, err, `expected one of: "info", "debug"`)
	}
	_, err = source.FormatRequirement(ctx, "trace")
	assert.ErrorIs(t, err, codec.ErrValidation)
	_, err = source.DisplayRequirement(ctx, "trace")
	assert.ErrorIs(t, err, codec.ErrValidation)
	_, err = codec.CloneRequirement(ctx, source, "trace")
	assert.ErrorIs(t, err, codec.ErrValidation)
	_, err = source.Resolve(ctx, new(codec.ResourceScope), "trace")
	assert.ErrorIs(t, err, codec.ErrValidation)
	value, err := source.Resolve(ctx, new(codec.ResourceScope), "info")
	require.NoError(t, err)
	assert.Equal(t, "info", value)
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	_, err = source.ParseRequirement(canceled, codec.UsageFlag, "info")
	assert.ErrorIs(t, err, context.Canceled)
	_, err = source.Resolve(canceled, new(codec.ResourceScope), "info")
	assert.ErrorIs(t, err, context.Canceled)
	_, err = codec.DisplayEnum(canceled, source)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestEnumSnapshotsMutableRequirementsAndKeepsTokenBoundaries(t *testing.T) {
	ctx := context.Background()
	first, second := []string{"a", "bc"}, []string{"ab", "c"}
	source, err := codec.NewEnum(codec.NewStrings(), first, second, []string{}, nil)
	require.NoError(t, err)
	first[0] = "mutated"
	variants, err := source.(codec.Enum[[]string]).EnumVariants(ctx)
	require.NoError(t, err)
	assert.Equal(t, []string{"a", "bc"}, variants[0].Value)
	variants[0].Value[0] = "mutated again"
	for _, input := range [][]string{{"a", "bc"}, {"ab", "c"}, {}, nil} {
		value, err := source.ParseRequirement(ctx, codec.UsageDefault, input...)
		require.NoError(t, err)
		assert.Equal(t, input, value)
		plain, err := source.FormatRequirement(ctx, value)
		require.NoError(t, err)
		copy, err := source.ParseRequirement(ctx, codec.UsageDefault, plain...)
		require.NoError(t, err)
		assert.Equal(t, value, copy)
	}
	_, err = source.ParseRequirement(ctx, codec.UsageDefault, "abc")
	assert.ErrorIs(t, err, codec.ErrValidation)
	opaque, err := codec.NewEnum(codec.NewString(), "opaque-\xff", "opaque-\xfe", "")
	require.NoError(t, err)
	for _, input := range []string{"opaque-\xff", "opaque-\xfe", ""} {
		value, err := opaque.ParseRequirement(ctx, codec.UsageFlag, input)
		require.NoError(t, err)
		assert.Equal(t, input, value)
	}
}

func TestEnumConfigurationAndUnderlyingNormalization(t *testing.T) {
	_, err := codec.NewEnum[string, string](nil, "a")
	assert.ErrorIs(t, err, codec.ErrInvalid)
	_, err = codec.NewEnum(codec.NewString())
	assert.ErrorIs(t, err, codec.ErrInvalid)
	_, err = codec.NewEnum(codec.NewString(), "a", "a")
	assert.ErrorIs(t, err, codec.ErrInvalid)
	source, err := codec.NewEnum(codec.NewInt(), 1, 2)
	require.NoError(t, err)
	value, err := source.ParseRequirement(context.Background(), codec.UsageFlag, "0x2")
	require.NoError(t, err)
	assert.Equal(t, 2, value, "the generic wrapper preserves underlying parse rules")
	boolean, err := codec.NewEnum(codec.NewBool(), true, false)
	require.NoError(t, err)
	assert.True(t, boolean.(interface{ IsBoolFlag() bool }).IsBoolFlag())
	truth, err := boolean.ParseRequirement(context.Background(), codec.UsageFlag)
	require.NoError(t, err)
	assert.True(t, truth)
}

// A third-party codec only implements the optional public Enum contract.
type externalEnumCodec struct{ codec.Codec[string, string] }

func (this externalEnumCodec) ToRaw(ctx context.Context) codec.Codec[any, any] {
	return codec.Raw(ctx, this)
}
func (externalEnumCodec) EnumVariants(ctx context.Context) ([]codec.EnumVariant[string], error) {
	return []codec.EnumVariant[string]{{Value: "a", Description: "First."}, {Value: "b"}}, ctx.Err()
}
func (this externalEnumCodec) ParseRequirement(ctx context.Context, usage codec.Usage, args ...string) (string, error) {
	value, err := this.Codec.ParseRequirement(ctx, usage, args...)
	if err == nil && value != "a" && value != "b" {
		err = codec.NewValidationErrorf("unknown letter")
	}
	return value, err
}

func TestCustomEnumMetadataSurvivesRawAndCollectionAdapters(t *testing.T) {
	ctx := context.Background()
	source := externalEnumCodec{Codec: codec.NewString()}
	raw := source.ToRaw(ctx)
	_, isEnum := raw.(codec.Enum[any])
	require.True(t, isEnum)
	_, isEnum = raw.ToRaw(ctx).(codec.Enum[any])
	assert.True(t, isEnum)
	variants, err := raw.(codec.Enum[any]).EnumVariants(ctx)
	require.NoError(t, err)
	assert.Equal(t, "a", variants[0].Value)
	for _, adapted := range []codec.Codec[any, any]{
		raw,
		codec.NewSlice(source).ToRaw(ctx),
		codec.NewMap[string](source).ToRaw(ctx),
		codec.NewSlice(codec.NewMap[string](source)).ToRaw(ctx),
	} {
		choices, err := codec.DisplayEnum(ctx, adapted)
		require.NoError(t, err)
		assert.Equal(t, []codec.EnumChoice{{Values: []string{"a"}, Description: "First."}, {Values: []string{"b"}}}, choices)
		choices[0].Values[0] = "changed"
		copy, err := codec.DisplayEnum(ctx, adapted)
		require.NoError(t, err)
		assert.Equal(t, "a", copy[0].Values[0])
	}
	for _, normal := range []codec.Codec[any, any]{codec.NewString().ToRaw(ctx), codec.NewSlice(source).ToRaw(ctx), codec.NewMap[string](source).ToRaw(ctx)} {
		_, isEnum := normal.(codec.Enum[any])
		assert.False(t, isEnum, "ordinary codecs and collections must not claim a finite requirement set")
	}
	choices, err := codec.DisplayEnum(ctx, codec.NewStrings())
	require.NoError(t, err)
	assert.Nil(t, choices)
}

type enumMetadataProbe struct {
	codec.Codec[any, any]
	variant any
	cancel  context.CancelFunc
}

func (this enumMetadataProbe) EnumVariants(context.Context) ([]codec.EnumVariant[any], error) {
	if this.cancel != nil {
		this.cancel()
		return nil, nil
	}
	return []codec.EnumVariant[any]{{Value: this.variant}}, nil
}

func TestEnumMetadataRejectsResourcesAndStopsAfterCancellation(t *testing.T) {
	ctx := context.Background()
	source := enumMetadataProbe{Codec: codec.NewString().ToRaw(ctx), variant: func() {}}
	_, err := codec.DisplayEnum(ctx, source)
	assert.ErrorIs(t, err, codec.ErrResourceRequirement)
	canceled, cancel := context.WithCancel(ctx)
	defer cancel()
	source = enumMetadataProbe{Codec: codec.NewString().ToRaw(ctx), cancel: cancel}
	_, err = codec.DisplayEnum(canceled, source)
	assert.ErrorIs(t, err, context.Canceled)
	assert.NotErrorIs(t, err, codec.ErrInvalid)
}
