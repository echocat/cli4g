package codec_test

import (
	"context"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/echocat/cli4g"
	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type registryValue struct{ From, Text string }
type registryCodec struct {
	from   string
	checks *atomic.Int64
}

func (this registryCodec) ToRaw(ctx context.Context) codec.Codec[any, any] {
	return codec.Raw(ctx, this)
}
func (this registryCodec) CanHandleType(_ context.Context, typ reflect.Type) (bool, error) {
	if this.checks != nil {
		this.checks.Add(1)
	}
	return typ == reflect.TypeFor[registryValue](), nil
}
func (registryCodec) ExpectMinMax(context.Context, codec.Usage) (int, int) { return 1, 1 }
func (this registryCodec) ParseRequirement(ctx context.Context, usage codec.Usage, args ...string) (string, error) {
	if err := codec.EnsureArgumentsMatching(ctx, usage, this, args...); err != nil {
		return "", err
	}
	return args[0], nil
}
func (registryCodec) FormatRequirement(ctx context.Context, input string) ([]string, error) {
	return []string{input}, ctx.Err()
}
func (this registryCodec) DisplayRequirement(ctx context.Context, input string) ([]string, error) {
	return this.FormatRequirement(ctx, input)
}
func (this registryCodec) Resolve(ctx context.Context, _ codec.Scope, input string) (registryValue, error) {
	return registryValue{this.from, input}, ctx.Err()
}

func TestCodecRegistryDeepestWinsForAncestorOptionsAndNamedAliases(t *testing.T) {
	type options struct {
		Named   registryValue `flag:"named" type:"moo" default:"fallback"`
		Alias   registryValue `flag:"alias" type:"other"`
		Unnamed registryValue `flag:"unnamed"`
	}
	app := cli.NewApplication[*options]("app").
		WithErrorHandlerEnabled(false).
		WithCodec(registryCodec{from: "root"}, "moo", "other").
		WithCodec(registryCodec{from: "unnamed"}).
		WithCommand(cli.NewSimpleCommand("parent").
			WithCodec(registryCodec{from: "parent"}, "moo").
			WithCommand(cli.NewSimpleCommand("leaf").
				WithCodec(registryCodec{from: "leaf"}, "moo"),
			),
		).
		MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	for _, test := range []struct {
		args       []string
		from, text string
	}{
		{[]string{"--named=first", "--alias=second", "--unnamed=third", "parent", "leaf"}, "leaf", "first"},
		{[]string{"parent"}, "parent", "fallback"},
		{nil, "root", "fallback"},
	} {
		invocation, err := app.Parse(nil, nil, test.args...)
		require.NoError(t, err)
		_, parsed := invocation.Application()
		assert.Equal(t, registryValue{test.from, test.text}, parsed.Named)
		if len(test.args) > 2 {
			assert.Equal(t, registryValue{"root", "second"}, parsed.Alias)
			assert.Equal(t, registryValue{"unnamed", "third"}, parsed.Unnamed)
		}
		require.NoError(t, invocation.Close())
	}
}

func TestRegistryResolutionIsCompileTimeAndParallelRequirementsAreIndependent(t *testing.T) {
	checks := new(atomic.Int64)
	type options struct {
		Value registryValue `flag:"value"`
	}
	app, err := cli.NewApplication[options]("app").WithCodec(registryCodec{from: "selected", checks: checks}).Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, app.Close()) }()
	baseline := checks.Load()
	var workers sync.WaitGroup
	results := make(chan error, 32)
	for range 32 {
		workers.Go(func() {
			invocation, err := app.Parse(nil, nil, "--value=one")
			if err == nil {
				err = invocation.Close()
			}
			results <- err
		})
	}
	workers.Wait()
	close(results)
	for err := range results {
		require.NoError(t, err)
	}
	assert.Equal(t, baseline, checks.Load(), "Parse never searches the registry")
}

func TestRegistryAndSnapshotRejectMissingUnknownAndInvalidCodecs(t *testing.T) {
	for _, candidate := range []any{nil, 42, (*registryCodec)(nil)} {
		_, err := codec.Snapshot(context.Background(), candidate)
		require.ErrorIs(t, err, codec.ErrInvalid)
	}
	type options struct {
		Value registryValue `flag:"value" type:"missing"`
	}
	_, err := cli.NewApplication[options]("app").WithCodec(registryCodec{from: "unnamed"}).Create()
	require.ErrorContains(t, err, "no codec")
	_, err = codec.NewRegistry(context.Background(), []codec.Registration{{Codec: registryCodec{}, Names: []string{""}}})
	require.ErrorContains(t, err, "empty codec name")
}
