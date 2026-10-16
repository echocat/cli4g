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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorHandlerExecutionRunsAfterCleanupAndIdentifiesFailingContainer(t *testing.T) {
	primary, cleanup := errors.New("command failed"), errors.New("cleanup failed")
	var events []string
	var reported error
	var retained *Invocation[NoOptions]
	calls := 0
	app, err := NewSimpleApplication("app").
		WithCloseHandler(func(NoOptions) error { events = append(events, "close app"); return cleanup }).
		WithCommand(NewSimpleCommand("group").
			WithHandler(func(context.Context, NoOptions) error { events = append(events, "execute group"); return primary }).
			WithCloseHandler(func(NoOptions) error { events = append(events, "close group"); return nil }).
			WithCommand(NewSimpleCommand("leaf").
				WithHandler(func(context.Context, NoOptions) error {
					assert.Fail(t, "later handlers must not run")
					return nil
				}),
			),
		).
		WithErrorHandler(func(ctx context.Context, invocation InvocationLike, err error) error {
			calls++
			reported = err
			assert.Equal(t, []string{"execute group", "close group", "close app"}, events)
			assert.ErrorIs(t, err, primary)
			assert.ErrorIs(t, err, cleanup)
			current, found := ContainerFromContext(ctx)
			assert.True(t, found)
			assert.Equal(t, "group", current.Name())
			fromContext, found := InvocationFromContext(ctx)
			assert.True(t, found)
			assert.Same(t, invocation, fromContext)
			retained = invocation.(*Invocation[NoOptions])
			assert.True(t, retained.lifecycle.closed.Load())
			return nil
		}).
		Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(nil, nil, "group", "leaf")
	require.NoError(t, err)
	err = invocation.Execute(nil)
	assert.Same(t, reported, err)
	assert.Equal(t, 1, calls)
	assert.Same(t, invocation, retained)
	assert.ErrorIs(t, invocation.Close(), cleanup)
	assert.Equal(t, []string{"execute group", "close group", "close app"}, events)
}

func TestManagedExecutionReportsHandlerAndCompletionCleanupErrorsOnce(t *testing.T) {
	for _, failExecution := range []bool{false, true} {
		t.Run(fmt.Sprint(failExecution), func(t *testing.T) {
			primary, cleanup := errors.New("execute failed"), errors.New("close failed")
			calls, closes := 0, 0
			var reported error
			var reportedApp *Application[NoOptions]
			root := NewSimpleApplication("app").
				WithHandler(func(context.Context, NoOptions) error {
					if failExecution {
						return primary
					}
					return nil
				}).
				WithCloseHandler(func(NoOptions) error { closes++; return cleanup }).
				WithErrorHandler(func(_ context.Context, invocation InvocationLike, err error) error {
					calls++
					reported = err
					assert.Equal(t, 1, closes)
					assert.ErrorIs(t, err, cleanup)
					if failExecution {
						assert.ErrorIs(t, err, primary)
					}
					state := invocation.(*Invocation[NoOptions])
					reportedApp = state.application
					assert.True(t, state.lifecycle.closed.Load())
					assert.False(t, state.application.definition.closed.Load(), "keep metadata/defaults available to the response")
					return nil
				})
			err := root.CreateParseAndExecute(nil, nil)
			assert.Same(t, reported, err)
			assert.Equal(t, 1, calls)
			assert.Equal(t, 1, closes)
			assert.True(t, reportedApp.definition.closed.Load(), "managed application closes when the returning response completes")
			assert.Equal(t, 1, strings.Count(err.Error(), "options could not be closed"), "cleanup error must not be joined repeatedly")
			assert.NotContains(t, err.Error(), cleanup.Error())
		})
	}
}

func TestExecutionErrorHandlingCoversCancellationAndHelp(t *testing.T) {
	for _, stage := range []string{"cancellation", "help"} {
		t.Run(stage, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			failure := errors.New("help failed")
			calls := 0
			app, err := NewSimpleApplication("app").
				WithHelpHandler(func(context.Context, InvocationLike) error { return failure }).
				WithErrorHandler(func(received context.Context, invocation InvocationLike, err error) error {
					calls++
					assert.True(t, invocation.(*Invocation[NoOptions]).lifecycle.closed.Load())
					if stage == "cancellation" {
						assert.ErrorIs(t, received.Err(), context.Canceled)
						assert.ErrorIs(t, err, context.Canceled)
					} else {
						assert.ErrorIs(t, err, failure)
					}
					return nil
				}).Create()
			require.NoError(t, err)
			defer func() { assert.NoError(t, app.Close()) }()
			var args []string
			if stage == "help" {
				args = []string{"--help"}
			}
			invocation, err := app.Parse(nil, nil, args...)
			require.NoError(t, err)
			if stage == "cancellation" {
				cancel()
			}
			require.Error(t, invocation.Execute(ctx))
			assert.Equal(t, 1, calls)
			require.NoError(t, invocation.Close())
		})
	}
}

func TestExecutionHandlersAreCapturedOverriddenAndDisabledLocally(t *testing.T) {
	original := DefaultErrorHandler
	t.Cleanup(func() { DefaultErrorHandler = original })
	primary, replacement := errors.New("execute failed"), errors.New("replacement")
	var events []string
	build := func(name string) *ApplicationRequirement[NoOptions] {
		return NewSimpleApplication(name).
			WithHandler(func(context.Context, NoOptions) error { return primary })
	}
	DefaultErrorHandler = func(context.Context, InvocationLike, error) error { events = append(events, "first"); return nil }
	first, err := build("first").Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, first.Close()) }()
	DefaultErrorHandler = func(context.Context, InvocationLike, error) error {
		events = append(events, "second")
		return replacement
	}
	second, err := build("second").WithErrorHandler(nil).Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, second.Close()) }()
	explicitRequirement := build("explicit").WithErrorHandler(func(context.Context, InvocationLike, error) error {
		events = append(events, "explicit")
		return nil
	})
	explicit, err := explicitRequirement.Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, explicit.Close()) }()
	explicitRequirement.SetErrorHandler(nil)
	closed := false
	disabledRequirement := build("disabled").WithErrorHandlerEnabled(false).
		WithCloseHandler(func(NoOptions) error { closed = true; return nil })
	disabled, err := disabledRequirement.Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, disabled.Close()) }()
	disabledRequirement.SetErrorHandlerEnabled(true)
	DefaultErrorHandler = nil
	for _, app := range []*Application[NoOptions]{first, second, explicit, disabled} {
		invocation, err := app.Parse(nil, nil)
		require.NoError(t, err)
		got := invocation.Execute(nil)
		if app == second {
			assert.Same(t, replacement, got)
		} else {
			assert.Same(t, primary, got)
		}
		if app == disabled {
			assert.False(t, closed, "disabled handling preserves caller cleanup responsibility")
		}
		require.NoError(t, invocation.Close())
	}
	assert.Equal(t, []string{"first", "second", "explicit"}, events)
	assert.True(t, closed)
}

func TestDirectExecutionErrorsAreReportedButNestedCallsAreNotDuplicated(t *testing.T) {
	primary := errors.New("direct execution failed")
	var events []string
	app, err := NewSimpleApplication("app").
		WithCommand(NewSimpleCommand("child").
			WithHandler(func(context.Context, NoOptions) error { return primary }),
		).
		WithErrorHandler(func(ctx context.Context, _ InvocationLike, err error) error {
			current, found := ContainerFromContext(ctx)
			assert.True(t, found)
			events = append(events, current.Name())
			return nil
		}).
		Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, app.Close()) }()
	command, found := FindCommand[NoOptions](app, "child")
	require.True(t, found)
	assert.Same(t, primary, command.Execute(nil, NoOptions{}))
	assert.Equal(t, []string{"child"}, events)

	nestedCalls := 0
	var nestedCommand *Command[NoOptions]
	nested, err := NewSimpleApplication("nested").
		WithCommand(NewSimpleCommand("child").
			WithHandler(func(context.Context, NoOptions) error { return primary }),
		).
		WithHandler(func(ctx context.Context, _ NoOptions) error { return nestedCommand.Execute(ctx, NoOptions{}) }).
		WithErrorHandler(func(context.Context, InvocationLike, error) error { nestedCalls++; return nil }).
		Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, nested.Close()) }()
	nestedCommand, found = FindCommand[NoOptions](nested, "child")
	require.True(t, found)
	assert.Same(t, primary, nested.Execute(nil, NoOptions{}))
	assert.Equal(t, 1, nestedCalls)
	nestedCalls = 0
	invocation, err := nested.Parse(nil, nil, "child")
	require.NoError(t, err)
	assert.Same(t, primary, invocation.Execute(nil))
	assert.Equal(t, 1, nestedCalls)
	require.NoError(t, invocation.Close())
}

func TestRespondExecutionErrorCapturesOutputAndPreservesErrors(t *testing.T) {
	var output bytes.Buffer
	requirement := NewRespondErrorHandler().WithOutput(&output)
	handler, err := requirement.Create()
	require.NoError(t, err)
	requirement.SetOutput(io.Discard)
	requirement.SetExecutionExitCode(7)
	primary := errors.New("execution failed")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	assert.Same(t, primary, handler(ctx, nil, primary))
	assert.Equal(t, "Error: execution failed\n", output.String())
	assert.NoError(t, handler(ctx, nil, nil))
	for _, writer := range []helpTestWriter{{err: io.ErrClosedPipe}, {short: true}} {
		handler, err := NewRespondErrorHandler().WithOutput(writer).Create()
		require.NoError(t, err)
		err = handler(nil, nil, primary)
		assert.ErrorIs(t, err, primary)
		if writer.short {
			assert.ErrorIs(t, err, io.ErrShortWrite)
		} else {
			assert.ErrorIs(t, err, io.ErrClosedPipe)
		}
	}
	var invalid *RespondErrorHandlerRequirement
	_, err = invalid.Create()
	require.Error(t, err)
}

func TestExecutionErrorOutputIsSafeForConcurrentInvocations(t *testing.T) {
	var output bytes.Buffer
	handler, err := NewRespondErrorHandler().WithOutput(&output).Create()
	require.NoError(t, err)
	primary := errors.New("failed")
	app, err := NewSimpleApplication("app").
		WithErrorHandler(handler).
		WithHandler(func(context.Context, NoOptions) error { return primary }).
		Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, app.Close()) }()
	var workers sync.WaitGroup
	for range 32 {
		workers.Go(func() {
			invocation, err := app.Parse(nil, nil)
			if !assert.NoError(t, err) {
				return
			}
			assert.ErrorIs(t, invocation.Execute(nil), primary)
			assert.NoError(t, invocation.Close())
		})
	}
	workers.Wait()
	assert.Equal(t, strings.Repeat("Error: failed\n", 32), output.String())
}

func TestExecutionErrorExitProcess(t *testing.T) {
	mode := os.Getenv("CLI4G_EXECUTION_ERROR_EXIT_TEST")
	if mode != "" {
		DefaultErrorHandler = subprocessDefaultErrorHandler
		if mode == "invalid-config" {
			NewSimpleApplication().MustCreateParseAndExecute(nil, nil)
			return
		}
		primary := errors.New("execution failed")
		if strings.HasPrefix(mode, "cancel-") {
			primary = fmt.Errorf("wrapped cancellation: %w", context.Canceled)
		}
		if mode == "input-cancel-cleanup" {
			primary = NewValidationErrorf("bad input")
		}
		root := NewSimpleApplication("app").
			WithHandler(func(context.Context, NoOptions) error {
				if mode == "cleanup" {
					return nil
				}
				return primary
			}).WithCloseHandler(func(NoOptions) error {
			_, err := io.WriteString(os.Stderr, "CLEANUP\n")
			if mode == "input-cancel-cleanup" {
				return errors.Join(err, context.Canceled)
			}
			if mode == "cleanup" || mode == "cancel-joined" {
				return errors.Join(err, errors.New("cleanup failed"))
			}
			return err
		})
		if mode == "snapshot" || mode == "zero" || mode == "writer-error" {
			response := NewRespondErrorHandler().WithExecutionExitCode(7)
			if mode == "zero" {
				response.SetExecutionExitCode(0)
			}
			if mode == "writer-error" {
				response.SetOutput(helpTestWriter{err: io.ErrClosedPipe})
			}
			handler, err := response.Create()
			require.NoError(t, err)
			response.SetExecutionExitCode(9)
			root.WithErrorHandler(handler)
		}
		if mode == "cancel-snapshot" || mode == "cancel-zero" || mode == "cancel-writer-error" {
			response := NewRespondErrorHandler().WithExecutionExitCode(9).WithCanceledExitCode(7)
			if mode == "cancel-zero" {
				response.SetCanceledExitCode(0)
			}
			if mode == "cancel-writer-error" {
				response.SetOutput(helpTestWriter{err: io.ErrClosedPipe})
			}
			handler := response.MustCreate()
			response.SetCanceledExitCode(11)
			root.WithErrorHandler(handler)
		}
		if mode == "cancel" {
			app := root.MustCreate()
			invocation, err := app.Parse(nil, nil)
			require.NoError(t, err)
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_ = invocation.Execute(ctx)
			return
		}
		if mode == "help" {
			root.WithHelpHandler(func(context.Context, InvocationLike) error { return errors.New("help failed") }).
				MustCreateParseAndExecute(nil, nil, "--help")
			return
		}
		if mode == "zero" || mode == "cancel-zero" {
			require.ErrorIs(t, root.CreateParseAndExecute(nil, nil), primary)
			return
		}
		root.MustCreateParseAndExecute(nil, nil)
		return
	}
	for _, mode := range []string{"default", "cleanup", "help", "cancel", "snapshot", "zero", "writer-error", "invalid-config",
		"cancel-snapshot", "cancel-zero", "cancel-writer-error", "cancel-joined", "input-cancel-cleanup"} {
		t.Run(mode, func(t *testing.T) {
			command := exec.Command(os.Args[0], "-test.run=^TestExecutionErrorExitProcess$")
			command.Env = append(os.Environ(), "CLI4G_EXECUTION_ERROR_EXIT_TEST="+mode)
			var stdout, stderr bytes.Buffer
			command.Stdout, command.Stderr = &stdout, &stderr
			err := command.Run()
			want := 1
			if mode == "snapshot" || mode == "writer-error" || mode == "cancel-snapshot" || mode == "cancel-writer-error" {
				want = 7
			} else if mode == "cancel" || mode == "cancel-joined" {
				want = 130
			} else if mode == "invalid-config" || mode == "input-cancel-cleanup" {
				want = 2
			} else if mode == "zero" || mode == "cancel-zero" {
				want = 0
			}
			if want == 0 {
				require.NoError(t, err, stderr.String())
			} else {
				var exit *exec.ExitError
				require.ErrorAs(t, err, &exit, stderr.String())
				assert.Equal(t, want, exit.ExitCode())
			}
			assert.NotContains(t, stdout.String(), "Error: ")
			assert.NotContains(t, stdout.String(), "CLEANUP")
			if mode == "invalid-config" {
				assert.Contains(t, stderr.String(), "panic: application needs a name")
				return
			}
			assert.NotContains(t, stderr.String(), "panic:")
			if mode != "help" {
				assert.Equal(t, 1, strings.Count(stderr.String(), "CLEANUP\n"))
			}
			if mode != "writer-error" && mode != "cancel-writer-error" {
				assert.Equal(t, 1, strings.Count(stderr.String(), "Error: "))
				if mode != "help" {
					assert.Less(t, strings.Index(stderr.String(), "CLEANUP"), strings.Index(stderr.String(), "Error:"))
				}
			}
			if mode == "cancel" || strings.HasPrefix(mode, "cancel-") {
				assert.NotContains(t, stderr.String(), "Usage:")
			}
			if mode == "input-cancel-cleanup" {
				assert.Contains(t, stderr.String(), "Usage:")
			}
		})
	}
}
