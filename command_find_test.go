package cli

import (
	"context"
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type independentRootOptions struct {
	Verbose bool `flag:"verbose" short:"v"`
}

type independentParentOptions struct {
	Config string `flag:"config"`
}

type independentServeOptions struct {
	Port uint16 `flag:"port" required:"true"`
}

type independentInspectOptions struct {
	ID string `argument:"id" required:"true"`
}

func TestIndependentOptionsAndTypedCommandHandles(t *testing.T) {
	var events []string
	serve := NewCommand[*independentServeOptions]("serve").
		WithAliases("s").
		WithHandler(func(_ context.Context, options *independentServeOptions) error {
			assert.Equal(t, uint16(8080), options.Port)
			events = append(events, "serve")
			return nil
		})
	appRequirement := NewApplication[*independentRootOptions]("app").
		WithHandler(func(_ context.Context, options *independentRootOptions) error {
			assert.True(t, options.Verbose)
			events = append(events, "app")
			return nil
		}).
		WithCommand(
			NewCommand[independentParentOptions]("group").
				WithHandler(func(_ context.Context, options independentParentOptions) error {
					assert.Equal(t, "config-value", options.Config)
					events = append(events, "group")
					return nil
				}).
				WithCommand(serve),
			NewCommand[independentInspectOptions]("inspect"),
		)
	assert.Equal(t, reflect.TypeFor[*independentRootOptions](), appRequirement.OptionsType())
	assert.Equal(t, reflect.TypeFor[*independentServeOptions](), serve.OptionsType())
	app, err := appRequirement.Create()
	require.NoError(t, err)
	assert.Equal(t, reflect.TypeFor[*independentRootOptions](), app.OptionsType())

	// A builder change after Create must not replace the compiled typed adapter.
	serve.WithHandler(func(context.Context, *independentServeOptions) error {
		assert.Fail(t, "compiled command called a modified builder handler")
		return nil
	})
	ctx := context.Background()
	invocation, err := app.Parse(ctx, nil, "-v", "group", "--config=config-value", "s", "--port=8080")
	require.NoError(t, err)
	_, rootOptions := invocation.Application()
	assert.True(t, rootOptions.Verbose)

	parent, found := FindCommand[independentParentOptions](app, "group")
	require.True(t, found)
	parentOptions, found := parent.OptionsFrom(invocation)
	require.True(t, found)
	assert.Equal(t, "config-value", parentOptions.Config)
	command, found := FindCommand[*independentServeOptions](app, "group", "serve")
	require.True(t, found)
	alias, found := FindCommand[*independentServeOptions](app, "group", "s")
	require.True(t, found)
	assert.Same(t, command, alias)
	serveOptions, found := command.OptionsFrom(invocation)
	require.True(t, found)
	assert.Equal(t, uint16(8080), serveOptions.Port)
	_, found = FindCommand[independentServeOptions](app, "group", "serve")
	assert.False(t, found, "pointer and value options must not be interchangeable")

	require.NoError(t, invocation.Execute(ctx))
	assert.Equal(t, []string{"app", "group", "serve"}, events)
	require.NoError(t, invocation.Close())

	inspectInvocation, err := app.Parse(ctx, nil, "inspect", "id-value")
	require.NoError(t, err)
	inspect, found := FindCommand[independentInspectOptions](app, "inspect")
	require.True(t, found)
	inspectOptions, found := inspect.OptionsFrom(inspectInvocation)
	require.True(t, found)
	assert.Equal(t, "id-value", inspectOptions.ID)
	_, found = command.OptionsFrom(inspectInvocation)
	assert.False(t, found, "a command outside the selected path has no options")
	require.NoError(t, inspectInvocation.Close())
}

func TestTypedHandleDoesNotAcceptAnotherCompiledCommand(t *testing.T) {
	app := NewSimpleApplication("app").
		WithCommand(NewCommand[*independentServeOptions]("serve")).
		MustCreate()
	other := NewSimpleApplication("other").
		WithCommand(NewCommand[*independentServeOptions]("serve")).
		MustCreate()
	command, found := FindCommand[*independentServeOptions](app, "serve")
	require.True(t, found)
	invocation, err := other.Parse(context.Background(), nil, "serve", "--port=80")
	require.NoError(t, err)
	_, found = command.OptionsFrom(invocation)
	assert.False(t, found, "options are associated with the compiled command identity")
	require.NoError(t, invocation.Close())
}

func TestIndependentCommandOptionsConcurrentParsing(t *testing.T) {
	app := NewApplication[*independentRootOptions]("app").
		WithCommand(NewCommand[*independentServeOptions]("serve")).
		MustCreate()
	command, found := FindCommand[*independentServeOptions](app, "serve")
	require.True(t, found)
	var wg sync.WaitGroup
	for range 40 {
		wg.Go(func() {
			ctx := context.Background()
			invocation, err := app.Parse(ctx, nil, "--verbose", "serve", "--port=80")
			if !assert.NoError(t, err) {
				return
			}
			defer func() { assert.NoError(t, invocation.Close()) }()
			_, rootOptions := invocation.Application()
			assert.True(t, rootOptions.Verbose)
			options, found := command.OptionsFrom(invocation)
			if assert.True(t, found) && assert.NotNil(t, options) {
				assert.Equal(t, uint16(80), options.Port)
				options.Port = 123
			}
		})
	}
	wg.Wait()
	invocation, err := app.Parse(context.Background(), nil, "serve", "--port=80")
	require.NoError(t, err)
	options, found := command.OptionsFrom(invocation)
	require.True(t, found)
	assert.Equal(t, uint16(80), options.Port)
	require.NoError(t, invocation.Close())
}
