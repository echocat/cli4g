package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithCompletionOnlyAssignsAndClearsConfiguration(t *testing.T) {
	previousCommand, previousRequest := DefaultCompletionCommand, DefaultCompletionRequestCommandName
	t.Cleanup(func() {
		DefaultCompletionCommand, DefaultCompletionRequestCommandName = previousCommand, previousRequest
	})
	defaultCommand := NewCompletionCommand().WithName("global-shells")
	DefaultCompletionCommand, DefaultCompletionRequestCommandName = defaultCommand, "__global_suggest"
	build := NewSimpleApplication("tool")
	assert.Nil(t, build.CompletionCommand)
	assert.Empty(t, build.CompletionRequestCommandName)
	build.WithCompletion(true)
	assert.Same(t, defaultCommand, build.CompletionCommand)
	assert.Equal(t, "__global_suggest", build.CompletionRequestCommandName)
	assert.Empty(t, build.Commands, "shortcut must not append permanently to Commands")
	// The shortcut captures the defaults at call time, not during Create.
	DefaultCompletionCommand, DefaultCompletionRequestCommandName = nil, ""
	app := build.MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	assert.Equal(t, "global-shells", app.CompletionCommandName())
	assert.Equal(t, "__global_suggest", app.CompletionRequestCommandName())
	build.WithCompletionCommand(NewSimpleCommand("custom")).WithCompletionRequestCommandName("__custom").WithCompletion(false)
	assert.Nil(t, build.CompletionCommand)
	assert.Empty(t, build.CompletionRequestCommandName)
	off := build.MustCreate()
	defer func() { assert.NoError(t, off.Close()) }()
	assert.Empty(t, slices.Collect(off.Commands()))
	assert.Nil(t, off.CompletionCommand())
	assert.Empty(t, off.CompletionRequestCommandName())
	// True also assigns deliberately empty defaults without inventing fallbacks.
	build.WithCompletion(true)
	assert.Nil(t, build.CompletionCommand)
	assert.Empty(t, build.CompletionRequestCommandName)
}

func TestDefaultCompletionCommandIsAnOrdinaryCompiledCommand(t *testing.T) {
	var events []string
	definition := NewCompletionCommand().WithName("shell-setup").WithAliases("shells")
	build := NewSimpleApplication("tool").WithCompletionCommand(definition).
		WithCompletionRequestCommandName("__suggest").WithErrorHandlerEnabled(false).
		WithInitializeHandler(func(context.Context, NoOptions) error { events = append(events, "initialize"); return nil }).
		WithCloseHandler(func(NoOptions) error { events = append(events, "close"); return nil })
	app := build.MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	assert.Empty(t, build.Commands)
	command, found := app.Command("shells")
	require.True(t, found)
	assert.Same(t, command, app.CompletionCommand())
	typed, found := FindCommand[NoOptions](app, "shell-setup")
	require.True(t, found)
	assert.Same(t, command, typed)
	assert.Len(t, slices.Collect(app.Commands()), 1)
	_, found = app.Command("__suggest")
	assert.False(t, found, "request entry is not registered as a command")
	definition.WithName("changed")
	assert.Equal(t, "shell-setup", app.CompletionCommandName())
	for _, shell := range []CompletionShell{CompletionBash, CompletionNushell, CompletionPowerShell} {
		var output, diagnostics bytes.Buffer
		require.NoError(t, app.ParseAndExecute(ContextWithStdio(nil, nil, &output, &diagnostics), nil, "shells", string(shell)))
		assert.Contains(t, output.String(), "__suggest")
		assert.Empty(t, diagnostics.String(), "ancestor group handler must not print help for a selected child")
	}
	assert.Equal(t, []string{"initialize", "close", "initialize", "close", "initialize", "close"}, events)
	var help bytes.Buffer
	require.NoError(t, app.ParseAndExecute(ContextWithStdio(nil, nil, &help, &help), nil, "shell-setup", "--help"))
	assert.Contains(t, help.String(), "tool shell-setup")
	assert.Contains(t, help.String(), "install")
	assert.Len(t, events, 6, "ordinary --help skips initialization")
	var output bytes.Buffer
	directory := t.TempDir()
	require.NoError(t, app.ParseAndExecute(ContextWithStdio(nil, nil, &output, &output), nil, "shell-setup", "install", "bash", directory))
	installed, err := os.ReadFile(filepath.Join(directory, "tool.bash"))
	require.NoError(t, err)
	assert.Contains(t, string(installed), "__suggest")
}

func TestCompletionCommandsObeyOrdinaryRequirements(t *testing.T) {
	type options struct {
		Token string `flag:"token" required:"true"`
	}
	created := 0
	app := NewApplication[options]("tool").WithCompletion(true).WithErrorHandlerEnabled(false).
		WithCreateOptionsHandler(func(context.Context) (options, error) { created++; return options{}, nil }).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	ctx := ContextWithStdio(nil, nil, nil, nil)
	require.ErrorIs(t, app.ParseAndExecute(ctx, nil, "completion", "bash"), ErrValidation)
	var output bytes.Buffer
	require.NoError(t, app.ParseAndExecute(ContextWithStdio(nil, nil, &output, &output), nil, "--token", "provided", "completion", "bash"))
	assert.Contains(t, output.String(), "__complete")
	assert.Equal(t, 2, created)
	output.Reset()
	request := strings.NewReader(`{"words":["--to"],"word":0,"offset":4}`)
	require.NoError(t, app.ParseAndExecute(ContextWithStdio(nil, request, &output, &output), nil, "__complete", "1", "nu"))
	assert.Equal(t, 2, created, "request entry alone bypasses options and required input")
	var result codec.CompletionResult
	require.NoError(t, json.Unmarshal(output.Bytes(), &result))
	assert.Equal(t, []string{"--token"}, completionTexts(result))
	_, err := NewApplication[struct {
		Name string `argument:"name" required:"true"`
	}]("tool").WithCompletion(true).Create()
	require.ErrorContains(t, err, "required positional")
	_, err = NewSimpleApplication("tool").WithCompletion(true).WithMaxCommandDepth(0).Create()
	require.ErrorContains(t, err, "maximum command depth")
}

func TestCompletionCommandAndRequestAreIndependentAndReplaceable(t *testing.T) {
	type customOptions struct {
		Value string `argument:"value" required:"true"`
	}
	var selected string
	custom := NewCommand[customOptions]("custom-shells").WithHandler(func(_ context.Context, options customOptions) error {
		selected = options.Value
		return nil
	})
	app := NewSimpleApplication("tool").WithCompletionCommand(custom).WithErrorHandlerEnabled(false).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	require.NoError(t, app.ParseAndExecute(nil, nil, "custom-shells", "ordinary"))
	assert.Equal(t, "ordinary", selected)
	_, found := FindCommand[customOptions](app, "custom-shells")
	assert.True(t, found)
	assert.Empty(t, app.CompletionRequestCommandName())
	_, err := app.CompletionScript(CompletionBash)
	require.ErrorContains(t, err, "request entry is not configured")
	type rootOptions struct {
		Name string `argument:"name" required:"true"`
	}
	requestOnly := NewApplication[rootOptions]("tool").WithCompletionRequestCommandName("__suggest").
		WithMaxCommandDepth(0).WithErrorHandlerEnabled(false).MustCreate()
	defer func() { assert.NoError(t, requestOnly.Close()) }()
	assert.Empty(t, slices.Collect(requestOnly.Commands()))
	assert.Nil(t, requestOnly.CompletionCommand())
	var output bytes.Buffer
	require.NoError(t, requestOnly.ParseAndExecute(ContextWithStdio(nil, strings.NewReader(`{}`), &output, &output), nil, "__suggest", "1", "nu"))
	assert.Contains(t, output.String(), `"handled":true`)
	plain := NewSimpleApplication("tool").WithCommand(NewSimpleCommand("__complete")).MustCreate()
	defer func() { assert.NoError(t, plain.Close()) }()
	_, found = plain.Command("__complete")
	assert.True(t, found, "an empty request name does not reserve any name")
}

func TestCompletionRequestNameAndNormalDuplicateChecks(t *testing.T) {
	for _, invalid := range []string{"--complete", "shell request", "a\nb", "a/b", "a\\b", "a=b", "a\x00b"} {
		_, err := NewSimpleApplication("tool").WithCompletionRequestCommandName(invalid).Create()
		require.Error(t, err)
	}
	for _, command := range []*CommandRequirement[NoOptions]{
		NewSimpleCommand("__complete"), NewSimpleCommand("other").WithAliases("__complete"), NewSimpleCommand("__complete").WithProfiles("inactive"),
	} {
		_, err := NewSimpleApplication("tool").WithCompletion(true).WithCommand(command).Create()
		require.ErrorContains(t, err, "reserved for completion requests")
	}
	_, err := NewSimpleApplication("tool").WithCompletion(true).WithCommand(NewSimpleCommand("completion")).Create()
	require.ErrorContains(t, err, "duplicate or invalid command name")
	_, err = NewSimpleApplication("tool").WithCompletionCommand(NewSimpleCommand("__suggest")).WithCompletionRequestCommandName("__suggest").Create()
	require.ErrorContains(t, err, "reserved for completion requests")
}

func TestNormalCompletionCommandCompletesItsOwnSchema(t *testing.T) {
	app := NewSimpleApplication("tool").WithCompletion(true).
		WithCompletionCommand(NewCompletionCommand().WithName("shell-setup")).WithCompletionRequestCommandName("__suggest").MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	for _, test := range []struct {
		words []string
		want  []string
		dirs  bool
	}{
		{[]string{"shell-s"}, []string{"shell-setup"}, false},
		{[]string{"__s"}, []string{}, false},
		{[]string{"shell-setup", ""}, []string{"bash", "install", "nu", "powershell"}, false},
		{[]string{"shell-setup", "install", "n"}, []string{"nu"}, false},
		{[]string{"shell-setup", "install", "bash", "dir"}, []string{}, true},
	} {
		result, err := app.Complete(nil, nil, completionAtEnd(test.words...))
		require.NoError(t, err)
		assert.Equal(t, test.want, completionTexts(result))
		assert.Equal(t, test.dirs, result.Directories)
	}
	renamed := NewSimpleApplication("tool").WithCompletion(true).
		WithCompletionCommand(NewCompletionCommand().WithName("shell-admin")).WithCompletionRequestCommandName("__suggest").MustCreate()
	defer func() { assert.NoError(t, renamed.Close()) }()
	for _, shell := range []CompletionShell{CompletionBash, CompletionNushell, CompletionPowerShell} {
		before, err := app.CompletionScript(shell)
		require.NoError(t, err)
		after, err := renamed.CompletionScript(shell)
		require.NoError(t, err)
		assert.Equal(t, before, after)
	}
}
