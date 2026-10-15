package codec_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type metadataContextCodec struct {
	codec.Codec[string, string]
	seen   *[]context.Context
	cancel context.CancelFunc
}

func (this metadataContextCodec) ToRaw(ctx context.Context) codec.Codec[any, any] {
	*this.seen = append(*this.seen, ctx)
	if this.cancel != nil {
		this.cancel()
	}
	return codec.Raw(ctx, this)
}
func (this metadataContextCodec) CanHandleType(ctx context.Context, typ reflect.Type) (bool, error) {
	*this.seen = append(*this.seen, ctx)
	if this.cancel != nil {
		this.cancel()
	}
	return this.Codec.CanHandleType(ctx, typ)
}
func (this metadataContextCodec) ExpectMinMax(ctx context.Context, usage codec.Usage) (int, int) {
	*this.seen = append(*this.seen, ctx)
	if this.cancel != nil {
		this.cancel()
	}
	return this.Codec.ExpectMinMax(ctx, usage)
}

func TestMetadataContextReachesTypedRawSliceAndRegistry(t *testing.T) {
	type marker struct{}
	ctx := context.WithValue(context.Background(), marker{}, "metadata")
	var seen []context.Context
	source := metadataContextCodec{Codec: codec.NewString(), seen: &seen}
	raw, err := codec.Snapshot(ctx, source)
	require.NoError(t, err)
	registry, err := codec.NewRegistry(ctx, []codec.Registration{{Codec: source}})
	require.NoError(t, err)
	_, found, err := registry.Find(ctx, reflect.TypeFor[string](), "")
	require.NoError(t, err)
	assert.True(t, found)
	_, _ = raw.ExpectMinMax(ctx, codec.UsageFlag)
	matched, err := codec.NewSlice(source).CanHandleType(ctx, reflect.TypeFor[[]string]())
	require.NoError(t, err)
	assert.True(t, matched)
	require.Len(t, seen, 5)
	for _, actual := range seen {
		assert.Same(t, ctx, actual)
		assert.Equal(t, "metadata", actual.Value(marker{}))
	}
}

func TestMetadataCancellationIsPreservedBeforeAndDuringCallbacks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := codec.Snapshot(ctx, codec.NewString())
	require.ErrorIs(t, err, context.Canceled)
	_, err = codec.NewRegistry(ctx, nil)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, codec.NewString().ToRaw(ctx))
	matched, err := codec.NewString().CanHandleType(ctx, reflect.TypeFor[string]())
	_ = matched
	require.ErrorIs(t, err, context.Canceled)
	for _, operation := range []string{"snapshot", "arity", "lookup"} {
		t.Run(operation, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var seen []context.Context
			source := metadataContextCodec{Codec: codec.NewString(), seen: &seen, cancel: cancel}
			switch operation {
			case "snapshot":
				_, err = codec.Snapshot(ctx, source)
			case "arity":
				err = codec.EnsureArgumentsMatching(ctx, codec.UsageFlag, source, "value")
			case "lookup":
				registry, buildErr := codec.NewRegistry(context.Background(), []codec.Registration{{Codec: codec.Raw(context.Background(), source)}})
				require.NoError(t, buildErr)
				// CanHandleType cancels during lookup, through the raw adapter.
				_, _, err = registry.Find(ctx, reflect.TypeFor[string](), "")
			}
			require.ErrorIs(t, err, context.Canceled)
		})
	}
}
