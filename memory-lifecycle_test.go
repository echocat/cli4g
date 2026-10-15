package cli

import (
	"context"
	"errors"
	"reflect"
	"runtime"
	"testing"
	"weak"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type retainedRuntimeValue struct{ Payload []byte }
type retainedCleanupPayload struct{ Payload []byte }
type retentionObservations struct {
	values  []weak.Pointer[retainedRuntimeValue]
	cleanup []weak.Pointer[retainedCleanupPayload]
	closes  int
}
type retentionCodec struct {
	codec.Codec[string, string]
	observed *retentionObservations
}

func (this retentionCodec) ToRaw(ctx context.Context) codec.Codec[any, any] {
	return codec.Raw(ctx, this)
}
func (retentionCodec) CanHandleType(ctx context.Context, typ reflect.Type) (bool, error) {
	return typ == reflect.TypeFor[*retainedRuntimeValue](), ctx.Err()
}
func (this retentionCodec) Resolve(ctx context.Context, scope codec.Scope, _ string) (*retainedRuntimeValue, error) {
	value := &retainedRuntimeValue{Payload: make([]byte, 64<<10)}
	cleanup := &retainedCleanupPayload{Payload: make([]byte, 64<<10)}
	this.observed.values = append(this.observed.values, weak.Make(value))
	this.observed.cleanup = append(this.observed.cleanup, weak.Make(cleanup))
	scope.OnClose(func() error { this.observed.closes++; runtime.KeepAlive(cleanup); return nil })
	return value, ctx.Err()
}

func awaitWeakCollection[T any](t *testing.T, references []weak.Pointer[T]) {
	t.Helper()
	for range 8 {
		runtime.GC()
		collected := true
		for _, reference := range references {
			if reference.Value() != nil {
				collected = false
				break
			}
		}
		if collected {
			return
		}
		runtime.Gosched()
	}
	for index, reference := range references {
		assert.Nil(t, reference.Value(), "retained object %d", index)
	}
}

// Keep the application and closed invocation alive: completed cleanup callbacks
// must release their captures, even though public getters keep runtime values.
func TestClosedInvocationDropsCleanupCapturesButKeepsReadableValues(t *testing.T) {
	observed := new(retentionObservations)
	app, err := NewSimpleApplication().WithName("app").WithErrorHandlerEnabled(false).
		WithFlag(NewFlag(retentionCodec{Codec: codec.NewString(), observed: observed}).WithName("value")).Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(nil, nil, "--value=payload")
	require.NoError(t, err)
	require.NoError(t, invocation.Close())
	require.NoError(t, invocation.Close())
	assert.Equal(t, 1, observed.closes)
	awaitWeakCollection(t, observed.cleanup)
	_, value := invocation.Flag("value")
	require.Len(t, value.(*retainedRuntimeValue).Payload, 64<<10)
	runtime.KeepAlive(invocation)
	runtime.KeepAlive(app)
}

func TestSuccessfulAndFailedInvocationsAreNotRetainedByParsePools(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "rollback"}[fail], func(t *testing.T) {
			observed := new(retentionObservations)
			builder := NewSimpleApplication().WithName("app").WithErrorHandlerEnabled(false).
				WithFlag(NewFlag(retentionCodec{Codec: codec.NewString(), observed: observed}).WithName("value"))
			primary := errors.New("validate failed")
			if fail {
				builder.WithValidateHandler(func(context.Context, NoOptions) error { return primary })
			}
			app, err := builder.Create()
			require.NoError(t, err)
			defer func() { assert.NoError(t, app.Close()) }()
			for range 64 {
				runRetentionInvocation(t, app, fail, primary)
			}
			assert.Equal(t, 64, observed.closes)
			awaitWeakCollection(t, observed.values)
			awaitWeakCollection(t, observed.cleanup)
			runtime.KeepAlive(app)
		})
	}
}

// Prevent the test's last invocation variable from extending object lifetime.
//
//go:noinline
func runRetentionInvocation(t *testing.T, app *Application[NoOptions], fail bool, primary error) {
	t.Helper()
	invocation, err := app.Parse(nil, nil, "--value=payload")
	if fail {
		require.ErrorIs(t, err, primary)
		require.Nil(t, invocation)
		return
	}
	require.NoError(t, err)
	require.NoError(t, invocation.Close())
}
