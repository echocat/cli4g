package cli

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/echocat/cli4g/codec"
)

func TestInvocationAliasesCloseOnlyOnce(t *testing.T) {
	closeErr := errors.New("cleanup failed")
	var calls atomic.Int64
	app, err := NewSimpleApplication("app").
		WithCloseHandler(func(NoOptions) error {
			calls.Add(1)
			return closeErr
		}).
		Create()
	require.NoError(t, err)
	ctx := context.Background()
	invocation, err := app.Parse(ctx, nil)
	require.NoError(t, err)
	copy := *invocation
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			assert.ErrorIs(t, copy.Close(), closeErr)
		})
	}
	wg.Wait()
	assert.Equal(t, int64(1), calls.Load())
	assert.ErrorIs(t, invocation.Close(), closeErr)
	assert.ErrorIs(t, invocation.Execute(ctx), ErrInvocationClosed)
}

func TestZeroInvocationLifecycle(t *testing.T) {
	var invocation Invocation[NoOptions]
	assert.NoError(t, invocation.Close())
	assert.ErrorIs(t, invocation.Execute(context.Background()), ErrInvocationNotReady)
	var nilInvocation *Invocation[NoOptions]
	assert.NoError(t, nilInvocation.Close())
	assert.ErrorIs(t, nilInvocation.Execute(context.Background()), ErrInvocationNotReady)
}

func TestInvocationRetainsResourcesUntilClose(t *testing.T) {
	for _, execute := range []bool{false, true} {
		name := "without execution"
		if execute {
			name = "failed execution"
		}
		t.Run(name, func(t *testing.T) {
			type contextKey struct{}
			executeErr := errors.New("execution failed")
			resourceCloseErr := errors.New("resource close failed")
			handlerCloseErr := errors.New("handler close failed")
			var events []string
			resource := &ownedTestResource{events: &events, closeErr: resourceCloseErr}
			resourceCodec := codec.NewMarshallerP(func() *ownedTestResource { return resource })
			app, err := NewApplication[resourceOptions]("app").
				WithCodec(resourceCodec).
				WithHandler(func(context.Context, resourceOptions) error {
					return executeErr
				}).
				WithCloseHandler(func(options resourceOptions) error {
					assert.Equal(t, "initialize", options.ctx.Value(contextKey{}))
					assert.ErrorIs(t, options.ctx.Err(), context.Canceled)
					events = append(events, "close container root")
					return handlerCloseErr
				}).
				Create()
			require.NoError(t, err)
			ctx, cancel := context.WithCancel(context.WithValue(context.Background(), contextKey{}, "initialize"))
			defer cancel()
			invocation, err := app.Parse(ctx, nil, "--resource=root")
			require.NoError(t, err)
			if execute {
				assert.ErrorIs(t, invocation.Execute(ctx), executeErr)
			}
			assert.Zero(t, resource.closes)
			cancel()
			closeErr := invocation.Close()
			assert.ErrorIs(t, closeErr, handlerCloseErr)
			assert.ErrorIs(t, closeErr, resourceCloseErr)
			assert.Equal(t, []string{
				"validate options root", "initialize options root",
				"close container root", "close options root",
			}, events)
			assert.Equal(t, 1, resource.closes)
			assert.Same(t, closeErr, invocation.Close())
			assert.Equal(t, 1, resource.closes)
		})
	}
}
