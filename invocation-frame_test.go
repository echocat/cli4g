package cli

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/echocat/cli4g/codec"
)

type frameRootOptions struct {
	Root string `flag:"root" short:"r"`
}

func (this *frameRootOptions) Initialize(context.Context) error {
	this.Root = "initialized " + this.Root
	return nil
}

func (this *frameRootOptions) Close() error {
	this.Root = "closed"
	return nil
}

type frameChildOptions struct {
	Child string `flag:"child" short:"c"`
	Name  string `argument:"name"`
}

func (this *frameChildOptions) Initialize(context.Context) error {
	this.Child = "initialized " + this.Child
	this.Name = "initialized " + this.Name
	return nil
}

func (this *frameChildOptions) Close() error {
	this.Child = "closed"
	this.Name = "closed"
	return nil
}

func TestInvocationFramesKeepScopedValuesAndCurrentOptions(t *testing.T) {
	app := NewApplication[frameRootOptions]().
		WithName("app").
		WithCommand(NewSimpleCommand().
			WithName("middle").
			WithFlag(NewFlag(codec.NewString()).
				WithName("middle"),
			).
			WithCommand(NewCommand[*frameChildOptions]().
				WithName("leaf"),
			),
		).
		MustCreate()
	invocation, err := app.Parse(context.Background(), nil,
		"--root=root-value", "middle", "--middle=middle-value", "leaf", "--child=child-value", "name-value")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	middle, found := FindCommand[NoOptions](app, "middle")
	require.True(t, found)
	leaf, found := FindCommand[*frameChildOptions](app, "middle", "leaf")
	require.True(t, found)
	require.Len(t, invocation.frames, 3)
	assert.Same(t, app, invocation.frames[0].container)
	assert.Same(t, middle, invocation.frames[1].container)
	assert.Same(t, leaf, invocation.frames[2].container)
	for i, name := range []string{"root", "middle", "child"} {
		for j, frame := range invocation.frames {
			flag, value := frame.flag(name)
			if i == j {
				require.NotNil(t, flag)
				assert.Equal(t, name+"-value", value)
			} else {
				assert.Nil(t, flag, "flag %s leaked into frame %d", name, j)
			}
		}
	}
	_, rootOptions := invocation.Application()
	assert.Equal(t, "initialized root-value", rootOptions.Root)
	childOptions, found := leaf.OptionsFrom(invocation)
	require.True(t, found)
	assert.Equal(t, "initialized child-value", childOptions.Child)
	assert.Equal(t, "initialized name-value", childOptions.Name)
	_, value := invocation.ShortFlag('r')
	assert.Equal(t, "root-value", value)
	_, value = invocation.ShortFlag('c')
	assert.Equal(t, "child-value", value)
	_, value = invocation.Argument(0)
	assert.Equal(t, "name-value", value)
	var flagNames []string
	for flag := range invocation.Flags() {
		flagNames = append(flagNames, flag.Name())
	}
	assert.Equal(t, []string{"root", "middle", "child"}, flagNames)
	stack := invocation.Stack()
	require.Len(t, stack, 2)
	assert.Same(t, middle, stack[0].Command)
	assert.Equal(t, NoOptions{}, stack[0].Options)
	assert.Same(t, leaf, stack[1].Command)
	assert.Same(t, childOptions, stack[1].Options)
	stack[0].Command = nil
	stack[1].Options = nil
	assert.Same(t, middle, invocation.Stack()[0].Command)
	assert.Same(t, childOptions, invocation.Stack()[1].Options)
	require.NoError(t, invocation.Close())
	_, rootOptions = invocation.Application()
	assert.Equal(t, "closed", rootOptions.Root)
	assert.Equal(t, "closed", invocation.Stack()[1].Options.(*frameChildOptions).Child)
	_, value = invocation.Flag("root")
	assert.Equal(t, "root-value", value)
	_, value = invocation.Argument(0)
	assert.Equal(t, "name-value", value)
}

func TestInvocationRootFrameWithoutCommands(t *testing.T) {
	app, err := NewApplication[*frameChildOptions]().WithName("app").Create()
	require.NoError(t, err)
	invocation, err := app.Parse(context.Background(), nil, "--child=value", "name")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	require.Len(t, invocation.frames, 1)
	assert.Same(t, app, invocation.frames[0].container)
	assert.Empty(t, invocation.Stack())
	assert.Empty(t, invocation.NameStack())
	application, options := invocation.Application()
	assert.Same(t, app, application)
	assert.Equal(t, "initialized value", options.Child)
	assert.Equal(t, "initialized name", options.Name)
	_, value := invocation.Flag("child")
	assert.Equal(t, "value", value)
	_, value = invocation.Argument(0)
	assert.Equal(t, "name", value)
	var arguments []any
	for _, value := range invocation.Arguments() {
		arguments = append(arguments, value)
	}
	assert.Equal(t, []any{"name"}, arguments)
}

func TestDeepFrameLifecycleStateAndRetainedContexts(t *testing.T) {
	const depth = 32
	for _, failAt := range []int{-1, 17} {
		name := "success"
		if failAt >= 0 {
			name = "initializer failure"
		}
		t.Run(name, func(t *testing.T) {
			failure := errors.New("initialization failed")
			var initialized []string
			var closed []string
			var contexts []context.Context
			root := NewSimpleApplication().WithName("app").
				WithInitializeHandler(func(ctx context.Context, _ NoOptions) error {
					initialized = append(initialized, "app")
					contexts = append(contexts, ctx)
					return nil
				}).WithCloseHandler(func(NoOptions) error {
				closed = append(closed, "app")
				return nil
			})
			var previous *CommandRequirement[NoOptions]
			args := make([]string, depth)
			for i := range depth {
				commandName := "command" + strconv.Itoa(i)
				args[i] = commandName
				command := NewSimpleCommand().WithName(commandName).
					WithInitializeHandler(func(ctx context.Context, _ NoOptions) error {
						initialized = append(initialized, commandName)
						contexts = append(contexts, ctx)
						if i == failAt {
							return failure
						}
						return nil
					}).WithCloseHandler(func(NoOptions) error {
					closed = append(closed, commandName)
					return nil
				})
				if previous == nil {
					root.WithCommand(command)
				} else {
					previous.WithCommand(command)
				}
				previous = command
			}
			app, err := root.Create()
			require.NoError(t, err)
			invocation, err := app.Parse(context.Background(), nil, args...)
			count := depth
			if failAt >= 0 {
				require.ErrorIs(t, err, failure)
				assert.Nil(t, invocation)
			} else {
				require.NoError(t, err)
				defer func() { assert.NoError(t, invocation.Close()) }()
				assert.Empty(t, closed)
				require.NoError(t, invocation.Close())
			}
			wantClosed := make([]string, 0, count+1)
			for i := count - 1; i >= 0; i-- {
				wantClosed = append(wantClosed, args[i])
			}
			wantClosed = append(wantClosed, "app")
			assert.Equal(t, wantClosed, closed)
			for i, ctx := range contexts {
				retained, found := InvocationFromContext(ctx)
				require.True(t, found)
				assert.Equal(t, args, retained.NameStack())
				container, found := ContainerFromContext(ctx)
				require.True(t, found)
				assert.Equal(t, initialized[i], container.Name())
				if failAt < 0 {
					assert.Same(t, invocation, retained)
				}
			}
		})
	}
}
