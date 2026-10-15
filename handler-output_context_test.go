package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuiltInHandlersSelectOutputForEachContext(t *testing.T) {
	for _, fallback := range []bool{false, true} {
		failure := errors.New("execution failed")
		app := NewSimpleApplication().WithName("context-output").WithFallbackToStdio(fallback).
			WithErrorHandler(NewRespondErrorHandler().MustCreate()).
			WithHandler(func(context.Context, NoOptions) error { return failure }).MustCreate()
		defer func() { assert.NoError(t, app.Close()) }()
		for range 2 {
			var output, stdout bytes.Buffer
			ctx := codec.ContextWithStdio(nil, nil, &stdout, &output)
			require.NoError(t, app.ParseAndExecute(ctx, nil, "--help"))
			assert.Contains(t, output.String(), "Usage:")
			output.Reset()
			_, err := app.Parse(ctx, nil, "--unknown")
			require.ErrorIs(t, err, ErrValidation)
			assert.Contains(t, output.String(), "Error: unknown flag --unknown\n\n")
			assert.Contains(t, output.String(), "Usage:")
			output.Reset()
			assert.ErrorIs(t, app.ParseAndExecute(ctx, nil), failure)
			assert.Equal(t, "Error: execution failed\n", output.String())
			output.Reset()
			canceled, cancel := context.WithCancel(ctx)
			cancel()
			_, err = app.Parse(canceled, nil)
			assert.ErrorIs(t, err, context.Canceled)
			assert.Equal(t, "Error: context canceled\n", output.String(), "reporting preserves context output after cancellation")
			assert.Empty(t, stdout.String(), "help and errors use stderr, not stdout")
		}
	}
}

func TestBuiltInHandlersExplicitOutputOverridesContext(t *testing.T) {
	var configured, contextual bytes.Buffer
	helpRequirement := NewTemplatedHelpHandler().WithApplicationTemplate("HELP\n").WithOutput(&configured)
	responseRequirement := NewRespondErrorHandler().WithOutput(&configured)
	app := NewSimpleApplication().WithName("context-output").WithFallbackToStdio(false).
		WithHelpHandler(helpRequirement.MustCreate()).WithErrorHandler(responseRequirement.MustCreate()).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	helpRequirement.SetOutput(nil)
	responseRequirement.SetOutput(nil)
	for _, ctx := range []context.Context{
		codec.ContextWithStdio(nil, nil, &contextual, nil),
		codec.ContextWithStdio(nil, nil, nil, nil),
	} {
		configured.Reset()
		require.NoError(t, app.ParseAndExecute(ctx, nil, "--help"))
		assert.Equal(t, "HELP\n", configured.String())
		configured.Reset()
		_, err := app.Parse(ctx, nil, "--unknown")
		require.ErrorIs(t, err, ErrValidation)
		assert.Equal(t, "Error: unknown flag --unknown\n\nHELP\n", configured.String())
		assert.Empty(t, contextual.String())
	}
}

func TestBuiltInHandlersSerializeSharedContextOutput(t *testing.T) {
	var output bytes.Buffer
	ctx := codec.ContextWithStdio(nil, nil, &output, nil)
	var apps []*Application[NoOptions]
	for range 2 {
		app := NewSimpleApplication().WithName("app").WithFallbackToStdio(false).
			WithHelpHandler(NewTemplatedHelpHandler().WithApplicationTemplate("HELP\n").MustCreate()).
			WithErrorHandler(NewRespondErrorHandler().MustCreate()).MustCreate()
		defer func() { assert.NoError(t, app.Close()) }()
		apps = append(apps, app)
	}
	var workers sync.WaitGroup
	for i := range 32 {
		app := apps[i%len(apps)]
		workers.Go(func() { assert.NoError(t, app.ParseAndExecute(ctx, nil, "--help")) })
		workers.Go(func() {
			_, err := app.Parse(ctx, nil, "--unknown")
			assert.ErrorIs(t, err, ErrValidation)
		})
	}
	workers.Wait()
	assert.Equal(t, 64, strings.Count(output.String(), "HELP\n"))
	assert.Equal(t, 32, strings.Count(output.String(), "Error: unknown flag --unknown\n\n"))
}

func TestBuiltInHandlersWithoutOutputProcess(t *testing.T) {
	mode := os.Getenv("CLI4G_SILENT_HANDLERS_TEST")
	if mode != "" {
		DefaultErrorHandler = subprocessDefaultErrorHandler
		ctx := context.Background()
		root := NewSimpleApplication().WithName("silent").WithFallbackToStdio(false)
		if strings.HasPrefix(mode, "explicit-nil-") {
			root.WithFallbackToStdio(true)
			ctx = codec.ContextWithStdio(ctx, nil, nil, nil)
		}
		switch {
		case mode == "standalone":
			help := NewTemplatedHelpHandler().WithApplicationTemplate("HELP\n").MustCreate()
			app := root.MustCreate()
			invocation := app.MustParse(nil, nil, "--help")
			require.NoError(t, help(nil, invocation))
			response := NewRespondErrorHandler().WithHelpHandler(help).MustCreate()
			primary := NewValidationErrorf("bad input")
			require.ErrorIs(t, response(nil, invocation, primary), primary)
			require.NoError(t, invocation.Close())
			require.NoError(t, app.Close())
		case strings.HasSuffix(mode, "error"):
			root.MustCreateParseAndExecute(ctx, nil, "--unknown")
			t.Fatal("default error response must still exit")
		case mode == "discard":
			root.WithHelpHandler(NewTemplatedHelpHandler().WithApplicationTemplate("HELP\n").WithOutput(io.Discard).MustCreate()).
				MustCreateParseAndExecute(codec.ContextWithFallbackToStdio(ctx, true), nil, "--help")
		default:
			root.MustCreateParseAndExecute(ctx, nil, "--help")
		}
		os.Exit(0)
	}
	for _, mode := range []string{"help", "error", "explicit-nil-help", "explicit-nil-error", "standalone", "discard"} {
		t.Run(mode, func(t *testing.T) {
			command := exec.Command(os.Args[0], "-test.run=^TestBuiltInHandlersWithoutOutputProcess$")
			command.Env = append(os.Environ(), "CLI4G_SILENT_HANDLERS_TEST="+mode)
			var stdout, stderr bytes.Buffer
			command.Stdout, command.Stderr = &stdout, &stderr
			err := command.Run()
			if strings.HasSuffix(mode, "error") {
				var exit *exec.ExitError
				require.ErrorAs(t, err, &exit)
				assert.Equal(t, 2, exit.ExitCode())
			} else {
				require.NoError(t, err, "stdout=%s stderr=%s", stdout.String(), stderr.String())
			}
			assert.Empty(t, stdout.String())
			assert.Empty(t, stderr.String())
		})
	}
}
