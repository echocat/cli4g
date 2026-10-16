package codec_test

import (
	"context"
	"errors"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCombinedDefaultsBindAllStandardTypes(t *testing.T) {
	ctx := context.Background()
	defaults := codec.Defaults(ctx)
	require.Len(t, defaults, 1)
	require.IsType(t, codec.CombinedDefaults{}, defaults[0])
	registry, err := codec.NewRegistry(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, registry.AddDefaults(ctx))
	for _, test := range []struct {
		typ   reflect.Type
		input string
	}{
		{reflect.TypeFor[bool](), "true"}, {reflect.TypeFor[string](), "value"},
		{reflect.TypeFor[time.Duration](), "42s"}, {reflect.TypeFor[time.Time](), "2026-10-14T12:30:45.123456789Z"},
		{reflect.TypeFor[int](), "42"}, {reflect.TypeFor[int8](), "42"}, {reflect.TypeFor[int16](), "42"},
		{reflect.TypeFor[int32](), "42"}, {reflect.TypeFor[int64](), "42"},
		{reflect.TypeFor[uint](), "42"}, {reflect.TypeFor[uint8](), "42"}, {reflect.TypeFor[uint16](), "42"},
		{reflect.TypeFor[uint32](), "42"}, {reflect.TypeFor[uint64](), "42"}, {reflect.TypeFor[uintptr](), "42"},
		{reflect.TypeFor[float32](), "1.25"}, {reflect.TypeFor[float64](), "1.25"},
		{reflect.TypeFor[complex64](), "1+2i"}, {reflect.TypeFor[complex128](), "1+2i"},
	} {
		for shape := range 3 {
			typ, input := test.typ, test.input
			if shape == 1 {
				typ = reflect.SliceOf(typ)
			}
			if shape == 2 {
				typ, input = reflect.MapOf(reflect.TypeFor[string](), typ), "key="+input
			}
			t.Run(typ.String(), func(t *testing.T) {
				bound, found, err := registry.Find(ctx, typ, "")
				require.NoError(t, err)
				require.True(t, found)
				_, unresolved := bound.(codec.TypeBinder)
				assert.False(t, unresolved, "runtime uses a concrete codec, not the resolver")
				requirement, err := bound.ParseRequirement(ctx, codec.UsageDefault, input)
				require.NoError(t, err)
				assert.Equal(t, typ, reflect.TypeOf(requirement))
				plain, err := bound.FormatRequirement(ctx, requirement)
				require.NoError(t, err)
				roundtrip, err := bound.ParseRequirement(ctx, codec.UsageDefault, plain...)
				require.NoError(t, err)
				assert.Equal(t, requirement, roundtrip)
				minimum, maximum := bound.ExpectMinMax(ctx, codec.UsageFlag)
				if shape == 0 && test.typ == reflect.TypeFor[bool]() {
					assert.Equal(t, 0, minimum)
				} else {
					assert.Equal(t, 1, minimum)
				}
				assert.Equal(t, 1, maximum)
				assert.Equal(t, shape != 0, codec.IsCumulative(bound))
			})
		}
	}
	for _, typ := range []reflect.Type{reflect.TypeFor[*os.File](), reflect.TypeFor[[]*os.File]()} {
		bound, found, err := registry.Find(ctx, typ, "")
		require.NoError(t, err)
		require.True(t, found)
		assert.Equal(t, typ == reflect.TypeFor[[]*os.File](), codec.IsCumulative(bound))
		_, err = bound.ParseRequirement(ctx, codec.UsageFlag, "file-path")
		assert.NoError(t, err)
	}
}

func TestCombinedDefaultsDoNotCaptureLastRequestedType(t *testing.T) {
	registry, err := codec.NewRegistry(context.Background(), nil)
	require.NoError(t, err)
	require.NoError(t, registry.AddDefaults(context.Background()))
	var workers sync.WaitGroup
	for i := range 24 {
		workers.Go(func() {
			typ, input := reflect.TypeFor[string](), "42"
			if i%2 == 0 {
				typ = reflect.TypeFor[uint8]()
			}
			bound, found, err := registry.Find(context.Background(), typ, "")
			if !assert.NoError(t, err) || !assert.True(t, found) {
				return
			}
			value, err := bound.ParseRequirement(context.Background(), codec.UsageFlag, input)
			assert.NoError(t, err)
			assert.Equal(t, typ, reflect.TypeOf(value))
		})
	}
	workers.Wait()
}

func TestCombinedDefaultsRejectUnsupportedAndUnboundOperations(t *testing.T) {
	type namedInteger uint16
	type namedDuration time.Duration
	ctx := context.Background()
	resolver := codec.NewCombinedDefaults()
	binder := resolver.(codec.TypeBinder)
	for _, typ := range []reflect.Type{nil, reflect.TypeFor[namedInteger](), reflect.TypeFor[namedDuration](), reflect.TypeFor[[]namedInteger](), reflect.TypeFor[map[int]string](), reflect.TypeFor[struct{}]()} {
		_, matched, err := binder.BindType(ctx, typ)
		assert.NoError(t, err)
		assert.False(t, matched)
	}
	_, err := resolver.ParseRequirement(ctx, codec.UsageFlag, "42")
	assert.ErrorIs(t, err, codec.ErrInvalid)
	_, err = resolver.FormatRequirement(ctx, "42")
	assert.ErrorIs(t, err, codec.ErrInvalid)
	_, err = resolver.DisplayRequirement(ctx, "42")
	assert.ErrorIs(t, err, codec.ErrInvalid)
	_, err = resolver.Resolve(ctx, new(codec.ResourceScope), "42")
	assert.ErrorIs(t, err, codec.ErrInvalid)
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	_, _, err = binder.BindType(ctx, reflect.TypeFor[uint8]())
	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, codec.Defaults(ctx))
}

type testTypeBinder struct {
	codec.Codec[string, string]
	bound   codec.Codec[any, any]
	failure error
	cancel  context.CancelFunc
	seen    context.Context
}

func (c *testTypeBinder) BindType(ctx context.Context, typ reflect.Type) (codec.Codec[any, any], bool, error) {
	c.seen = ctx
	if c.cancel != nil {
		c.cancel()
	}
	return c.bound, typ == reflect.TypeFor[uint8](), c.failure
}

func (c *testTypeBinder) ToRaw(ctx context.Context) codec.Codec[any, any] { return codec.Raw(ctx, c) }
func (c *testTypeBinder) EnumVariants(context.Context) ([]codec.EnumVariant[string], error) {
	return []codec.EnumVariant[string]{{Value: "a"}}, nil
}

func TestRawTypeBinderPreservesEnumMetadataAndRegistrySelection(t *testing.T) {
	type markerKey struct{}
	ctx := context.WithValue(context.Background(), markerKey{}, "binding")
	source := &testTypeBinder{Codec: codec.NewString(), bound: codec.NewUint8().ToRaw(ctx)}
	raw := source.ToRaw(ctx)
	_, binder := raw.(codec.TypeBinder)
	_, enumeration := raw.(codec.Enum[any])
	assert.True(t, binder)
	assert.True(t, enumeration)
	registry, err := codec.NewRegistry(ctx, []codec.Registration{codec.NewRegistration(source, "number")})
	require.NoError(t, err)
	bound, found, err := registry.Find(ctx, reflect.TypeFor[uint8](), "number")
	require.NoError(t, err)
	require.True(t, found)
	assert.Same(t, ctx, source.seen)
	value, err := bound.ParseRequirement(ctx, codec.UsageFlag, "42")
	assert.NoError(t, err)
	assert.Equal(t, uint8(42), value)
	_, found, err = registry.Find(ctx, reflect.TypeFor[uint8](), "other")
	assert.NoError(t, err)
	assert.False(t, found)
}

func TestTypeBinderErrorsAndCancellationArePreserved(t *testing.T) {
	for _, stage := range []string{"nil binding", "failure", "cancellation"} {
		t.Run(stage, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			failure := errors.New("binding failed")
			source := &testTypeBinder{Codec: codec.NewString()}
			if stage == "failure" {
				source.failure = failure
			}
			if stage == "cancellation" {
				source.cancel = cancel
			}
			registry, err := codec.NewRegistry(context.Background(), []codec.Registration{codec.NewRegistration(source)})
			require.NoError(t, err)
			_, found, err := registry.Find(ctx, reflect.TypeFor[uint8](), "")
			assert.False(t, found)
			switch stage {
			case "nil binding":
				assert.ErrorIs(t, err, codec.ErrInvalid)
			case "failure":
				assert.ErrorIs(t, err, failure)
			case "cancellation":
				assert.ErrorIs(t, err, context.Canceled)
			}
		})
	}
}
