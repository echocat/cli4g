package cli

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZeroInvocationInspection(t *testing.T) {
	var invocation Invocation[NoOptions]
	app, options := invocation.Application()
	assert.Nil(t, app)
	assert.Equal(t, NoOptions{}, options)
	assert.Empty(t, invocation.Stack())
	assert.Empty(t, invocation.NameStack())
	flag, value := invocation.Flag("missing")
	assert.Nil(t, flag)
	assert.Nil(t, value)
	flag, value = invocation.ShortFlag('m')
	assert.Nil(t, flag)
	assert.Nil(t, value)
	argument, value := invocation.Argument(0)
	assert.Nil(t, argument)
	assert.Nil(t, value)
	for range invocation.Flags() {
		assert.Fail(t, "zero invocation must not yield flags")
	}
	for range invocation.Arguments() {
		assert.Fail(t, "zero invocation must not yield arguments")
	}
}

func TestExecuteCancellationStopsHandlersAndRetainsOwnership(t *testing.T) {
	for _, test := range []struct {
		stage  string
		events []string
	}{
		{stage: "before execution"},
		{stage: "expired deadline"},
		{stage: "root handler", events: []string{"root"}},
		{stage: "leaf handler", events: []string{"root", "leaf"}},
	} {
		t.Run(test.stage, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var events []string
			closed := false
			app := NewSimpleApplication().
				WithName("app").
				WithHandler(func(context.Context, NoOptions) error {
					events = append(events, "root")
					if test.stage == "root handler" {
						cancel()
					}
					return nil
				}).
				WithCloseHandler(func(NoOptions) error { closed = true; return nil }).
				WithCommand(NewSimpleCommand().
					WithName("leaf").
					WithHandler(func(context.Context, NoOptions) error {
						events = append(events, "leaf")
						if test.stage == "leaf handler" {
							cancel()
						}
						return nil
					}),
				).
				MustCreate()
			invocation := app.MustParse(context.Background(), nil, "leaf")
			defer func() { assert.NoError(t, invocation.Close()) }()
			if test.stage == "before execution" {
				cancel()
			}
			wantErr := context.Canceled
			if test.stage == "expired deadline" {
				var expire context.CancelFunc
				ctx, expire = context.WithDeadline(context.Background(), time.Unix(0, 0))
				defer expire()
				wantErr = context.DeadlineExceeded
			}
			assert.ErrorIs(t, invocation.Execute(ctx), wantErr)
			assert.Equal(t, test.events, events)
			assert.False(t, closed)
			require.NoError(t, invocation.Close())
			assert.True(t, closed)
		})
	}
}

func TestExecuteNormalizesNilContext(t *testing.T) {
	called := false
	app, err := NewSimpleApplication().WithName("app").
		WithHandler(func(ctx context.Context, _ NoOptions) error {
			require.NotNil(t, ctx)
			assert.NoError(t, ctx.Err())
			called = true
			return nil
		}).Create()
	require.NoError(t, err)
	invocation, err := app.Parse(nil, nil)
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	require.NoError(t, invocation.Execute(nil))
	assert.True(t, called)
}

type lifecycleOptions struct {
	Value string `argument:"value" required:"true"`
}

func (o *lifecycleOptions) Validate(context.Context) error {
	if o.Value != "valid" {
		return errors.New("invalid options")
	}
	return nil
}

func (o *lifecycleOptions) Close() error {
	o.Value = "closed"
	return nil
}

func TestLifecycle(t *testing.T) {
	var events []string
	a := NewSimpleApplication().
		WithName("app").
		WithInitializeHandler(func(context.Context, NoOptions) error { events = append(events, "init app"); return nil }).
		WithHandler(func(context.Context, NoOptions) error {
			events = append(events, "exec app")
			return nil
		}).
		WithCloseHandler(func(NoOptions) error {
			events = append(events, "close app")
			return errors.New("app error")
		}).
		WithCommand(NewSimpleCommand().
			WithName("child").
			WithInitializeHandler(func(context.Context, NoOptions) error { events = append(events, "init child"); return nil }).
			WithHandler(func(context.Context, NoOptions) error {
				events = append(events, "exec child")
				return nil
			}).
			WithCloseHandler(func(NoOptions) error {
				events = append(events, "close child")
				return errors.New("child error")
			})).
		MustCreate()
	i := a.MustParse(context.Background(), nil, "child")
	require.NoError(t, i.Execute(context.Background()))
	err := i.Close()
	assert.ErrorContains(t, err, "app error")
	assert.ErrorContains(t, err, "child error")
	want := []string{"init app", "init child", "exec app", "exec child", "close child", "close app"}
	assert.Equal(t, want, events)
}

func TestOptionsLifecycle(t *testing.T) {
	a, err := NewApplication[*lifecycleOptions]().
		WithName("options").
		Create()
	require.NoError(t, err)
	_, err = a.Parse(context.Background(), nil, "invalid")
	require.Error(t, err, "options initializer should reject invalid input")
	i, err := a.Parse(context.Background(), nil, "valid")
	require.NoError(t, err)
	require.NoError(t, i.Close())
	_, opts := i.Application()
	assert.Equal(t, "closed", opts.Value)
}

func TestTypedOptionsWithDifferentContextValues(t *testing.T) {
	type contextKey struct{}
	type options struct {
		Name string `flag:"name" required:"true"`
	}
	var events []string
	app, err := NewApplication[options]().
		WithName("app").
		WithInitializeHandler(func(ctx context.Context, value options) error {
			assert.Equal(t, "parse", ctx.Value(contextKey{}))
			assert.Equal(t, "value", value.Name)
			events = append(events, "initialize")
			return nil
		}).
		WithHandler(func(ctx context.Context, value options) error {
			assert.Equal(t, "execute", ctx.Value(contextKey{}))
			assert.Equal(t, "value", value.Name)
			events = append(events, "execute")
			return nil
		}).
		WithCloseHandler(func(value options) error {
			assert.Equal(t, "value", value.Name)
			events = append(events, "close")
			return nil
		}).
		Create()
	require.NoError(t, err)
	invocation, err := app.Parse(context.WithValue(context.Background(), contextKey{}, "parse"), nil, "--name=value")
	require.NoError(t, err)
	_, parsed := invocation.Application()
	assert.Equal(t, "value", parsed.Name)
	require.NoError(t, invocation.Execute(context.WithValue(context.Background(), contextKey{}, "execute")))
	require.NoError(t, invocation.Close())
	assert.Equal(t, []string{"initialize", "execute", "close"}, events)
}

func TestIndependentOptionsInitializationRollback(t *testing.T) {
	type appOptions struct {
		Token string `flag:"token" required:"true"`
	}
	type childOptions struct {
		ID string `argument:"id" required:"true"`
	}
	failure := errors.New("child initialization failed")
	var events []string
	app := NewApplication[appOptions]().
		WithName("app").
		WithInitializeHandler(func(_ context.Context, options appOptions) error {
			assert.Equal(t, "token-value", options.Token)
			events = append(events, "initialize app")
			return nil
		}).
		WithCloseHandler(func(options appOptions) error {
			assert.Equal(t, "token-value", options.Token)
			events = append(events, "close app")
			return nil
		}).
		WithCommand(NewCommand[*childOptions]().
			WithName("child").
			WithInitializeHandler(func(_ context.Context, options *childOptions) error {
				assert.Equal(t, "id-value", options.ID)
				events = append(events, "initialize child")
				return failure
			}).
			WithCloseHandler(func(*childOptions) error {
				events = append(events, "close child")
				return nil
			}),
		).
		MustCreate()
	invocation, err := app.Parse(context.Background(), nil, "--token=token-value", "child", "id-value")
	require.ErrorIs(t, err, failure)
	assert.Nil(t, invocation)
	assert.Equal(t, []string{"initialize app", "initialize child", "close child", "close app"}, events)
}
