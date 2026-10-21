package cli

import (
	"context"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompletionReadsPositionalDependenciesWithoutInitialization(t *testing.T) {
	var value codec.CompletionValue
	app := NewSimpleApplication("tool").
		WithArgument(NewArgument(codec.NewString(), "region").WithDefault("east")).
		WithFlag(NewFlag(codec.WithCompletionHandler(codec.NewString(), func(ctx context.Context, input codec.CompletionInput) (codec.CompletionResult, error) {
			var found bool
			var err error
			value, found, err = input.Effective.Argument(nil, "region")
			require.NoError(t, err)
			require.True(t, found)
			_, found, err = input.Effective.Argument(ctx, "absent")
			require.NoError(t, err)
			assert.False(t, found)
			_, found, err = input.Effective.Flag(nil, "absent")
			require.NoError(t, err)
			assert.False(t, found)
			canceled, cancel := context.WithCancel(ctx)
			cancel()
			_, _, err = input.Effective.Argument(canceled, "region")
			require.ErrorIs(t, err, context.Canceled)
			_, _, err = input.Effective.Flag(canceled, "absent")
			require.ErrorIs(t, err, context.Canceled)
			return codec.CompletionResult{Handled: true, Candidates: []codec.CompletionCandidate{{Value: "result"}}}, nil
		}), "output")).
		WithInitializeHandler(func(context.Context, NoOptions) error { panic("completion must not initialize options") }).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	_, err := app.Complete(nil, nil, completionAtEnd("--output", ""))
	require.NoError(t, err)
	assert.Equal(t, codec.CompletionDefault, value.Source)
	assert.Equal(t, []string{"east"}, value.Values)
	request := CompletionRequest{Words: []string{"--output", "", "west"}, Word: 1}
	_, err = app.Complete(nil, nil, request)
	require.NoError(t, err)
	assert.Equal(t, codec.CompletionCLI, value.Source)
	assert.Equal(t, []string{"west"}, value.Values)
	value.Values[0] = "caller mutation"
	_, err = app.Complete(nil, nil, request)
	require.NoError(t, err)
	assert.Equal(t, []string{"west"}, value.Values)
}
