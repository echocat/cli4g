package codec_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompletionDecoratorPreservesCumulativeParsingAndIndependentRequirements(t *testing.T) {
	calls := 0
	wrapped := codec.WithCompletionHandler(codec.NewStrings(), func(context.Context, codec.CompletionInput) (codec.CompletionResult, error) {
		calls++
		return codec.CompletionResult{}, nil
	})
	ctx := context.Background()
	require.True(t, codec.IsCumulative(wrapped))
	first, err := wrapped.ParseRequirement(ctx, codec.UsageFlag, "first")
	require.NoError(t, err)
	values, err := wrapped.(codec.Cumulative[[]string]).ParseRequirementInto(ctx, codec.UsageFlag, first, "second")
	require.NoError(t, err)
	assert.Equal(t, []string{"first", "second"}, values)
	cloned, err := codec.CloneRequirement(ctx, wrapped, values)
	require.NoError(t, err)
	cloned[0] = "changed"
	assert.Equal(t, []string{"first", "second"}, values)
	assert.Zero(t, calls, "normal parsing and cloning do not invoke completion")
	raw := wrapped.ToRaw(ctx)
	require.True(t, codec.IsCumulative(raw))
	previous, err := raw.ParseRequirement(ctx, codec.UsageFlag, "third")
	require.NoError(t, err)
	combined, err := raw.(codec.Cumulative[any]).ParseRequirementInto(ctx, codec.UsageFlag, previous, "fourth")
	require.NoError(t, err)
	assert.Equal(t, []string{"third", "fourth"}, combined)
	assert.Zero(t, calls)
}

func TestCompletionDecoratorPreservesErasedBoolAndPlainStringBinding(t *testing.T) {
	ctx := context.Background()
	invoked := 0
	handler := func(context.Context, codec.CompletionInput) (codec.CompletionResult, error) {
		invoked++
		return codec.CompletionResult{Handled: true}, nil
	}
	boolean, err := codec.Snapshot(ctx, codec.WithCompletionHandler(codec.NewBool().ToRaw(ctx), handler))
	require.NoError(t, err)
	require.True(t, boolean.(interface{ IsBoolFlag() bool }).IsBoolFlag())
	value, err := boolean.ParseRequirement(ctx, codec.UsageFlag)
	require.NoError(t, err)
	assert.Equal(t, true, value)
	registry, err := codec.NewRegistry(ctx, []codec.Registration{codec.NewRegistration(codec.WithCompletionHandler(codec.NewString(), handler))})
	require.NoError(t, err)
	bound, found, err := registry.Find(ctx, reflect.TypeFor[string](), "")
	require.NoError(t, err)
	require.True(t, found)
	value, err = bound.ParseRequirement(ctx, codec.UsageFlag, "normal value")
	require.NoError(t, err)
	assert.Equal(t, "normal value", value)
	_, found, err = registry.Find(ctx, reflect.TypeFor[int](), "")
	require.NoError(t, err)
	assert.False(t, found)
	assert.Zero(t, invoked)
}

func TestCompletionDecoratorHonorsCancellationAndOwnsReturnedCandidates(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	wrapped := codec.WithCompletionHandler(codec.NewString(), func(context.Context, codec.CompletionInput) (codec.CompletionResult, error) {
		cancel()
		return codec.CompletionResult{Handled: true, Candidates: []codec.CompletionCandidate{{Value: "stale"}}}, nil
	})
	result, err := codec.Complete(ctx, wrapped, codec.CompletionInput{})
	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, result.Candidates)
	shared := []codec.CompletionCandidate{{Value: "original"}}
	wrapped = codec.WithCompletionHandler(codec.NewString(), func(context.Context, codec.CompletionInput) (codec.CompletionResult, error) {
		return codec.CompletionResult{Handled: true, Candidates: shared}, nil
	})
	result, err = codec.Complete(nil, wrapped, codec.CompletionInput{})
	require.NoError(t, err)
	result.Candidates[0].Value = "changed"
	assert.Equal(t, "original", shared[0].Value)
}
