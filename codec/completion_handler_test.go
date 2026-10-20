package codec_test

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompletionHandlerWrapperPreservesEnumMetadataAndFallback(t *testing.T) {
	source, err := codec.NewEnumVariants(codec.NewString(),
		codec.EnumVariant[string]{Value: "json", Description: "JSON encoding."},
		codec.EnumVariant[string]{Value: "yaml", Description: "YAML encoding."})
	require.NoError(t, err)
	calls := 0
	wrapped := codec.WithCompletionHandler(source, func(context.Context, codec.CompletionInput) (codec.CompletionResult, error) {
		calls++
		return codec.CompletionResult{}, nil
	})
	_, isEnum := wrapped.(codec.Enum[string])
	assert.True(t, isEnum)
	raw, err := codec.Snapshot(context.Background(), wrapped)
	require.NoError(t, err)
	_, isRawEnum := raw.(codec.Enum[any])
	assert.True(t, isRawEnum)
	choices, err := codec.DisplayEnum(context.Background(), raw)
	require.NoError(t, err)
	assert.Equal(t, []codec.EnumChoice{{Values: []string{"json"}, Description: "JSON encoding."}, {Values: []string{"yaml"}, Description: "YAML encoding."}}, choices)
	assert.Zero(t, calls)
	result, err := codec.Complete(nil, raw, codec.CompletionInput{})
	require.NoError(t, err)
	assert.Equal(t, []codec.CompletionCandidate{{Value: "json", Description: "JSON encoding."}, {Value: "yaml", Description: "YAML encoding."}}, result.Candidates)
	assert.Equal(t, 1, calls)
}

type resourceCompletionCodec struct{ codec.Codec[string, *os.File] }

func (source resourceCompletionCodec) ToRaw(ctx context.Context) codec.Codec[any, any] {
	return codec.Raw(ctx, source)
}

func TestCompletionHandlerPreservesRequirementSchemaThroughTypeErasure(t *testing.T) {
	ctx := context.Background()
	wrapped := codec.WithCompletionHandler(resourceCompletionCodec{}.ToRaw(ctx),
		func(context.Context, codec.CompletionInput) (codec.CompletionResult, error) {
			t.Fatal("metadata validation must not invoke completion")
			return codec.CompletionResult{}, nil
		})
	for _, source := range []codec.Codec[any, any]{wrapped, wrapped.ToRaw(ctx)} {
		_, err := codec.Snapshot(ctx, source)
		require.ErrorIs(t, err, codec.ErrResourceRequirement)
	}
}

func TestCompletionHandlerWrapperRetainsTypeBinding(t *testing.T) {
	calls := 0
	registry, err := codec.NewRegistry(context.Background(), []codec.Registration{
		codec.NewRegistration(codec.WithCompletionHandler(codec.NewCombinedDefaults(), func(context.Context, codec.CompletionInput) (codec.CompletionResult, error) {
			calls++
			return codec.CompletionResult{Handled: true, Candidates: []codec.CompletionCandidate{{Value: "42"}}}, nil
		})),
	})
	require.NoError(t, err)
	bound, found, err := registry.Find(context.Background(), reflect.TypeFor[int32](), "")
	require.NoError(t, err)
	require.True(t, found)
	assert.Zero(t, calls)
	value, err := bound.ParseRequirement(context.Background(), codec.UsageFlag, "13")
	require.NoError(t, err)
	assert.Equal(t, int32(13), value)
	result, err := codec.Complete(nil, bound, codec.CompletionInput{})
	require.NoError(t, err)
	assert.Equal(t, []codec.CompletionCandidate{{Value: "42"}}, result.Candidates)
	assert.Equal(t, 1, calls)
}
