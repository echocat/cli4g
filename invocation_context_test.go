package cli

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/echocat/cli4g/codec"
)

type contextRootOptions struct {
	Root  string `flag:"root" required:"true"`
	Ready bool
	ctx   context.Context
}

func (this *contextRootOptions) Initialize(ctx context.Context) error {
	this.ctx = ctx
	this.Ready = true
	invocation, found := InvocationFromContext(ctx)
	if !found {
		return errors.New("missing invocation during options initialization")
	}
	typed, found := invocation.(*Invocation[contextRootOptions])
	if !found {
		return errors.New("context does not contain the actual typed invocation")
	}
	_, current := typed.Application()
	if !current.Ready {
		return errors.New("context returned stale value options")
	}
	return nil
}

func (this *contextRootOptions) Close() error {
	this.Ready = false
	if this.ctx == nil {
		return nil
	}
	invocation, found := InvocationFromContext(this.ctx)
	if !found {
		return errors.New("missing invocation during options close")
	}
	typed, found := invocation.(*Invocation[contextRootOptions])
	if !found {
		return errors.New("wrong invocation during options close")
	}
	_, current := typed.Application()
	if current.Ready {
		return errors.New("context returned stale closed options")
	}
	return nil
}

type contextChildOptions struct {
	Child string `argument:"child" required:"true"`
	Ready bool
	ctx   context.Context
}

func (this *contextChildOptions) Initialize(ctx context.Context) error {
	this.ctx = ctx
	this.Ready = true
	invocation, found := InvocationFromContext(ctx)
	if !found {
		return errors.New("missing invocation during command options initialization")
	}
	container, found := ContainerFromContext(ctx)
	if !found {
		return errors.New("missing current container")
	}
	command, found := container.(*Command[contextChildOptions])
	if !found {
		return errors.New("wrong current command")
	}
	current, found := command.OptionsFrom(invocation)
	if !found || !current.Ready {
		return errors.New("context returned stale command options")
	}
	return nil
}

func TestInvocationContextInLifecycleHandlers(t *testing.T) {
	type markerKey struct{}
	var events []string
	var seenInvocation InvocationLike
	check := func(ctx context.Context, stage, containerName string) {
		invocation, found := InvocationFromContext(ctx)
		require.True(t, found)
		require.IsType(t, &Invocation[contextRootOptions]{}, invocation)
		if seenInvocation == nil {
			seenInvocation = invocation
		} else {
			assert.Same(t, seenInvocation, invocation)
		}
		assert.Equal(t, "app", invocation.Root().Name())
		assert.Equal(t, []string{"child"}, invocation.NameStack())
		require.Len(t, invocation.Stack(), 1)
		_, value := invocation.Flag("root")
		assert.Equal(t, "root-value", value)
		_, value = invocation.Flag("extra")
		assert.Equal(t, "extra-value", value, "the context must contain the complete invocation, not a scope view")
		_, value = invocation.Argument(0)
		assert.Equal(t, "child-value", value)
		container, found := ContainerFromContext(ctx)
		require.True(t, found)
		assert.Equal(t, containerName, container.Name())
		marker := "initialize"
		if stage == "execute" {
			marker = "execute"
		}
		assert.Equal(t, marker, ctx.Value(markerKey{}))
		events = append(events, stage+" "+containerName)
	}
	app := NewApplication[contextRootOptions]("app").
		WithValidateHandler(func(ctx context.Context, options contextRootOptions) error {
			assert.False(t, options.Ready)
			check(ctx, "validate", "app")
			return nil
		}).
		WithInitializeHandler(func(ctx context.Context, options contextRootOptions) error {
			assert.True(t, options.Ready)
			check(ctx, "initialize", "app")
			return nil
		}).
		WithHandler(func(ctx context.Context, options contextRootOptions) error {
			assert.True(t, options.Ready)
			check(ctx, "execute", "app")
			return nil
		}).
		WithCloseHandler(func(options contextRootOptions) error {
			check(options.ctx, "close", "app")
			return nil
		}).
		WithCommand(NewCommand[contextChildOptions]("child").
			WithFlag(NewFlag(codec.NewString(), "extra")).
			WithValidateHandler(func(ctx context.Context, options contextChildOptions) error {
				assert.False(t, options.Ready)
				check(ctx, "validate", "child")
				return nil
			}).
			WithInitializeHandler(func(ctx context.Context, options contextChildOptions) error {
				assert.True(t, options.Ready)
				check(ctx, "initialize", "child")
				return nil
			}).
			WithHandler(func(ctx context.Context, options contextChildOptions) error {
				assert.True(t, options.Ready)
				check(ctx, "execute", "child")
				return nil
			}).
			WithCloseHandler(func(options contextChildOptions) error {
				check(options.ctx, "close", "child")
				return nil
			}),
		).
		MustCreate()
	ctx := context.WithValue(context.Background(), markerKey{}, "initialize")
	invocation := app.MustParse(ctx, nil, "--root=root-value", "child", "--extra=extra-value", "child-value")
	assert.Same(t, invocation, seenInvocation)
	_, found := InvocationFromContext(ctx)
	assert.False(t, found, "the caller's context must not be mutated")
	require.NoError(t, invocation.Execute(context.WithValue(context.Background(), markerKey{}, "execute")))
	require.NoError(t, invocation.Close())
	_, rootOptions := invocation.Application()
	assert.False(t, rootOptions.Ready)
	assert.Equal(t, []string{
		"validate app", "validate child",
		"initialize app", "initialize child",
		"execute app", "execute child",
		"close child", "close app",
	}, events)
}

type contextCheckingDecoder struct{ codec.Codec[string, string] }

func (this contextCheckingDecoder) ParseRequirement(ctx context.Context, usage codec.Usage, args ...string) (string, error) {
	if _, found := InvocationFromContext(ctx); found {
		return "", errors.New("new invocation was attached before decoding finished")
	}
	return this.Codec.ParseRequirement(ctx, usage, args...)
}

func (this contextCheckingDecoder) ToRaw(ctx context.Context) codec.Codec[any, any] {
	return codec.Raw(ctx, this)
}
func (this contextCheckingDecoder) Clone(context.Context) (codec.Codec[string, string], error) {
	return this, nil
}
func (this contextCheckingDecoder) CloneDirect() codec.Codec[string, string] { return this }

func TestInvocationContextIsNotIntroducedDuringDecoding(t *testing.T) {
	type options struct {
		Value string `flag:"value" env:"VALUE" required:"true"`
	}
	app, err := NewApplication[*options]("app").
		WithCodec(contextCheckingDecoder{codec.NewString()}).
		Create()
	require.NoError(t, err)
	opts := &ParseOpts{LookupEnv: func(ctx context.Context, _ string) (string, bool) {
		_, found := InvocationFromContext(ctx)
		assert.False(t, found)
		return "env-value", true
	}}
	invocation, err := app.Parse(context.Background(), opts)
	require.NoError(t, err)
	require.NoError(t, invocation.Close())
	invocation, err = app.Parse(context.Background(), nil, "--value=cli-value")
	require.NoError(t, err)
	require.NoError(t, invocation.Close())
}

func TestInvocationContextDuringRollback(t *testing.T) {
	type markerKey struct{}
	failure := errors.New("initialization failed")
	cleanupSeen := false
	app := NewApplication[contextRootOptions]("app").
		WithCloseHandler(func(options contextRootOptions) error {
			ctx := options.ctx
			invocation, found := InvocationFromContext(ctx)
			require.True(t, found)
			assert.Equal(t, []string{"child"}, invocation.NameStack())
			_, value := invocation.Flag("extra")
			assert.Equal(t, "value", value)
			container, found := ContainerFromContext(ctx)
			require.True(t, found)
			assert.Equal(t, "app", container.Name())
			assert.Equal(t, "initialize", ctx.Value(markerKey{}))
			cleanupSeen = true
			return nil
		}).
		WithCommand(NewSimpleCommand("child").
			WithFlag(NewFlag(codec.NewString(), "extra")).
			WithInitializeHandler(func(ctx context.Context, _ NoOptions) error {
				invocation, found := InvocationFromContext(ctx)
				require.True(t, found)
				assert.Equal(t, []string{"child"}, invocation.NameStack())
				return failure
			}),
		).
		MustCreate()
	ctx := context.WithValue(context.Background(), markerKey{}, "initialize")
	invocation, err := app.Parse(ctx, nil, "--root=root-value", "child", "--extra=value")
	require.ErrorIs(t, err, failure)
	assert.Nil(t, invocation)
	assert.True(t, cleanupSeen)
}

func TestInvocationContextIsolationAcrossConcurrentCalls(t *testing.T) {
	type options struct {
		Value string `flag:"value" required:"true"`
	}
	app, err := NewApplication[*options]("app").
		WithHandler(func(ctx context.Context, options *options) error {
			invocation, found := InvocationFromContext(ctx)
			if !found {
				return errors.New("missing invocation")
			}
			_, value := invocation.Flag("value")
			if value != options.Value {
				return fmt.Errorf("wrong invocation value: %v", value)
			}
			return nil
		}).
		Create()
	require.NoError(t, err)
	var wg sync.WaitGroup
	for i := range 40 {
		wg.Go(func() {
			ctx := context.Background()
			invocation, err := app.Parse(ctx, nil, "--value="+fmt.Sprint(i))
			if !assert.NoError(t, err) {
				return
			}
			assert.NoError(t, invocation.Execute(ctx))
			assert.NoError(t, invocation.Close())
		})
	}
	wg.Wait()
}

func TestInvocationFromContextWithoutInvocation(t *testing.T) {
	_, found := InvocationFromContext(context.Background())
	assert.False(t, found)
	_, found = InvocationFromContext(nil)
	assert.False(t, found)
}
