package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type parseHookOptions struct {
	Input *os.File `flag:"input" required:"true"`
	late  *os.File
}

func TestDirectExecutionDuringParseKeepsOuterCleanupAndResponse(t *testing.T) {
	for _, phase := range []string{"validate", "initialize"} {
		for _, target := range []string{"application", "command"} {
			for _, handled := range []bool{false, true} {
				name := phase + "/" + target + "/propagated"
				if handled {
					name = phase + "/" + target + "/handled"
				}
				t.Run(name, func(t *testing.T) {
					path := filepath.Join(t.TempDir(), "input")
					require.NoError(t, os.WriteFile(path, []byte("value"), 0o600))
					primary := errors.New("nested execution failed")
					var app *Application[*parseHookOptions]
					var command *Command[NoOptions]
					var input, late *os.File
					rootClosed, childClosed, responses, executions := 0, 0, 0, 0
					normalExecution := false
					hook := func(ctx context.Context, options *parseHookOptions) error {
						input = options.Input
						var err error
						if target == "application" {
							err = app.Execute(ctx, options)
						} else {
							err = command.Execute(ctx, NoOptions{})
						}
						require.ErrorIs(t, err, primary)
						assert.Zero(t, responses, "Parse owns the error response")
						assert.Zero(t, rootClosed, "options must remain open inside the hook")
						assert.Zero(t, childClosed, "uninitialized children must remain open")
						_, statErr := input.Stat()
						require.NoError(t, statErr, "codec resources must remain open inside the hook")
						if !handled {
							return err
						}
						late, err = os.Open(path)
						require.NoError(t, err)
						options.late = late
						return nil
					}
					definition := NewApplication[*parseHookOptions]().WithName("app").
						WithHandler(func(context.Context, *parseHookOptions) error {
							if !normalExecution {
								return primary
							}
							executions++
							return nil
						}).
						WithCloseHandler(func(options *parseHookOptions) error {
							rootClosed++
							if options.late != nil {
								return options.late.Close()
							}
							return nil
						}).
						WithCommand(NewSimpleCommand().WithName("child").
							WithHandler(func(context.Context, NoOptions) error {
								if !normalExecution {
									return primary
								}
								executions++
								return nil
							}).
							WithCloseHandler(func(NoOptions) error { childClosed++; return nil })).
						WithErrorHandler(func(_ context.Context, _ InvocationLike, err error) error {
							responses++
							assert.ErrorIs(t, err, primary)
							assert.Equal(t, 1, rootClosed)
							assert.Equal(t, 1, childClosed)
							return nil
						})
					if phase == "validate" {
						definition.WithValidateHandler(hook)
					} else {
						definition.WithInitializeHandler(hook)
					}
					app = definition.MustCreate()
					t.Cleanup(func() { assert.NoError(t, app.Close()) })
					var found bool
					command, found = FindCommand[NoOptions](app, "child")
					require.True(t, found)
					invocation, err := app.Parse(t.Context(), nil, "--input="+path, "child")
					if handled {
						require.NoError(t, err)
						require.NotNil(t, invocation)
						normalExecution = true
						require.NoError(t, invocation.Execute(t.Context()))
						assert.Equal(t, 2, executions)
						require.NoError(t, invocation.Close())
						require.NoError(t, invocation.Close())
						assert.Zero(t, responses)
						_, err = late.Read(make([]byte, 1))
						assert.ErrorIs(t, err, os.ErrClosed)
					} else {
						require.ErrorIs(t, err, primary)
						assert.Nil(t, invocation)
						assert.Equal(t, 1, responses)
					}
					assert.Equal(t, 1, rootClosed)
					assert.Equal(t, 1, childClosed)
					_, err = input.Read(make([]byte, 1))
					assert.ErrorIs(t, err, os.ErrClosed)
				})
			}
		}
	}
}
