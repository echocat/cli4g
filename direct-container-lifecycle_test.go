package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirectContainerOperationsRespectApplicationClose(t *testing.T) {
	calls, closes := 0, 0
	handler := func(context.Context, NoOptions) error { calls++; return nil }
	cleanup := func(NoOptions) error { closes++; return nil }
	app := NewSimpleApplication().
		WithName("app").
		WithValidateHandler(handler).
		WithInitializeHandler(handler).
		WithHandler(handler).
		WithCloseHandler(cleanup).
		WithCommand(NewSimpleCommand().
			WithName("run").
			WithValidateHandler(handler).
			WithInitializeHandler(handler).
			WithHandler(handler).
			WithCloseHandler(cleanup),
		).
		MustCreate()
	command, found := FindCommand[NoOptions](app, "run")
	require.True(t, found)
	operations := []func(context.Context, NoOptions) error{app.Validate, app.Initialize, app.Execute, command.Validate, command.Initialize, command.Execute}
	for _, operation := range operations {
		require.NoError(t, operation(nil, NoOptions{}))
	}
	assert.Equal(t, 6, calls)
	require.NoError(t, app.Close())
	for _, operation := range operations {
		require.ErrorIs(t, operation(nil, NoOptions{}), ErrApplicationClosed)
	}
	assert.Equal(t, 6, calls, "closed definitions do not dispatch new work")
	require.NoError(t, app.CloseOptions(NoOptions{}))
	require.NoError(t, command.Close(NoOptions{}))
	assert.Equal(t, 2, closes, "cleanup remains callable after definition close")
}
