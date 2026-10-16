package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDirectContainerHandlersReceiveCurrentContainer(t *testing.T) {
	var seen []Container
	handler := func(ctx context.Context, _ NoOptions) error {
		container, found := ContainerFromContext(ctx)
		require.True(t, found)
		seen = append(seen, container)
		_, found = InvocationFromContext(ctx)
		assert.False(t, found, "direct calls have no invocation to attach")
		return nil
	}
	app := NewSimpleApplication("app").
		WithValidateHandler(handler).
		WithInitializeHandler(handler).
		WithHandler(handler).
		WithCommand(NewSimpleCommand("child").
			WithValidateHandler(handler).
			WithInitializeHandler(handler).
			WithHandler(handler),
		).
		MustCreate()
	command, found := FindCommand[NoOptions](app, "child")
	require.True(t, found)
	ctx := context.Background()
	require.NoError(t, app.Validate(ctx, NoOptions{}))
	require.NoError(t, app.Initialize(ctx, NoOptions{}))
	require.NoError(t, app.Execute(ctx, NoOptions{}))
	require.NoError(t, command.Validate(ctx, NoOptions{}))
	require.NoError(t, command.Initialize(ctx, NoOptions{}))
	require.NoError(t, command.Execute(ctx, NoOptions{}))
	assert.Equal(t, []Container{app, app, app, command, command, command}, seen)
	_, found = ContainerFromContext(ctx)
	assert.False(t, found)
}
