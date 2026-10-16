package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withTestStdio(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	dir := t.TempDir()
	input, err := os.Create(filepath.Join(dir, "input"))
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, input.Close()) })
	_, err = input.WriteString("payload")
	require.NoError(t, err)
	_, err = input.Seek(0, io.SeekStart)
	require.NoError(t, err)
	output, err := os.Create(filepath.Join(dir, "output"))
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, output.Close()) })
	return input, output
}

func TestFileStdioTagsSupportArgumentsSliceFlagsAndDefaults(t *testing.T) {
	input, output := withTestStdio(t)
	type options struct {
		Input   *os.File   `argument:"input" required:"true" allowStdio:"true"`
		Outputs []*os.File `flag:"output" fileMode:"write" allowStdio:"true"`
		Default *os.File   `flag:"default" default:"stdin" allowStdio:"true"`
	}
	app := NewApplication[options]("files").
		WithErrorHandlerEnabled(false).
		MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	ctx := codec.ContextWithStdio(context.Background(), input, output, nil)
	invocation, err := app.Parse(ctx, nil, "--output", "-", "--output", "stdout", "-")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, opts := invocation.Application()
	assert.Same(t, input, opts.Input)
	assert.Same(t, input, opts.Default)
	require.Len(t, opts.Outputs, 2)
	for _, file := range opts.Outputs {
		assert.Same(t, output, file)
	}
	_, err = io.Copy(opts.Outputs[0], opts.Input)
	require.NoError(t, err)
	require.NoError(t, invocation.Close())
	require.NoError(t, app.Close())
	_, err = input.Stat()
	require.NoError(t, err, "invocation and application cleanup must leave stdin open")
	_, err = output.Seek(0, io.SeekStart)
	require.NoError(t, err, "invocation and application cleanup must leave stdout open")
	content, err := io.ReadAll(output)
	require.NoError(t, err)
	assert.Equal(t, "payload", string(content))
}

func TestFileStdioBuildersPreserveStreamsOnSuccessAndValidationFailure(t *testing.T) {
	input, output := withTestStdio(t)
	ctx := codec.ContextWithStdio(context.Background(), input, output, nil)
	failure := errors.New("validation failed")
	for _, fail := range []bool{false, true} {
		app := NewSimpleApplication("files").
			WithErrorHandlerEnabled(false).
			WithFileFlag("input", func(f *FlagRequirement[*os.File, codec.FileRequirement]) {
				// Typed defaults opt in independently of the codec's policy.
				f.WithDefault(codec.FileRequirement{Path: "stdin", AllowStdio: true})
			}).
			WithFileArgument("output", func(a *ArgumentRequirement[*os.File, codec.FileRequirement]) {
				a.WithCodec(codec.NewFile(codec.FileWrite).WithAllowStdio(true)).WithDefaultStr("stdout")
			}).
			WithValidateHandler(func(context.Context, NoOptions) error {
				if fail {
					return failure
				}
				return nil
			}).
			MustCreate()
		invocation, err := app.Parse(ctx, nil)
		if fail {
			assert.ErrorIs(t, err, failure)
			assert.Nil(t, invocation)
		} else {
			require.NoError(t, err)
			_, value := invocation.Flag("input")
			assert.Same(t, input, value)
			_, value = invocation.Argument(0)
			assert.Same(t, output, value)
			require.NoError(t, invocation.Close())
		}
		require.NoError(t, app.Close())
		_, err = input.Stat()
		require.NoError(t, err)
		_, err = output.Stat()
		require.NoError(t, err)
	}
}

func TestFileStdioApplicationFallbackPolicy(t *testing.T) {
	for _, fallback := range []bool{false, true} {
		app := NewSimpleApplication("files").
			WithErrorHandlerEnabled(false).
			WithFallbackToStdio(fallback).
			WithFileFlag("input", func(f *FlagRequirement[*os.File, codec.FileRequirement]) {
				f.WithCodec(codec.NewFile().
					WithAllowStdio(true),
				).WithDefaultStr("-")
			}).
			MustCreate()
		defer func() { assert.NoError(t, app.Close()) }()
		invocation, err := app.Parse(nil, nil)
		if fallback {
			require.NoError(t, err)
			_, file := invocation.Flag("input")
			assert.Same(t, os.Stdin, file)
			require.NoError(t, invocation.Close())
		} else {
			assert.ErrorIs(t, err, codec.ErrStdioUnavailable)
			assert.Nil(t, invocation)
		}
		// Explicit nil suppresses even an enabled application fallback.
		invocation, err = app.Parse(codec.ContextWithStdio(nil, nil, nil, nil), nil)
		assert.ErrorIs(t, err, codec.ErrStdioUnavailable)
		assert.Nil(t, invocation)
	}
}
