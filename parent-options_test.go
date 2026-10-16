package cli

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sharedParentOptions struct{ Name string }

type benchmarkUnlinkedParentOptions struct{ Parent *sharedParentOptions }
type benchmarkRootParentOptions struct {
	Parent *sharedParentOptions `parent:"application"`
}
type benchmarkMatchingParentOptions struct {
	Parent *sharedParentOptions `parent:"matchingType"`
}

func BenchmarkApplicationParentInjection(b *testing.B) {
	for _, mode := range []string{"Unlinked", "Application", "MatchingType"} {
		b.Run(mode, func(b *testing.B) {
			var leaf CommandRequirementLike
			switch mode {
			case "Unlinked":
				leaf = NewCommand[*benchmarkUnlinkedParentOptions]("leaf")
			case "Application":
				leaf = NewCommand[*benchmarkRootParentOptions]("leaf")
			case "MatchingType":
				leaf = NewCommand[*benchmarkMatchingParentOptions]("leaf")
			}
			app := NewApplication[*sharedParentOptions]("app").
				WithHelp(false).
				WithStringFlag("name", func(f *FlagRequirement[string, string]) { f.WithField("Name") }).
				WithCommand(NewSimpleCommand("group").
					WithCommand(leaf),
				).
				MustCreate()
			defer func() { require.NoError(b, app.Close()) }()
			benchmarkParse(b, app, context.Background(), nil, "group", "leaf", "--name=value")
		})
	}
}

type nearestChildOptions struct {
	Nearest *sharedParentOptions `parent:"matchingType"`
	Root    *sharedParentOptions `parent:"application"`
}

func TestParentInjectionUsesNearestMatchAndCanonicalStructRoot(t *testing.T) {
	initialized := false
	app := NewApplication[sharedParentOptions]("app").
		WithHelp(false).
		WithErrorHandlerEnabled(false).
		WithStringFlag("root-name", func(f *FlagRequirement[string, string]) {
			f.
				WithField("Name").
				WithDefault("root")
		}).
		WithCommand(NewCommand[*sharedParentOptions]("a").
			WithStringFlag("parent-name", func(f *FlagRequirement[string, string]) {
				f.
					WithField("Name").
					WithDefault("parent")
			}).
			WithCommand(NewCommand[*nearestChildOptions]("b").
				WithInitializeHandler(func(_ context.Context, options *nearestChildOptions) error {
					if options.Nearest.Name != "parent" || options.Root.Name != "root" {
						return errors.New("parents not decoded before initialization")
					}
					initialized = true
					return nil
				}),
			),
		).
		MustCreate()
	defer func() { require.NoError(t, app.Close()) }()
	invocation := app.MustParse(nil, nil, "a", "b")
	defer func() { require.NoError(t, invocation.Close()) }()
	child, found := FindCommand[*nearestChildOptions](app, "a", "b")
	require.True(t, found)
	options, found := child.OptionsFrom(invocation)
	require.True(t, found)
	parent, found := FindCommand[*sharedParentOptions](app, "a")
	require.True(t, found)
	parentOptions, found := parent.OptionsFrom(invocation)
	require.True(t, found)
	assert.Same(t, parentOptions, options.Nearest)
	assert.NotSame(t, options.Root, options.Nearest)
	assert.True(t, initialized)
	options.Root.Name = "changed through canonical root pointer"
	_, root := invocation.Application()
	assert.Equal(t, options.Root.Name, root.Name)
}

type InjectableAppOptions struct {
	Foo string `flag:"foo" default:"default"`
}
type anonymousParentChild struct {
	*InjectableAppOptions `parent:"application"`
	Own                   string `flag:"own" default:"child"`
}

func TestAnonymousParentReferenceIsOpaqueAndInvocationLocal(t *testing.T) {
	app := NewApplication[*InjectableAppOptions]("app").
		WithHelp(false).
		WithErrorHandlerEnabled(false).
		WithCommand(NewCommand[*anonymousParentChild]("run")).
		MustCreate()
	defer func() { require.NoError(t, app.Close()) }()
	command, found := FindCommand[*anonymousParentChild](app, "run")
	require.True(t, found)
	_, found = command.Flag("foo")
	assert.False(t, found, "parent fields do not define child flags")
	results := make(chan error, 32)
	var workers sync.WaitGroup
	for range 32 {
		workers.Go(func() {
			invocation, err := app.Parse(nil, nil, "run", "--foo=hello")
			if err != nil {
				results <- err
				return
			}
			_, root := invocation.Application()
			child, found := command.OptionsFrom(invocation)
			if !found || child.InjectableAppOptions != root || child.Foo != "hello" || child.Own != "child" {
				err = errors.New("incorrect borrowed parent")
			}
			child.Foo = "private mutation"
			results <- errors.Join(err, invocation.Close())
		})
	}
	workers.Wait()
	close(results)
	for err := range results {
		require.NoError(t, err)
	}
	invocation, err := app.Parse(nil, nil, "run")
	require.NoError(t, err)
	defer func() { require.NoError(t, invocation.Close()) }()
	_, root := invocation.Application()
	assert.Equal(t, "default", root.Foo)
}

func TestMatchingTypeChildRequirementCanBeReusedUnderDifferentNames(t *testing.T) {
	type childOptions struct {
		Parent *sharedParentOptions `parent:"matchingType"`
	}
	child := NewCommand[*childOptions]("child")
	branch := func(name string) *CommandRequirement[*sharedParentOptions] {
		return NewCommand[*sharedParentOptions](name).
			WithCreateOptionsHandler(func(context.Context) (*sharedParentOptions, error) { return &sharedParentOptions{Name: name}, nil }).
			WithCommand(child)
	}
	app := NewSimpleApplication("app").
		WithHelp(false).
		WithErrorHandlerEnabled(false).
		WithCommand(branch("a"), branch("z")).
		MustCreate()
	defer func() { require.NoError(t, app.Close()) }()
	for _, name := range []string{"a", "z"} {
		invocation, err := app.Parse(nil, nil, name, "child")
		require.NoError(t, err)
		command, found := FindCommand[*childOptions](app, name, "child")
		require.True(t, found)
		options, found := command.OptionsFrom(invocation)
		require.True(t, found)
		assert.Equal(t, name, options.Parent.Name)
		require.NoError(t, invocation.Close())
	}
}

type LifecycleParentOptions struct{ closes *int }

func (options *LifecycleParentOptions) Close() error { *options.closes++; return nil }

type lifecycleChildOptions struct {
	Parent *LifecycleParentOptions `parent:"matchingType"`
	closes *int
}

func (options *lifecycleChildOptions) Close() error { *options.closes++; return nil }
func TestNamedParentDoesNotTransferLifecycleOwnership(t *testing.T) {
	for _, rollback := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "factory-error"}[rollback], func(t *testing.T) {
			parentCloses, childCloses := 0, 0
			primary := errors.New("factory failed")
			app := NewApplication[*LifecycleParentOptions]("app").
				WithHelp(false).
				WithErrorHandlerEnabled(false).
				WithCreateOptionsHandler(func(context.Context) (*LifecycleParentOptions, error) {
					return &LifecycleParentOptions{closes: &parentCloses}, nil
				}).
				WithCommand(NewCommand[*lifecycleChildOptions]("run").
					WithCreateOptionsHandler(func(context.Context) (*lifecycleChildOptions, error) {
						options := &lifecycleChildOptions{closes: &childCloses}
						if rollback {
							return options, primary
						}
						return options, nil
					}),
				).
				MustCreate()
			defer func() { require.NoError(t, app.Close()) }()
			invocation, err := app.Parse(nil, nil, "run")
			if rollback {
				require.ErrorIs(t, err, primary)
				assert.Nil(t, invocation)
			} else {
				require.NoError(t, err)
				require.NoError(t, invocation.Close())
				require.NoError(t, invocation.Close())
			}
			assert.Equal(t, 1, parentCloses)
			assert.Equal(t, 1, childCloses)
		})
	}
}

type cancelingParentChild struct {
	Parent *LifecycleParentOptions `parent:"application"`
	linked *bool
}

func (options *cancelingParentChild) Close() error {
	*options.linked = options.Parent != nil
	return nil
}
func TestCanceledFactoryRetainsParentDependencyForCleanup(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	linked := false
	closes := 0
	app := NewApplication[*LifecycleParentOptions]("app").
		WithHelp(false).
		WithErrorHandlerEnabled(false).
		WithCreateOptionsHandler(func(context.Context) (*LifecycleParentOptions, error) {
			return &LifecycleParentOptions{closes: &closes}, nil
		}).
		WithCommand(NewCommand[*cancelingParentChild]("run").
			WithCreateOptionsHandler(func(context.Context) (*cancelingParentChild, error) {
				cancel()
				return &cancelingParentChild{linked: &linked}, nil
			}),
		).
		MustCreate()
	defer func() { require.NoError(t, app.Close()) }()
	invocation, err := app.Parse(ctx, nil, "run")
	assert.Nil(t, invocation)
	require.ErrorIs(t, err, context.Canceled)
	assert.True(t, linked)
	assert.Equal(t, 1, closes)
}

func TestParentInjectionRejectsInvalidSchemasAndConflictingFactoryPointers(t *testing.T) {
	for _, test := range []struct {
		name  string
		child CommandRequirementLike
		text  string
	}{
		{"wrong-root-type", NewCommand[*struct {
			Parent *sharedParentOptions `parent:"application"`
		}]("run"), "no application ancestor"},
		{"no-match", NewCommand[*struct {
			Parent *sharedParentOptions `parent:"matchingType"`
		}]("run"), "no matchingType ancestor"},
		{"unknown-mode", NewCommand[*struct {
			Parent *NoOptions `parent:"command:a"`
		}]("run"), "invalid parent tag"},
		{"not-pointer", NewCommand[*struct {
			Parent NoOptions `parent:"matchingType"`
		}]("run"), "pointer to struct"},
		{"mixed-tag", NewCommand[*struct {
			Parent *NoOptions `parent:"application" flag:"parent"`
		}]("run"), "also be a flag"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewSimpleApplication("app").
				WithCommand(test.child).
				Create()
			require.ErrorContains(t, err, test.text)
		})
	}
	_, err := NewApplication[*LifecycleParentOptions]("app").WithCommand(NewCommand[*struct {
		*LifecycleParentOptions `parent:"application"`
	}]("run")).
		Create()
	require.ErrorContains(t, err, "promotes lifecycle methods")
	_, err = NewApplication[*sharedParentOptions]("app").
		WithCommand(NewCommand[*struct {
			Parent *sharedParentOptions `parent:"application"`
		}]("run").
			WithStringFlag("overlap", func(f *FlagRequirement[string, string]) { f.WithField("Parent", "Name") })).
		Create()
	require.ErrorContains(t, err, "overlaps an explicit flag")
	type childOptions struct {
		Parent *sharedParentOptions `parent:"application"`
	}
	app := NewApplication[*sharedParentOptions]("app").
		WithHelp(false).
		WithErrorHandlerEnabled(false).
		WithCommand(NewCommand[*childOptions]("run").
			WithCreateOptionsHandler(func(context.Context) (*childOptions, error) {
				return &childOptions{Parent: new(sharedParentOptions)}, nil
			}),
		).
		MustCreate()
	defer func() { require.NoError(t, app.Close()) }()
	invocation, err := app.Parse(nil, nil, "run")
	assert.Nil(t, invocation)
	require.ErrorContains(t, err, "conflicting factory-provided pointer")
}
