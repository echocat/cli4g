package cli

import (
	"context"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/require"
)

type canceledArityCodec struct {
	codec.Codec[any, any]
	cancel context.CancelFunc
}

func (this canceledArityCodec) ExpectMinMax(context.Context, codec.Usage) (int, int) {
	this.cancel()
	return -1, -2 // cancellation wins over invalid metadata returned afterwards
}
func TestCompileArityPreservesCancellationDuringMetadata(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	source := canceledArityCodec{Codec: codec.NewString().ToRaw(context.Background()), cancel: cancel}
	_, err := compileValueArity(ctx, source, codec.UsageFlag)
	require.ErrorIs(t, err, context.Canceled)
}
