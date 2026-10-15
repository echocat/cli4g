package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stdioObservingCodec struct {
	codec.Codec[string, string]
	inspect func(context.Context, string)
}

func (this stdioObservingCodec) ToRaw(ctx context.Context) codec.Codec[any, any] {
	return codec.Raw(ctx, this)
}

func (this stdioObservingCodec) ParseRequirement(ctx context.Context, usage codec.Usage, args ...string) (string, error) {
	this.inspect(ctx, "parse requirement")
	return this.Codec.ParseRequirement(ctx, usage, args...)
}

func (this stdioObservingCodec) Resolve(ctx context.Context, scope codec.Scope, requirement string) (string, error) {
	this.inspect(ctx, "resolve")
	return this.Codec.Resolve(ctx, scope, requirement)
}

func TestApplicationStdioThroughoutOperations(t *testing.T) {
	input, output, diagnostic := strings.NewReader("input"), new(bytes.Buffer), new(bytes.Buffer)
	for _, test := range []struct {
		name     string
		fallback bool
		ctx      context.Context
		stdin    io.Reader
		stdout   io.Writer
		stderr   io.Writer
	}{
		{"fallback", true, nil, os.Stdin, os.Stdout, os.Stderr},
		{"disabled", false, nil, nil, nil, nil},
		{"explicit-with-fallback", true, codec.ContextWithStdio(nil, input, output, diagnostic), input, output, diagnostic},
		{"explicit-without-fallback", false, codec.ContextWithStdio(nil, input, output, diagnostic), input, output, diagnostic},
		{"explicit-nil", true, codec.ContextWithStdio(nil, nil, nil, nil), nil, nil, nil},
		{"explicit-stdout-only", true, codec.ContextWithStdio(nil, nil, output, nil), nil, output, output},
		{"explicit-stderr-only", true, codec.ContextWithStdio(nil, nil, nil, diagnostic), nil, nil, diagnostic},
	} {
		t.Run(test.name, func(t *testing.T) {
			seen := make(map[string]int)
			inspect := func(ctx context.Context, phase string) {
				seen[phase]++
				stdin, stdout, stderr := codec.StdioOfContext(ctx)
				assert.Equal(t, test.stdin, stdin, phase)
				assert.Equal(t, test.stdout, stdout, phase)
				assert.Equal(t, test.stderr, stderr, phase)
			}
			create := func(ctx context.Context) (NoOptions, error) {
				inspect(ctx, "factory")
				return NoOptions{}, nil
			}
			validate := func(ctx context.Context, _ NoOptions) error { inspect(ctx, "validate"); return nil }
			initialize := func(ctx context.Context, _ NoOptions) error { inspect(ctx, "initialize"); return nil }
			execute := func(ctx context.Context, _ NoOptions) error { inspect(ctx, "execute"); return nil }
			observed := stdioObservingCodec{Codec: codec.NewString(), inspect: inspect}
			builder := NewSimpleApplication().WithName("stdio").WithFallbackToStdio(test.fallback).
				WithCreateOptionsHandler(create).WithValidateHandler(validate).WithInitializeHandler(initialize).WithHandler(execute).
				WithHelpHandler(func(ctx context.Context, _ InvocationLike) error { inspect(ctx, "help"); return nil }).
				WithErrorHandler(func(ctx context.Context, _ InvocationLike, _ error) error { inspect(ctx, "error"); return nil }).
				WithFlag(NewFlag(observed).WithName("value")).
				WithCommand(NewCommand[NoOptions]().WithName("child").WithCreateOptionsHandler(create).
					WithValidateHandler(validate).WithInitializeHandler(initialize).WithHandler(execute))
			app := builder.MustCreate()
			defer func() { assert.NoError(t, app.Close()) }()
			// Mutating the builder must not change the compiled fallback policy.
			builder.SetFallbackToStdio(!test.fallback)
			require.NoError(t, app.ParseAndExecute(test.ctx, nil, "--value", "x", "child"))
			for _, phase := range []string{"factory", "parse requirement", "resolve", "validate", "initialize", "execute"} {
				assert.Positive(t, seen[phase], phase)
			}
			require.NoError(t, app.ParseAndExecute(test.ctx, nil, "--help"))
			assert.Equal(t, 1, seen["help"])
			_, err := app.Parse(test.ctx, nil, "--unknown")
			require.Error(t, err)
			assert.Equal(t, 1, seen["error"])
			command, found := FindCommand[NoOptions](app, "child")
			require.True(t, found)
			require.NoError(t, app.Validate(test.ctx, NoOptions{}))
			require.NoError(t, app.Initialize(test.ctx, NoOptions{}))
			require.NoError(t, app.Execute(test.ctx, NoOptions{}))
			require.NoError(t, command.Validate(test.ctx, NoOptions{}))
			require.NoError(t, command.Initialize(test.ctx, NoOptions{}))
			require.NoError(t, command.Execute(test.ctx, NoOptions{}))
			invocation, err := app.Parse(test.ctx, nil, "child")
			require.NoError(t, err)
			require.NoError(t, invocation.Execute(test.ctx))
			require.NoError(t, invocation.Close())
		})
	}
}

func TestApplicationStdioDefaults(t *testing.T) {
	assert.True(t, NewApplication[NoOptions]().FallbackToStdio)
	assert.True(t, NewSimpleApplication().FallbackToStdio)
	assert.False(t, new(ApplicationRequirement[NoOptions]).FallbackToStdio)
}

func TestApplicationStdioExecutionErrorsKeepContextStreams(t *testing.T) {
	input, output := strings.NewReader("input"), new(bytes.Buffer)
	ctx := codec.ContextWithStdio(nil, input, output, nil)
	failure := errors.New("handler failed")
	responses := 0
	app := NewSimpleApplication().WithName("stdio").WithFallbackToStdio(false).
		WithHandler(func(context.Context, NoOptions) error { return failure }).
		WithErrorHandler(func(ctx context.Context, _ InvocationLike, err error) error {
			responses++
			assert.Same(t, input, codec.StdinOfContext(ctx))
			assert.Same(t, output, codec.StdoutOfContext(ctx))
			assert.ErrorIs(t, err, failure)
			return nil
		}).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	assert.ErrorIs(t, app.ParseAndExecute(ctx, nil), failure)
	invocation, err := app.Parse(ctx, nil)
	require.NoError(t, err)
	assert.ErrorIs(t, invocation.Execute(ctx), failure)
	require.NoError(t, invocation.Close())
	assert.ErrorIs(t, app.Execute(ctx, NoOptions{}), failure)
	assert.Equal(t, 3, responses)
}

func TestApplicationStdioConcurrentInvocationsAreIsolated(t *testing.T) {
	app := NewSimpleApplication().WithName("stdio").WithFallbackToStdio(false).WithErrorHandlerEnabled(false).
		WithHandler(func(ctx context.Context, _ NoOptions) error {
			stdin, stdout, _ := codec.StdioOfContext(ctx)
			_, err := io.Copy(stdout, stdin)
			return err
		}).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	var workers sync.WaitGroup
	for _, content := range []string{"first", "second", "third", "fourth"} {
		workers.Go(func() {
			var output bytes.Buffer
			ctx := codec.ContextWithStdio(nil, strings.NewReader(content), &output, nil)
			assert.NoError(t, app.ParseAndExecute(ctx, nil))
			assert.Equal(t, content, output.String())
		})
	}
	workers.Wait()
}
