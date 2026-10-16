package cli

import (
	"context"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/echocat/cli4g/codec"
)

func TestPreparedFlagsPreserveAliasesScopesAndEnvironmentFallback(t *testing.T) {
	type rootOptions struct {
		Items   []string `flag:"items|item" short:"i"`
		Name    string   `flag:"name" env:"BIFROEST_CLI_RESERVE_NAME"`
		EnvOnly string   `flag:"env-only" env:"BIFROEST_CLI_RESERVE_ENV" required:"true"`
		Unused  int      `flag:"unused"`
	}
	type childOptions struct {
		Count int `flag:"count"`
	}
	app := NewApplication[rootOptions]("app").
		WithCommand(NewCommand[*childOptions]("child")).
		MustCreate()
	var lookups []string
	opts := &ParseOpts{LookupEnv: func(_ context.Context, name string) (string, bool) {
		lookups = append(lookups, name)
		return "environment", name == "BIFROEST_CLI_RESERVE_ENV"
	}}
	invocation, err := app.Parse(context.Background(), opts,
		"--item=one", "--name=CLI", "child", "--count=2", "--items=two", "-ithree")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, root := invocation.Application()
	assert.Equal(t, []string{"one", "two", "three"}, root.Items)
	assert.Equal(t, "CLI", root.Name)
	assert.Equal(t, "environment", root.EnvOnly)
	assert.Zero(t, root.Unused)
	assert.Equal(t, []string{"BIFROEST_CLI_RESERVE_ENV"}, lookups)
	child, found := FindCommand[*childOptions](app, "child")
	require.True(t, found)
	value, found := child.OptionsFrom(invocation)
	require.True(t, found)
	assert.Equal(t, 2, value.Count)
	var names []string
	for flag := range invocation.Flags() {
		names = append(names, flag.Name())
	}
	assert.Equal(t, []string{"items", "name", "env-only", "count"}, names)
}

func TestRepeatedSliceFlagsKeepPerOccurrenceCodecIsolation(t *testing.T) {
	type options struct {
		Values []int `flag:"value"`
	}
	var clones atomic.Int64
	app, err := NewApplication[*options]("app").
		WithCodec(callbackCodec[[]int, []int]{
			Codec: codec.NewSlice(&constantOneCodec{}),
			onClone: func(context.Context) {
				clones.Add(1)
			},
		}).Create()
	require.NoError(t, err)
	args := make([]string, 16)
	want := make([]int, len(args))
	for i := range args {
		args[i] = "--value=ignored"
		want[i] = 1
	}
	var workers sync.WaitGroup
	for range 32 {
		workers.Go(func() {
			invocation, err := app.Parse(context.Background(), nil, args...)
			if !assert.NoError(t, err) {
				return
			}
			defer func() { assert.NoError(t, invocation.Close()) }()
			_, parsed := invocation.Application()
			assert.Equal(t, want, parsed.Values)
			_, value := invocation.Flag("value")
			assert.Equal(t, want, value)
		})
	}
	workers.Wait()
	assert.EqualValues(t, 32, clones.Load(), "one Resolve per aggregate")
}

type flagChunkValues []string

// flagChunkCodec returns custom named slices, including empty and overlapping
// chunks, to exercise aggregation beyond the standard one-element codecs.
type flagChunkCodec struct {
	decode func(string) flagChunkValues
}

func (c flagChunkCodec) ToRaw(ctx context.Context) codec.Codec[any, any] { return codec.Raw(ctx, c) }
func (flagChunkCodec) IsCumulative() bool                                { return true }
func (c flagChunkCodec) ParseRequirementInto(ctx context.Context, usage codec.Usage, previous flagChunkValues, args ...string) (flagChunkValues, error) {
	additional, err := c.ParseRequirement(ctx, usage, args...)
	if err != nil {
		return previous, err
	}
	return append(previous, additional...), ctx.Err()
}
func (flagChunkCodec) CanHandleType(_ context.Context, typ reflect.Type) (bool, error) {
	return typ == reflect.TypeFor[flagChunkValues](), nil
}
func (flagChunkCodec) ExpectMinMax(context.Context, codec.Usage) (int, int) { return 1, 1 }
func (c flagChunkCodec) Parse(ctx context.Context, usage codec.Usage, args ...string) (flagChunkValues, error) {
	if err := codec.EnsureArgumentsMatching(ctx, usage, c, args...); err != nil {
		return nil, err
	}
	return c.decode(args[0]), nil
}
func (flagChunkCodec) Format(_ context.Context, values flagChunkValues) ([]string, error) {
	return []string{strings.Join(values, ",")}, nil
}
func (c flagChunkCodec) Clone(context.Context) (codec.Codec[flagChunkValues, flagChunkValues], error) {
	return c, nil
}
func (c flagChunkCodec) CloneDirect() codec.Codec[flagChunkValues, flagChunkValues] { return c }

func (c flagChunkCodec) CloneValue(ctx context.Context, value flagChunkValues) (flagChunkValues, error) {
	return append(value[:0:0], value...), ctx.Err()
}

func TestRepeatedCustomSliceChunksPreserveAppendSemantics(t *testing.T) {
	type options struct {
		Values flagChunkValues
	}
	for _, test := range []struct {
		name  string
		first flagChunkValues
		last  flagChunkValues
		want  flagChunkValues
	}{
		{
			name: "nil",
		},
		{
			name:  "empty",
			first: flagChunkValues{},
			last:  flagChunkValues{},
			want:  flagChunkValues{},
		},
		{
			name:  "multiple",
			first: flagChunkValues{"a", "b"},
			last:  flagChunkValues{"c", "d"},
			want:  flagChunkValues{"a", "b", "c", "d"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			decoder := flagChunkCodec{decode: func(input string) flagChunkValues {
				if input == "first" {
					return test.first
				}
				return test.last
			}}
			app, err := NewApplication[options]("app").
				WithFlag(NewFlag(decoder, "value").
					WithField("Values"),
				).
				Create()
			require.NoError(t, err)
			invocation, err := app.Parse(context.Background(), nil, "--value=first", "--value=last")
			require.NoError(t, err)
			defer func() { assert.NoError(t, invocation.Close()) }()
			_, parsed := invocation.Application()
			assert.Equal(t, test.want, parsed.Values)
			_, value := invocation.Flag("value")
			assert.Equal(t, test.want, value)
		})
	}
	t.Run("overlapping chunks", func(t *testing.T) {
		backing := flagChunkValues{"a", "b", "c", "d", "e"}
		decoder := flagChunkCodec{decode: func(input string) flagChunkValues {
			if input == "first" {
				return backing[:1]
			}
			return backing[:3]
		}}
		app, err := NewApplication[*options]("app").
			WithFlag(NewFlag(decoder, "value").
				WithField("Values"),
			).
			Create()
		require.NoError(t, err)
		invocation, err := app.Parse(context.Background(), nil, "--value=first", "--value=last")
		require.NoError(t, err)
		defer func() { assert.NoError(t, invocation.Close()) }()
		_, parsed := invocation.Application()
		assert.Equal(t, flagChunkValues{"a", "a", "b", "c"}, parsed.Values)
		parsed.Values = flagChunkValues{"replacement"}
		_, value := invocation.Flag("value")
		assert.Equal(t, flagChunkValues{"a", "a", "b", "c"}, value)
	})
}

type nestedSliceFlagOptions struct {
	Values []string
}

func (*nestedSliceFlagOptions) UnmarshalText([]byte) error   { return nil }
func (*nestedSliceFlagOptions) MarshalText() ([]byte, error) { return nil, nil }

func TestRepeatedSliceFlagsUseDecodedHeaderAfterCallbackReplacesNestedField(t *testing.T) {
	type options struct {
		Nested *nestedSliceFlagOptions
	}
	var nested *nestedSliceFlagOptions
	parses := 0
	app, err := NewApplication[*options]("app").
		WithCreateOptionsHandler(func(context.Context) (*options, error) {
			nested = &nestedSliceFlagOptions{}
			return &options{Nested: nested}, nil
		}).
		WithFlag(NewFlag(callbackCodec[[]string, []string]{
			Codec: codec.NewStrings(),
			onParse: func(context.Context) {
				parses++
				if parses == 2 {
					nested.Values = []string{"replacement"}
				}
			},
		}, "value").
			WithField("Nested", "Values"),
		).
		Create()
	require.NoError(t, err)
	invocation, err := app.Parse(context.Background(), nil, "--value=first", "--value=last")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, parsed := invocation.Application()
	assert.Equal(t, []string{"first", "last"}, parsed.Nested.Values)
	parsed.Nested.Values = parsed.Nested.Values[:1]
	_, value := invocation.Flag("value")
	assert.Equal(t, []string{"first", "last"}, value)
}

type sliceResourceOptions struct {
	Resources []*ownedTestResource `flag:"resource"`
}

func (options *sliceResourceOptions) Close() error {
	return nil // element cleanup belongs to the scope, including partial results
}

func TestRepeatedSliceFlagsRetainResourcesOnCancellationAndDecodeFailure(t *testing.T) {
	for _, cancelOnDecode := range []bool{false, true} {
		name := "later decode failure"
		if cancelOnDecode {
			name = "cancellation during repeated flag"
		}
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var events []string
			var resources []*ownedTestResource
			parses := 0
			decoder := codec.NewSlice(codec.NewMarshallerP(func() *ownedTestResource {
				resource := &ownedTestResource{events: &events}
				resources = append(resources, resource)
				return resource
			}))
			app, err := NewApplication[*sliceResourceOptions]("app").
				WithCodec(callbackCodec[[]*ownedTestResource, []string]{
					Codec: decoder,
					onParse: func(context.Context) {
						parses++
						if cancelOnDecode && parses == 2 {
							cancel()
						}
					},
				}).WithFlag(NewFlag(codec.NewInt(), "number")).
				Create()
			require.NoError(t, err)
			invocation, err := app.Parse(ctx, nil, "--resource=first", "--resource=last", "--number=invalid")
			assert.Nil(t, invocation)
			if cancelOnDecode {
				require.ErrorIs(t, err, context.Canceled)
			} else {
				require.ErrorContains(t, err, "flag --number")
			}
			assert.Empty(t, resources, "Resolve is skipped after requirement errors/cancellation")
			assert.Empty(t, events)
		})
	}
}
