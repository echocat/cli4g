package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type parseErrorOwnedOptions struct {
	Number   int `flag:"number"`
	closed   *atomic.Int64
	closeErr error
}

func (options *parseErrorOwnedOptions) Close() error {
	options.closed.Add(1)
	return options.closeErr
}

type parseErrorChildOptions struct{ closed *atomic.Int64 }

func (options *parseErrorChildOptions) Close() error { options.closed.Add(1); return nil }

func TestErrorHandlerParseRunsAfterCleanupAndReplacesOrPreservesError(t *testing.T) {
	for _, replace := range []bool{false, true} {
		name := "preserve on nil"
		if replace {
			name = "replace"
		}
		t.Run(name, func(t *testing.T) {
			var closes atomic.Int64
			cleanupError := errors.New("cleanup failed")
			replacement := errors.New("replacement")
			var received error
			var retained *Invocation[*parseErrorOwnedOptions]
			calls := 0
			app, err := NewApplication[*parseErrorOwnedOptions]("app").
				WithCreateOptionsHandler(func(context.Context) (*parseErrorOwnedOptions, error) {
					return &parseErrorOwnedOptions{closed: &closes, closeErr: cleanupError}, nil
				}).WithErrorHandler(func(ctx context.Context, invocation InvocationLike, err error) error {
				calls++
				received = err
				assert.ErrorIs(t, err, ErrValidation)
				assert.EqualValues(t, 1, closes.Load())
				assert.ErrorIs(t, err, cleanupError)
				assert.ErrorContains(t, err, "flag --number")
				fromContext, found := InvocationFromContext(ctx)
				assert.True(t, found)
				assert.Same(t, invocation, fromContext)
				container, found := ContainerFromContext(ctx)
				assert.True(t, found)
				assert.Equal(t, "app", container.Name())
				assert.False(t, invocation.HelpRequested())
				retained = invocation.(*Invocation[*parseErrorOwnedOptions])
				assert.ErrorIs(t, retained.Close(), cleanupError)
				if replace {
					return replacement
				}
				return nil
			}).Create()
			require.NoError(t, err)
			invocation, err := app.Parse(nil, nil, "--number=invalid")
			assert.Nil(t, invocation)
			require.Error(t, err)
			assert.Equal(t, 1, calls)
			assert.EqualValues(t, 1, closes.Load())
			if replace {
				assert.Same(t, replacement, err)
			} else {
				assert.Same(t, received, err)
			}
			require.NotNil(t, retained)
			assert.ErrorIs(t, retained.Execute(nil), ErrInvocationNotReady)
			assert.Equal(t, "app", retained.HelpInfo().Target.Name())
		})
	}
}

func TestValidationErrorMetadataRetainsFullSelectedPathOnScannerAndEarlyFactoryFailures(t *testing.T) {
	for _, scannerError := range []bool{false, true} {
		var closes atomic.Int64
		factories := 0
		calls := 0
		failure := errors.New("factory failed")
		app := NewApplication[*parseErrorOwnedOptions]("app").
			WithCreateOptionsHandler(func(context.Context) (*parseErrorOwnedOptions, error) {
				factories++
				return &parseErrorOwnedOptions{closed: &closes}, nil
			}).
			WithCommand(NewCommand[*parseErrorChildOptions]("middle").
				WithCreateOptionsHandler(func(context.Context) (*parseErrorChildOptions, error) {
					factories++
					return &parseErrorChildOptions{closed: &closes}, failure
				}).
				WithCommand(NewSimpleCommand("leaf")),
			).
			WithErrorHandler(func(_ context.Context, invocation InvocationLike, err error) error {
				calls++
				assert.Equal(t, []string{"middle", "leaf"}, invocation.NameStack())
				assert.Equal(t, "leaf", invocation.HelpInfo().Target.Name())
				assert.Len(t, invocation.HelpInfo().Path, 3)
				stack := invocation.Stack()
				if assert.Len(t, stack, 2) {
					assert.Nil(t, stack[1].Options)
				}
				if scannerError {
					assert.Zero(t, factories)
					assert.Zero(t, closes.Load())
				} else {
					assert.Equal(t, 2, factories)
					assert.EqualValues(t, 2, closes.Load())
				}
				return err
			}).
			MustCreate()
		args := []string{"middle", "leaf"}
		if scannerError {
			args = append(args, "--unknown")
		}
		invocation, err := app.Parse(context.Background(), nil, args...)
		assert.Nil(t, invocation)
		assert.Equal(t, 1, calls)
		if scannerError {
			require.ErrorContains(t, err, "unknown flag --unknown")
		} else {
			require.ErrorIs(t, err, failure)
		}
	}
}

func TestErrorHandlerParseCancellationAndSuccessBoundaries(t *testing.T) {
	var calls int
	app, err := NewSimpleApplication("app").WithErrorHandler(func(ctx context.Context, invocation InvocationLike, err error) error {
		calls++
		assert.ErrorIs(t, ctx.Err(), context.Canceled)
		assert.ErrorIs(t, err, context.Canceled)
		assert.Equal(t, "app", invocation.HelpInfo().Target.Name())
		return nil
	}).Create()
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	invocation, err := app.Parse(ctx, nil)
	assert.Nil(t, invocation)
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, calls)
	for _, args := range [][]string{nil, {"--help"}} {
		invocation, err := app.Parse(nil, nil, args...)
		require.NoError(t, err)
		require.NoError(t, invocation.Close())
	}
	assert.Equal(t, 1, calls)
}

func TestDefaultErrorHandlerParseOverrideAndApplicationHandlerAreCaptured(t *testing.T) {
	original := DefaultErrorHandler
	defer func() { DefaultErrorHandler = original }()
	var calls []string
	DefaultErrorHandler = func(context.Context, InvocationLike, error) error {
		calls = append(calls, "first")
		return nil
	}
	first, err := NewSimpleApplication("first").Create()
	require.NoError(t, err)
	DefaultErrorHandler = func(context.Context, InvocationLike, error) error {
		calls = append(calls, "second")
		return nil
	}
	second, err := NewSimpleApplication("second").Create()
	require.NoError(t, err)
	requirement := NewSimpleApplication("explicit")
	requirement.SetErrorHandler(func(context.Context, InvocationLike, error) error {
		calls = append(calls, "explicit")
		return nil
	})
	explicit, err := requirement.Create()
	require.NoError(t, err)
	requirement.SetErrorHandler(nil)
	DefaultErrorHandler = nil
	for _, app := range []*Application[NoOptions]{first, second, explicit} {
		invocation, err := app.Parse(nil, nil, "--unknown")
		assert.Nil(t, invocation)
		require.ErrorContains(t, err, "unknown flag --unknown")
	}
	assert.Equal(t, []string{"first", "second", "explicit"}, calls)
	disabled, err := NewSimpleApplication("disabled").Create()
	require.NoError(t, err)
	_, err = disabled.Parse(nil, nil, "--unknown")
	require.ErrorContains(t, err, "unknown flag --unknown")
	assert.Len(t, calls, 3)
}

func TestRespondWithHelpUsesApplicationOrExplicitHandlerAndContextFallback(t *testing.T) {
	var output bytes.Buffer
	responseRequirement := NewRespondErrorHandler().WithOutput(&output)
	handler, err := responseRequirement.Create()
	require.NoError(t, err)
	responseRequirement.SetOutput(io.Discard)
	app, err := NewSimpleApplication("app").WithHelpHandler(func(_ context.Context, invocation InvocationLike) error {
		_, err := fmt.Fprintf(&output, "HELP %s\n", invocation.HelpInfo().Target.Name())
		return err
	}).WithErrorHandler(handler).Create()
	require.NoError(t, err)
	invocation, err := app.Parse(nil, nil, "--unknown")
	assert.Nil(t, invocation)
	require.ErrorContains(t, err, "unknown flag --unknown")
	assert.Equal(t, "Error: unknown flag --unknown\n\nHELP app\n", output.String())
	output.Reset()
	helpInvocation, err := app.Parse(nil, nil, "--help")
	require.NoError(t, err)
	defer func() { assert.NoError(t, helpInvocation.Close()) }()
	ctx := withInvocationContext(context.Background(), helpInvocation)
	validationError := NewValidationError("bad input")
	var typedNil *Invocation[NoOptions]
	assert.Same(t, validationError, handler(ctx, typedNil, validationError))
	assert.Equal(t, "Error: bad input\n\nHELP app\n", output.String())
	output.Reset()
	explicit, err := NewRespondErrorHandler().WithOutput(&output).
		WithHelpHandler(func(context.Context, InvocationLike) error {
			_, err := io.WriteString(&output, "EXPLICIT\n")
			return err
		}).Create()
	require.NoError(t, err)
	assert.Same(t, validationError, explicit(ctx, helpInvocation, validationError))
	assert.Equal(t, "Error: bad input\n\nEXPLICIT\n", output.String())
}

func TestRespondWithHelpReturnsOriginalAndResponseFailuresWithoutExiting(t *testing.T) {
	validationError := NewValidationError("bad input")
	responseError := errors.New("response failed")
	var calls int
	for _, writer := range []helpTestWriter{{err: responseError}, {short: true}} {
		handler, err := NewRespondErrorHandler().WithOutput(writer).
			WithHelpHandler(func(context.Context, InvocationLike) error { calls++; return nil }).Create()
		require.NoError(t, err)
		err = handler(nil, nil, validationError)
		require.ErrorIs(t, err, validationError)
		if writer.short {
			require.ErrorIs(t, err, io.ErrShortWrite)
		} else {
			require.ErrorIs(t, err, responseError)
		}
	}
	assert.Zero(t, calls)
	handler, err := NewRespondErrorHandler().WithOutput(io.Discard).
		WithHelpHandler(func(context.Context, InvocationLike) error { return responseError }).Create()
	require.NoError(t, err)
	err = handler(nil, nil, validationError)
	require.ErrorIs(t, err, validationError)
	require.ErrorIs(t, err, responseError)
	assert.NoError(t, handler(nil, nil, nil))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = handler(ctx, nil, validationError)
	require.ErrorIs(t, err, validationError)
	require.ErrorIs(t, err, responseError)
	assert.NotErrorIs(t, err, context.Canceled, "reporting retains values but ignores context cancellation")
	app, err := NewSimpleApplication("app").Create()
	require.NoError(t, err)
	invocation, err := app.Parse(nil, nil, "--help")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	ctx, cancel = context.WithCancel(context.Background())
	handler, err = NewRespondErrorHandler().WithOutput(io.Discard).
		WithHelpHandler(func(context.Context, InvocationLike) error { cancel(); return nil }).Create()
	require.NoError(t, err)
	err = handler(ctx, invocation, validationError)
	require.ErrorIs(t, err, validationError)
	assert.NotErrorIs(t, err, context.Canceled, "cancellation during reporting does not replace the primary input error")
}

func TestRespondWithHelpConcurrentResponsesRemainOrdered(t *testing.T) {
	var output bytes.Buffer
	handler, err := NewRespondErrorHandler().WithOutput(&output).
		WithHelpHandler(func(context.Context, InvocationLike) error {
			_, err := io.WriteString(&output, "HELP\n")
			return err
		}).Create()
	require.NoError(t, err)
	app, err := NewSimpleApplication("app").WithErrorHandler(handler).Create()
	require.NoError(t, err)
	var workers sync.WaitGroup
	for range 32 {
		workers.Go(func() {
			invocation, err := app.Parse(nil, nil, "--unknown")
			assert.Nil(t, invocation)
			assert.ErrorContains(t, err, "unknown flag --unknown")
		})
	}
	workers.Wait()
	assert.Equal(t, strings.Repeat("Error: unknown flag --unknown\n\nHELP\n", 32), output.String())
}

func TestRespondInputErrorExitProcess(t *testing.T) {
	mode := os.Getenv("BIFROEST_CLI_PARSE_ERROR_EXIT_TEST")
	if mode != "" {
		response := NewRespondErrorHandler().WithValidationExitCode(7)
		switch mode {
		case "zero", "default":
			response.SetValidationExitCode(0)
		case "writer error":
			response.SetOutput(helpTestWriter{err: errors.New("writer failed")})
		}
		help := HelpHandler(func(context.Context, InvocationLike) error {
			if mode == "help error" || mode == "default-help-error" {
				return errors.New("help failed")
			}
			_, err := io.WriteString(os.Stderr, "HELP-OUTPUT\n")
			return err
		})
		handler, err := response.Create()
		require.NoError(t, err)
		if mode == "snapshot" {
			response.SetValidationExitCode(9)
		}
		if strings.HasPrefix(mode, "default") {
			DefaultErrorHandler = subprocessDefaultErrorHandler
			handler = nil // use the actual package DefaultErrorHandler
		}
		var closes atomic.Int64
		app, err := NewApplication[*parseErrorOwnedOptions]("app").
			WithCreateOptionsHandler(func(context.Context) (*parseErrorOwnedOptions, error) {
				return &parseErrorOwnedOptions{closed: &closes}, nil
			}).WithHelpHandler(func(ctx context.Context, invocation InvocationLike) error {
			wantCloses := int64(1)
			if mode == "default-cancel" {
				wantCloses = 0 // canceled before any options factory ran
			}
			require.EqualValues(t, wantCloses, closes.Load(), "cleanup must precede os.Exit")
			return help(ctx, invocation)
		}).WithErrorHandler(handler).Create()
		require.NoError(t, err)
		ctx := context.Background()
		if mode == "cancel" || mode == "default-cancel" {
			var cancel context.CancelFunc
			ctx, cancel = context.WithCancel(ctx)
			cancel()
		}
		invocation, err := app.Parse(ctx, nil, "--number=invalid")
		assert.Nil(t, invocation)
		require.Error(t, err)
		return
	}
	for _, mode := range []string{"exit", "snapshot", "zero", "writer error", "help error", "cancel", "default", "default-cancel", "default-help-error"} {
		t.Run(mode, func(t *testing.T) {
			command := exec.Command(os.Args[0], "-test.run=^TestRespondInputErrorExitProcess$")
			command.Env = append(os.Environ(), "BIFROEST_CLI_PARSE_ERROR_EXIT_TEST="+mode)
			var stdout, stderr bytes.Buffer
			command.Stdout, command.Stderr = &stdout, &stderr
			err := command.Run()
			if mode == "exit" || mode == "snapshot" || mode == "writer error" || mode == "help error" || strings.HasPrefix(mode, "default") {
				var exit *exec.ExitError
				require.ErrorAs(t, err, &exit)
				want := 7
				if strings.HasPrefix(mode, "default") {
					want = 2
					if mode == "default-cancel" {
						want = 130
					}
				}
				assert.Equal(t, want, exit.ExitCode())
			} else {
				require.NoError(t, err, "stdout=%s stderr=%s", stdout.String(), stderr.String())
			}
			if mode == "exit" || mode == "snapshot" || mode == "zero" || strings.HasPrefix(mode, "default") {
				if mode == "default-cancel" {
					assert.Contains(t, stderr.String(), "Error: context canceled")
				} else {
					assert.Contains(t, stderr.String(), "Error: flag --number")
				}
				if mode != "default-help-error" && mode != "default-cancel" {
					assert.Contains(t, stderr.String(), "HELP-OUTPUT\n")
				}
			}
			assert.NotContains(t, stderr.String(), "panic:")
			assert.NotContains(t, stdout.String(), "HELP-OUTPUT")
			assert.NotContains(t, stdout.String(), "Error: flag --number")
		})
	}
}
