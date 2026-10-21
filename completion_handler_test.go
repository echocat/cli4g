package cli

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompletionHandlersOverrideDelegateAndCaptureDefinitions(t *testing.T) {
	codecCalls, flagCalls, argumentCalls := 0, 0, 0
	source := codec.WithCompletionHandler(codec.NewString(), func(context.Context, codec.CompletionInput) (codec.CompletionResult, error) {
		codecCalls++
		return codec.CompletionResult{Handled: true, Candidates: []codec.CompletionCandidate{{Value: "codec-choice"}}}, nil
	})
	flag := NewFlag(source, "region").WithCompletionHandler(func(_ context.Context, input codec.CompletionInput) (codec.CompletionResult, error) {
		flagCalls++
		assert.Equal(t, "region", input.Target.Name)
		return codec.CompletionResult{Handled: true, Candidates: []codec.CompletionCandidate{{Value: "flag-choice"}}}, nil
	})
	argument := NewArgument(source, "target").WithCompletionHandler(func(_ context.Context, input codec.CompletionInput) (codec.CompletionResult, error) {
		argumentCalls++
		assert.Equal(t, "target", input.Target.Name)
		return codec.CompletionResult{}, nil
	})
	app := NewSimpleApplication("tool").WithCompletion(true).WithErrorHandlerEnabled(false).
		WithHandler(func(context.Context, NoOptions) error { return nil }).
		WithFlag(flag).WithArgument(argument).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	assert.Zero(t, codecCalls+flagCalls+argumentCalls)
	// Mutating builders does not change the compiled callbacks.
	flag.SetCompletionHandler(nil)
	argument.SetCompletionHandler(nil)
	definition, found := app.Flag("region")
	require.True(t, found)
	assert.NotNil(t, definition.CompletionHandler())
	argumentDefinition, found := app.Argument(0)
	require.True(t, found)
	assert.NotNil(t, argumentDefinition.CompletionHandler())
	result, err := app.Complete(nil, nil, completionAtEnd("--region", ""))
	require.NoError(t, err)
	assert.Equal(t, []string{"flag-choice"}, completionTexts(result))
	assert.Equal(t, 1, flagCalls)
	assert.Zero(t, codecCalls)
	result, err = app.Complete(nil, nil, completionAtEnd("codec"))
	require.NoError(t, err)
	assert.Equal(t, []string{"codec-choice"}, completionTexts(result))
	assert.Equal(t, 1, argumentCalls)
	assert.Equal(t, 1, codecCalls)
	_, err = app.Complete(nil, nil, completionAtEnd("--reg"))
	require.NoError(t, err)
	var output bytes.Buffer
	require.NoError(t, app.ParseAndExecute(ContextWithStdio(nil, nil, &output, &output), nil, "--help"))
	invocation, err := app.Parse(nil, nil, "--region", "actual-region", "actual-target")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, value := invocation.Flag("region")
	assert.Equal(t, "actual-region", value)
	assert.Equal(t, 1, flagCalls)
	assert.Equal(t, 1, argumentCalls)
	assert.Equal(t, 1, codecCalls)
}

func TestCompletionHandlersFallThroughToEnumsAndPreserveParsing(t *testing.T) {
	formats, err := codec.NewEnum(codec.NewString(), "json", "yaml")
	require.NoError(t, err)
	delegations := 0
	delegate := func(context.Context, codec.CompletionInput) (codec.CompletionResult, error) {
		delegations++
		return codec.CompletionResult{}, nil
	}
	app := NewSimpleApplication("tool").WithErrorHandlerEnabled(false).
		WithFlag(NewFlag(codec.WithCompletionHandler(formats, delegate), "format").WithCompletionHandler(delegate)).
		WithFlag(NewFlag(codec.WithCompletionHandler(codec.NewBool(), delegate), "verbose")).
		WithFlag(NewFlag(codec.WithCompletionHandler(codec.NewStrings(), delegate), "label")).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	assert.Zero(t, delegations)
	result, err := app.Complete(nil, nil, completionAtEnd("--format", "j"))
	require.NoError(t, err)
	assert.Equal(t, []string{"json"}, completionTexts(result))
	assert.Equal(t, 2, delegations)
	invocation, err := app.Parse(nil, nil, "--verbose", "--label", "a", "--label", "b", "--format", "yaml")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, verbose := invocation.Flag("verbose")
	assert.Equal(t, true, verbose)
	_, labels := invocation.Flag("label")
	assert.Equal(t, []string{"a", "b"}, labels)
	assert.Equal(t, 2, delegations)
}

func TestCompletionHandlerErrorsAndAuthoritativeEmptyAnswers(t *testing.T) {
	failure := errors.New("service unavailable")
	codecCalled := false
	source := codec.WithCompletionHandler(codec.NewString(), func(context.Context, codec.CompletionInput) (codec.CompletionResult, error) {
		codecCalled = true
		return codec.CompletionResult{Handled: true, Files: true}, nil
	})
	app := NewSimpleApplication("tool").WithFlag(NewFlag(source, "value").WithCompletionHandler(
		func(_ context.Context, input codec.CompletionInput) (codec.CompletionResult, error) {
			if input.Prefix == "error" {
				return codec.CompletionResult{}, failure
			}
			return codec.CompletionResult{Handled: true}, nil
		})).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	result, err := app.Complete(nil, nil, completionAtEnd("--value", ""))
	require.NoError(t, err)
	assert.True(t, result.Handled)
	assert.False(t, result.Files)
	assert.Empty(t, result.Candidates)
	_, err = app.Complete(nil, nil, completionAtEnd("--value", "error"))
	require.ErrorIs(t, err, failure)
	assert.False(t, codecCalled)
}
