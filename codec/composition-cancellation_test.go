package codec_test

import (
	"context"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type cancelingRequirementCallbacks struct {
	codec.Codec[string, string]
	cancel context.CancelFunc
	calls  *int
}

func (this cancelingRequirementCallbacks) ParseRequirement(context.Context, codec.Usage, ...string) (string, error) {
	*this.calls++
	this.cancel()
	return "value", nil
}
func (this cancelingRequirementCallbacks) FormatRequirement(context.Context, string) ([]string, error) {
	*this.calls++
	this.cancel()
	return []string{"value"}, nil
}
func (this cancelingRequirementCallbacks) DisplayRequirement(context.Context, string) ([]string, error) {
	*this.calls++
	this.cancel()
	return []string{"value"}, nil
}
func (this cancelingRequirementCallbacks) CloneRequirement(context.Context, string) (string, error) {
	*this.calls++
	this.cancel()
	return "value", nil
}
func TestSliceRequirementCallbacksStopAfterCancellation(t *testing.T) {
	for _, operation := range []string{"parse", "format", "display", "clone"} {
		t.Run(operation, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			source := codec.NewSlice(cancelingRequirementCallbacks{Codec: codec.NewString(), cancel: cancel, calls: &calls})
			var err error
			switch operation {
			case "parse":
				_, err = source.ParseRequirement(ctx, codec.UsageDefault, "one", "two")
			case "format":
				_, err = source.FormatRequirement(ctx, []string{"one", "two"})
			case "display":
				_, err = source.DisplayRequirement(ctx, []string{"one", "two"})
			case "clone":
				_, err = codec.CloneRequirement(ctx, source, []string{"one", "two"})
			}
			require.ErrorIs(t, err, context.Canceled)
			assert.Equal(t, 1, calls)
		})
	}
}

func TestMapRequirementCallbacksStopAfterCancellation(t *testing.T) {
	for _, operation := range []string{"parse", "format", "display", "clone"} {
		t.Run(operation, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			calls := 0
			source := codec.NewMap[string](cancelingRequirementCallbacks{Codec: codec.NewString(), cancel: cancel, calls: &calls})
			values := map[string]string{"a": "one", "b": "two"}
			var err error
			switch operation {
			case "parse":
				_, err = source.ParseRequirement(ctx, codec.UsageDefault, "a=one", "b=two")
			case "format":
				_, err = source.FormatRequirement(ctx, values)
			case "display":
				_, err = source.DisplayRequirement(ctx, values)
			case "clone":
				_, err = codec.CloneRequirement(ctx, source, values)
			}
			require.ErrorIs(t, err, context.Canceled)
			assert.Equal(t, 1, calls)
		})
	}
}

type cancelingElementCodec struct {
	codec.Codec[int, int]
	cancel context.CancelFunc
	calls  *int
	closes *int
}

func (this cancelingElementCodec) Resolve(_ context.Context, scope codec.Scope, value int) (int, error) {
	*this.calls++
	scope.OnClose(func() error { *this.closes++; return nil })
	this.cancel()
	return value, nil // composition must enforce its own boundary
}
func TestSliceResolveStopsAfterElementCancelsAndRetainsCleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls, closes := 0, 0
	source := codec.NewSlice(cancelingElementCodec{Codec: codec.NewInt(), cancel: cancel, calls: &calls, closes: &closes})
	scope := new(codec.ResourceScope)
	_, err := source.Resolve(ctx, scope, []int{1, 2, 3})
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, calls)
	require.NoError(t, scope.Close())
	require.NoError(t, scope.Close())
	assert.Equal(t, 1, closes)
}

func TestMapResolveStopsAfterElementCancelsAndRetainsCleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls, closes := 0, 0
	source := codec.NewMap[string](cancelingElementCodec{Codec: codec.NewInt(), cancel: cancel, calls: &calls, closes: &closes})
	scope := new(codec.ResourceScope)
	_, err := source.Resolve(ctx, scope, map[string]int{"a": 1, "b": 2, "c": 3})
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, calls)
	require.NoError(t, scope.Close())
	require.NoError(t, scope.Close())
	assert.Equal(t, 1, closes)
}

type cancellationTextTarget struct{ unmarshals, closes *int }

func (*cancellationTextTarget) MarshalText() ([]byte, error)      { return nil, nil }
func (target *cancellationTextTarget) UnmarshalText([]byte) error { *target.unmarshals++; return nil }
func (target *cancellationTextTarget) Close() error               { *target.closes++; return nil }
func TestMarshallerFactoryCancellationSkipsUnmarshalAndKeepsOwnership(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	unmarshals, closes := 0, 0
	source := codec.NewMarshallerP(func() *cancellationTextTarget { cancel(); return &cancellationTextTarget{&unmarshals, &closes} })
	scope := new(codec.ResourceScope)
	_, err := source.Resolve(ctx, scope, "input")
	require.ErrorIs(t, err, context.Canceled)
	assert.Zero(t, unmarshals)
	require.NoError(t, scope.Close())
	require.NoError(t, scope.Close())
	assert.Equal(t, 1, closes)
}
