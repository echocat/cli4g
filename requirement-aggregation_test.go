package cli

import (
	"context"
	"reflect"
	"slices"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type packedListRequirement struct{ Tokens []string }
type packedListCodec struct{}

func (c packedListCodec) ToRaw(ctx context.Context) codec.Codec[any, any] { return codec.Raw(ctx, c) }
func (packedListCodec) CanHandleType(ctx context.Context, typ reflect.Type) (bool, error) {
	return typ == reflect.TypeFor[[]string](), ctx.Err()
}
func (packedListCodec) ExpectMinMax(ctx context.Context, usage codec.Usage) (int, int) {
	return codec.NewStringSlice().ExpectMinMax(ctx, usage)
}
func (packedListCodec) ParseRequirement(ctx context.Context, usage codec.Usage, args ...string) (packedListRequirement, error) {
	tokens, err := codec.NewStringSlice().ParseRequirement(ctx, usage, args...)
	return packedListRequirement{tokens}, err
}
func (packedListCodec) FormatRequirement(ctx context.Context, value packedListRequirement) ([]string, error) {
	return codec.NewStringSlice().FormatRequirement(ctx, value.Tokens)
}
func (c packedListCodec) DisplayRequirement(ctx context.Context, value packedListRequirement) ([]string, error) {
	return c.FormatRequirement(ctx, value)
}
func (packedListCodec) Resolve(ctx context.Context, _ codec.Scope, value packedListRequirement) ([]string, error) {
	return slices.Clone(value.Tokens), ctx.Err()
}

type cumulativePackedListCodec struct{ packedListCodec }

func (c cumulativePackedListCodec) ToRaw(ctx context.Context) codec.Codec[any, any] {
	return codec.Raw(ctx, c)
}
func (cumulativePackedListCodec) IsCumulative() bool { return true }
func (c cumulativePackedListCodec) ParseRequirementInto(ctx context.Context, usage codec.Usage, previous packedListRequirement, args ...string) (packedListRequirement, error) {
	additional, err := c.ParseRequirement(ctx, usage, args...)
	if err != nil {
		return previous, err
	}
	return packedListRequirement{append(previous.Tokens, additional.Tokens...)}, nil
}

func TestRepeatedFlagsUseCodecAggregationRatherThanRuntimeSliceType(t *testing.T) {
	plain := NewSimpleApplication("app").
		WithErrorHandlerEnabled(false).
		WithFlag(NewFlag(packedListCodec{}, "item")).
		MustCreate()
	defer func() { assert.NoError(t, plain.Close()) }()
	invocation, err := plain.Parse(context.Background(), nil, "--item=one")
	require.NoError(t, err)
	_, value := invocation.Flag("item")
	assert.Equal(t, []string{"one"}, value)
	require.NoError(t, invocation.Close())
	_, err = plain.Parse(context.Background(), nil, "--item=one", "--item=two")
	require.ErrorIs(t, err, ErrValidation)
	assert.ErrorContains(t, err, "supplied more than once")

	aggregated := NewSimpleApplication("app").
		WithErrorHandlerEnabled(false).
		WithFlag(NewFlag(cumulativePackedListCodec{}, "item")).
		MustCreate()
	defer func() { assert.NoError(t, aggregated.Close()) }()
	for range 2 {
		invocation, err := aggregated.Parse(context.Background(), nil, "--item=one", "--item=two")
		require.NoError(t, err)
		_, value := invocation.Flag("item")
		assert.Equal(t, []string{"one", "two"}, value)
		require.NoError(t, invocation.Close())
	}
}
