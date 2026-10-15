package cli

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type typedContextRootOptions struct {
	Foo         string
	FactoryOnly string
}

func (options *typedContextRootOptions) String() string { return options.Foo }

type contextAOptions struct{ Name string }
type contextBOptions struct{ Name string }

func TestTypedContextHelpersReadRootCurrentParentAndDecodedFlags(t *testing.T) {
	var observedRoot *typedContextRootOptions
	var observedParent *contextAOptions
	var observedChild *contextBOptions
	root := NewApplication[*typedContextRootOptions]().WithName("app").WithHelp(false).WithErrorHandlerEnabled(false).
		WithCreateOptionsHandler(func(ctx context.Context) (*typedContextRootOptions, error) {
			_, found := ApplicationFromContext[*typedContextRootOptions](ctx)
			assert.False(t, found)
			return &typedContextRootOptions{FactoryOnly: "factory"}, nil
		}).WithStringFlag("foo", func(f *FlagRequirement[string, string]) { f.WithField("Foo").WithAliases("alias") }).
		WithInitializeHandler(func(_ context.Context, options *typedContextRootOptions) error { options.Foo = "updated"; return nil }).
		WithHandler(func(ctx context.Context, options *typedContextRootOptions) error {
			current, found := ApplicationFromContext[*typedContextRootOptions](ctx)
			assert.True(t, found)
			assert.Same(t, options, current)
			_, found = CommandFromContext[*contextBOptions](ctx)
			assert.False(t, found, "root handler has no current command")
			return nil
		})
	a := NewCommand[*contextAOptions]().WithName("a", "first").WithCreateOptionsHandler(func(context.Context) (*contextAOptions, error) { return &contextAOptions{Name: "a"}, nil })
	b := NewCommand[*contextBOptions]().WithName("b", "second").WithCreateOptionsHandler(func(context.Context) (*contextBOptions, error) { return &contextBOptions{Name: "b"}, nil }).
		WithHandler(func(ctx context.Context, options *contextBOptions) error {
			var found bool
			observedRoot, found = ApplicationFromContext[*typedContextRootOptions](ctx)
			assert.True(t, found)
			assert.Equal(t, "updated", observedRoot.Foo)
			assert.Equal(t, "factory", observedRoot.FactoryOnly)
			asInterface, found := ApplicationFromContext[fmt.Stringer](ctx)
			assert.True(t, found)
			assert.Same(t, observedRoot, asInterface)
			observedChild, found = CommandFromContext[*contextBOptions](ctx)
			assert.True(t, found)
			assert.Same(t, options, observedChild)
			observedParent, found = CommandFromContext[*contextAOptions](ctx, "first")
			assert.True(t, found)
			assert.Equal(t, "a", observedParent.Name)
			byPath, found := CommandFromContext[*contextBOptions](ctx, "a", "second")
			assert.True(t, found)
			assert.Same(t, options, byPath)
			_, found = CommandFromContext[*contextAOptions](ctx)
			assert.False(t, found, "wrong type does not select some ancestor")
			_, found = CommandFromContext[*contextBOptions](ctx, "other")
			assert.False(t, found, "unselected command has no options")
			_, found = CommandFromContext[*contextBOptions](ctx, "missing")
			assert.False(t, found)
			foo, found := FlagFromContext[string](ctx, "alias")
			assert.True(t, found)
			assert.Equal(t, "hello", foo, "flag getter reads decoded rather than updated options")
			_, found = FlagFromContext[int](ctx, "foo")
			assert.False(t, found)
			_, found = FlagFromContext[string](ctx, "FactoryOnly")
			assert.False(t, found)
			return nil
		})
	a.WithCommand(b)
	app, err := root.WithCommand(a, NewCommand[*contextBOptions]().
		WithName("other"),
	).Create()
	require.NoError(t, err)
	defer func() { require.NoError(t, app.Close()) }()
	invocation, err := app.Parse(nil, nil, "a", "b", "--foo=hello")
	require.NoError(t, err)
	require.NoError(t, invocation.Execute(nil))
	require.NoError(t, invocation.Close())
	_, actualRoot := invocation.Application()
	assert.Same(t, actualRoot, observedRoot)
	parent, found := FindCommand[*contextAOptions](app, "a")
	require.True(t, found)
	actualParent, found := parent.OptionsFrom(invocation)
	require.True(t, found)
	assert.Same(t, actualParent, observedParent)
}

type nilContextValueCodec struct{}

func (nilContextValueCodec) ToRaw(ctx context.Context) codec.Codec[any, any] {
	return codec.Raw(ctx, nilContextValueCodec{})
}
func (nilContextValueCodec) CanHandleType(ctx context.Context, typ reflect.Type) (bool, error) {
	return typ == reflect.TypeFor[any](), ctx.Err()
}
func (nilContextValueCodec) ExpectMinMax(context.Context, codec.Usage) (int, int) { return 1, 1 }
func (nilContextValueCodec) ParseRequirement(ctx context.Context, _ codec.Usage, args ...string) (string, error) {
	return args[0], ctx.Err()
}
func (nilContextValueCodec) FormatRequirement(ctx context.Context, value string) ([]string, error) {
	return []string{value}, ctx.Err()
}
func (nilContextValueCodec) DisplayRequirement(ctx context.Context, value string) ([]string, error) {
	return []string{value}, ctx.Err()
}
func (nilContextValueCodec) Resolve(ctx context.Context, _ codec.Scope, _ string) (any, error) {
	return nil, ctx.Err()
}
func TestFlagFromContextPreservesPresenceOfZeroAndNil(t *testing.T) {
	app, err := NewSimpleApplication().WithName("app").WithHelp(false).
		WithStringFlag("empty", func(f *FlagRequirement[string, string]) { f.WithDefault("") }).
		WithIntFlag("zero", func(f *FlagRequirement[int, int]) { f.WithDefault(0) }).
		WithBoolFlag("false", func(f *FlagRequirement[bool, bool]) { f.WithDefault(false) }).
		WithFlag(NewFlag(nilContextValueCodec{}).WithName("nil").WithDefault("nil")).
		WithHandler(func(ctx context.Context, _ NoOptions) error {
			empty, found := FlagFromContext[string](ctx, "empty")
			assert.True(t, found)
			assert.Empty(t, empty)
			zero, found := FlagFromContext[int](ctx, "zero")
			assert.True(t, found)
			assert.Zero(t, zero)
			falseValue, found := FlagFromContext[bool](ctx, "false")
			assert.True(t, found)
			assert.False(t, falseValue)
			nilValue, found := FlagFromContext[any](ctx, "nil")
			assert.True(t, found)
			assert.Nil(t, nilValue)
			_, found = FlagFromContext[string](ctx, "nil")
			assert.False(t, found)
			_, found = FlagFromContext[string](ctx, "absent")
			assert.False(t, found)
			return nil
		}).Create()
	require.NoError(t, err)
	defer func() { require.NoError(t, app.Close()) }()
	invocation, err := app.Parse(nil, nil)
	require.NoError(t, err)
	defer func() { require.NoError(t, invocation.Close()) }()
	require.NoError(t, invocation.Execute(nil))
}

func TestTypedContextHelpersReturnFalseWithoutInvocationOrHelpOptions(t *testing.T) {
	for _, ctx := range []context.Context{nil, context.Background()} {
		_, found := ApplicationFromContext[*typedContextRootOptions](ctx)
		assert.False(t, found)
		_, found = CommandFromContext[*contextBOptions](ctx)
		assert.False(t, found)
		_, found = FlagFromContext[string](ctx, "foo")
		assert.False(t, found)
	}
	app, err := NewApplication[*typedContextRootOptions]().
		WithName("app").
		WithCommand(NewCommand[*contextBOptions]().
			WithName("b"),
		).
		WithHelpHandler(func(ctx context.Context, _ InvocationLike) error {
			_, found := ApplicationFromContext[*typedContextRootOptions](ctx)
			assert.False(t, found)
			_, found = CommandFromContext[*contextBOptions](ctx)
			assert.False(t, found)
			_, found = CommandFromContext[*contextBOptions](ctx, "b")
			assert.False(t, found)
			return nil
		}).
		Create()
	require.NoError(t, err)
	defer func() { require.NoError(t, app.Close()) }()
	invocation, err := app.Parse(nil, nil, "b", "--help")
	require.NoError(t, err)
	defer func() { require.NoError(t, invocation.Close()) }()
	require.NoError(t, invocation.Execute(nil))
}
