package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type factoryValueOptions struct {
	Name   string `flag:"name"`
	Closes int
}

func (options *factoryValueOptions) Initialize(context.Context) error {
	options.Name += " initialized"
	return nil
}
func (options *factoryValueOptions) Close() error { options.Closes++; return nil }

func TestFactoryStructOptionsKeepCanonicalPointerLifecycle(t *testing.T) {
	app, err := NewApplication[factoryValueOptions]().WithName("app").
		WithCreateOptionsHandler(func(ctx context.Context) (factoryValueOptions, error) {
			assert.NotNil(t, ctx)
			return factoryValueOptions{Name: "factory"}, nil
		}).Create()
	require.NoError(t, err)
	invocation, err := app.Parse(nil, nil)
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, value := invocation.Application()
	assert.Equal(t, "factory initialized", value.Name)
	require.NoError(t, invocation.Close())
	_, value = invocation.Application()
	assert.Equal(t, 1, value.Closes)
}

func TestFactoryValuesDoNotSatisfyRequiredFlagsOrArguments(t *testing.T) {
	type options struct {
		Flag string `flag:"flag" required:"true"`
		Arg  string `argument:"arg" required:"true"`
	}
	app, err := NewApplication[*options]().WithName("app").
		WithCreateOptionsHandler(func(context.Context) (*options, error) {
			return &options{Flag: "factory", Arg: "factory"}, nil
		}).Create()
	require.NoError(t, err)
	invocation, err := app.Parse(context.Background(), nil)
	assert.Nil(t, invocation)
	require.ErrorContains(t, err, "required flag --flag")
	invocation, err = app.Parse(context.Background(), nil, "--flag=CLI")
	assert.Nil(t, invocation)
	require.ErrorContains(t, err, "required argument arg")
	invocation, err = app.Parse(context.Background(), nil, "--flag=CLI", "argument")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, value := invocation.Application()
	assert.Equal(t, &options{Flag: "CLI", Arg: "argument"}, value)
}

type factoryOwnedOptions struct {
	name     string
	closed   *[]string
	closeErr error
}

func (options *factoryOwnedOptions) Close() error {
	*options.closed = append(*options.closed, options.name)
	return options.closeErr
}

func TestFactoryFailureClosesReturnedOptionsAndEarlierScopesInReverse(t *testing.T) {
	failure := errors.New("factory failed")
	cleanupFailure := errors.New("cleanup failed")
	var closed []string
	var unselectedCalls int
	app := NewApplication[*factoryOwnedOptions]().
		WithName("app").
		WithCreateOptionsHandler(func(context.Context) (*factoryOwnedOptions, error) {
			return &factoryOwnedOptions{name: "app", closed: &closed}, nil
		}).
		WithCommand(
			NewCommand[*factoryOwnedOptions]().
				WithName("child").
				WithCreateOptionsHandler(func(context.Context) (*factoryOwnedOptions, error) {
					return &factoryOwnedOptions{name: "child", closed: &closed, closeErr: cleanupFailure}, failure
				}),
			NewSimpleCommand().
				WithName("unused").
				WithCreateOptionsHandler(func(context.Context) (NoOptions, error) {
					unselectedCalls++
					return NoOptions{}, nil
				}),
		).
		MustCreate()
	invocation, err := app.Parse(context.Background(), nil, "child")
	assert.Nil(t, invocation)
	require.ErrorIs(t, err, failure)
	require.ErrorIs(t, err, cleanupFailure)
	require.ErrorContains(t, err, "child: create options")
	assert.Equal(t, []string{"child", "app"}, closed)
	assert.Zero(t, unselectedCalls)
}

func TestFactoryNilPointersAreRejectedWithoutCallingNilCloser(t *testing.T) {
	for _, failure := range []error{nil, errors.New("factory failed")} {
		app, err := NewApplication[*factoryOwnedOptions]().WithName("app").
			WithCreateOptionsHandler(func(context.Context) (*factoryOwnedOptions, error) {
				return nil, failure
			}).Create()
		require.NoError(t, err)
		invocation, err := app.Parse(context.Background(), nil)
		assert.Nil(t, invocation)
		if failure != nil {
			require.ErrorIs(t, err, failure)
		} else {
			require.ErrorContains(t, err, "options factory returned a nil pointer")
		}
	}
}

func TestFactoryCancellationRetainsOwnedOptionsAndSkipsLaterFactories(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var closed []string
	laterCalled := false
	app := NewApplication[*factoryOwnedOptions]().WithName("app").
		WithCreateOptionsHandler(func(received context.Context) (*factoryOwnedOptions, error) {
			assert.Equal(t, ctx.Done(), received.Done())
			assert.NoError(t, received.Err())
			cancel()
			assert.ErrorIs(t, received.Err(), context.Canceled)
			return &factoryOwnedOptions{name: "app", closed: &closed}, nil
		}).
		WithCommand(NewSimpleCommand().
			WithName("child").
			WithCreateOptionsHandler(func(context.Context) (NoOptions, error) {
				laterCalled = true
				return NoOptions{}, nil
			}),
		).MustCreate()
	invocation, err := app.Parse(ctx, nil, "child")
	assert.Nil(t, invocation)
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, []string{"app"}, closed)
	assert.False(t, laterCalled)
}

func TestOptionsFactoryIsCapturedAtCreateAndDoesNotReadLaterRequirementChanges(t *testing.T) {
	requirement := NewApplication[factoryValueOptions]().WithName("app")
	requirement.SetCreateOptionsHandler(func(context.Context) (factoryValueOptions, error) {
		return factoryValueOptions{Name: "original"}, nil
	})
	app, err := requirement.Create()
	require.NoError(t, err)
	requirement.SetCreateOptionsHandler(func(context.Context) (factoryValueOptions, error) {
		return factoryValueOptions{Name: "replacement"}, nil
	})
	invocation, err := app.Parse(context.Background(), nil)
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, value := invocation.Application()
	assert.Equal(t, "original initialized", value.Name)
}
