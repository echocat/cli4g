package codec_test

import (
	"context"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type contextCompleter struct {
	codec.Codec[string, string]
	seen *codec.CompletionInput
}

func (source contextCompleter) ToRaw(ctx context.Context) codec.Codec[any, any] {
	return codec.Raw(ctx, source)
}

func (source contextCompleter) Complete(ctx context.Context, input codec.CompletionInput) (codec.CompletionResult, error) {
	*source.seen = input
	return codec.CompletionResult{Handled: true, Candidates: []codec.CompletionCandidate{{Value: input.Flags["cluster"][0].Values[0] + "-namespace"}}}, ctx.Err()
}

func TestCompletionRetainsContextThroughRawAndCollections(t *testing.T) {
	var seen codec.CompletionInput
	source := contextCompleter{Codec: codec.NewString(), seen: &seen}
	input := codec.CompletionInput{Prefix: "prod", Target: codec.CompletionTarget{ValueIndex: 3},
		Flags: map[string][]codec.CompletionOccurrence{"cluster": {{Values: []string{"production"}}}}}
	for _, adapted := range []codec.Codec[any, any]{
		codec.Raw(context.Background(), source),
		codec.Raw(context.Background(), codec.NewSlice(source)),
	} {
		result, err := codec.Complete(context.Background(), adapted, input)
		require.NoError(t, err)
		require.Len(t, result.Candidates, 1)
		assert.Equal(t, "production-namespace", result.Candidates[0].Value)
		assert.Equal(t, "production", seen.Flags["cluster"][0].Values[0])
	}
	input.Prefix = "destination=prod"
	result, err := codec.Complete(context.Background(), codec.NewMap[string](source), input)
	require.NoError(t, err)
	assert.Equal(t, "destination=production-namespace", result.Candidates[0].Value)
	assert.Equal(t, "prod", seen.Prefix)
	assert.Equal(t, 3, seen.Target.ValueIndex)
	assert.Zero(t, seen.ValueIndex)
}

type unhandledEnum struct {
	codec.Codec[string, string]
}

func (source unhandledEnum) Complete(context.Context, codec.CompletionInput) (codec.CompletionResult, error) {
	return codec.CompletionResult{}, nil
}

func (source unhandledEnum) EnumVariants(context.Context) ([]codec.EnumVariant[string], error) {
	return []codec.EnumVariant[string]{{Value: "actual-input", Description: "Visible label."}}, nil
}

func (source unhandledEnum) DisplayRequirement(context.Context, string) ([]string, error) {
	return []string{"redacted-display"}, nil
}

func TestUnhandledCompletionFallsBackToActualEnumInputs(t *testing.T) {
	result, err := codec.Complete[string, string](nil, unhandledEnum{codec.NewString()}, codec.CompletionInput{})
	require.NoError(t, err)
	assert.True(t, result.Handled)
	assert.Equal(t, []codec.CompletionCandidate{{Value: "actual-input", Description: "Visible label."}}, result.Candidates)
}
