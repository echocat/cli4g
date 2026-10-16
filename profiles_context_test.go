package cli

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/echocat/cli4g/codec"
)

type profileContextCodec struct {
	codec.Codec[string, string]
	check func(context.Context, string)
}

func (c profileContextCodec) ParseRequirement(ctx context.Context, usage codec.Usage, args ...string) (string, error) {
	c.check(ctx, "decode")
	return c.Codec.ParseRequirement(ctx, usage, args...)
}

func (c profileContextCodec) Resolve(ctx context.Context, scope codec.Scope, value string) (string, error) {
	c.check(ctx, "resolve")
	return c.Codec.Resolve(ctx, scope, value)
}

func (c profileContextCodec) ToRaw(ctx context.Context) codec.Codec[any, any] {
	return codec.Raw(ctx, c)
}

func (c profileContextCodec) Clone(context.Context) (codec.Codec[string, string], error) {
	return c, nil
}

func (c profileContextCodec) CloneDirect() codec.Codec[string, string] { return c }

func TestProfilesContextThroughoutLifecycle(t *testing.T) {
	type options struct {
		Value string
		ctx   context.Context
	}
	var events []string
	var factoryContext context.Context
	check := func(ctx context.Context, phase string) {
		profiles, found := ProfilesFromContext(ctx)
		require.True(t, found, phase)
		assert.Equal(t, []string{"A", "B"}, slices.Collect(profiles), phase)
		assert.True(t, ContextContainsProfile(ctx, "A"), phase)
		assert.True(t, ContextContainsProfile(ctx, "B"), phase)
		assert.False(t, ContextContainsProfile(ctx, "a"), phase)
		assert.False(t, ContextContainsProfile(ctx, "unknown"), phase)
		if invocation, found := InvocationFromContext(ctx); found {
			assert.Equal(t, []string{"A", "B"}, slices.Collect(invocation.Profiles()), phase)
			assert.True(t, invocation.ContainsProfile("A"), phase)
		}
		events = append(events, phase)
	}
	app := NewApplication[*options]().WithName("app").WithFlag(
		NewFlag(profileContextCodec{Codec: codec.NewString(), check: check}).WithName("value").WithProfiles("A", "B").WithField("Value"),
		NewFlag(codec.NewString()).WithName("environment").WithProfiles("A", "B").WithEnvVar("VALUE"),
	).WithCreateOptionsHandler(func(ctx context.Context) (*options, error) {
		check(ctx, "factory")
		factoryContext = ctx
		_, found := InvocationFromContext(ctx)
		assert.False(t, found, "profile access must not publish an incomplete invocation")
		return &options{}, nil
	}).WithValidateHandler(func(ctx context.Context, _ *options) error {
		check(ctx, "validate")
		return nil
	}).WithInitializeHandler(func(ctx context.Context, opts *options) error {
		check(ctx, "initialize")
		opts.ctx = ctx
		return nil
	}).WithHandler(func(ctx context.Context, _ *options) error {
		check(ctx, "execute")
		return nil
	}).WithCloseHandler(func(opts *options) error {
		check(opts.ctx, "close")
		return nil
	}).WithCommand(NewSimpleCommand().WithName("child").WithCreateOptionsHandler(func(ctx context.Context) (NoOptions, error) {
		check(ctx, "command factory")
		return NoOptions{}, nil
	}).WithHandler(func(ctx context.Context, _ NoOptions) error {
		check(ctx, "command execute")
		return nil
	})).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	parseOptions := &ParseOpts{Profiles: []string{"B", "A", "B"}, LookupEnv: func(ctx context.Context, _ string) (string, bool) {
		check(ctx, "environment")
		return "env-value", true
	}}
	callerContext := context.Background()
	invocation, err := app.Parse(callerContext, parseOptions, "child", "--value", "hello")
	require.NoError(t, err)
	assert.Equal(t, []string{"B", "A", "B"}, parseOptions.Profiles, "normalization must not modify caller data")
	parseOptions.Profiles[0] = "mutated"
	assert.Equal(t, []string{"A", "B"}, slices.Collect(invocation.Profiles()))
	assert.True(t, invocation.ContainsProfile("B"))
	assert.False(t, invocation.ContainsProfile("mutated"))
	assert.NoError(t, invocation.Execute(context.Background()))
	assert.NoError(t, invocation.Close())
	check(factoryContext, "retained factory context")
	assert.Equal(t, []string{"A", "B"}, slices.Collect(invocation.Profiles()), "metadata survives Close")
	assert.Equal(t, []string{"factory", "command factory", "decode", "environment", "resolve", "validate", "initialize", "execute", "command execute", "close", "retained factory context"}, events)
	_, found := ProfilesFromContext(callerContext)
	assert.False(t, found, "the caller's context must not be mutated")
	seen := 0
	for range invocation.Profiles() {
		seen++
		break
	}
	assert.Equal(t, 1, seen)
}

func TestProfilesContextHelpAndErrors(t *testing.T) {
	sentinel := errors.New("test failure")
	for _, test := range []struct {
		name         string
		profiles     []string
		args         []string
		factoryError bool
		executeError bool
		wantError    bool
	}{
		{name: "help", profiles: []string{"A"}, args: []string{"--help"}},
		{name: "help without selection", args: []string{"--help"}},
		{name: "unknown profile", profiles: []string{"unknown"}, args: []string{"--help"}, wantError: true},
		{name: "conflicting profiles", profiles: []string{"A", "B"}, wantError: true},
		{name: "syntax error", profiles: []string{"B"}, args: []string{"--unknown"}, wantError: true},
		{name: "factory error", profiles: []string{"A"}, factoryError: true, wantError: true},
		{name: "execute error", profiles: []string{"B"}, executeError: true, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			called := false
			check := func(ctx context.Context, invocation InvocationLike) {
				called = true
				profiles, found := ProfilesFromContext(ctx)
				require.True(t, found)
				assert.Equal(t, test.profiles, slices.Collect(profiles))
				assert.Equal(t, test.profiles, slices.Collect(invocation.Profiles()))
				for _, profile := range test.profiles {
					assert.True(t, invocation.ContainsProfile(profile))
					assert.True(t, ContextContainsProfile(ctx, profile))
				}
			}
			app := NewSimpleApplication().WithName("app").WithFlag(
				NewFlag(codec.NewString()).WithName("foo").WithProfiles("A"),
				NewFlag(codec.NewInt32()).WithName("foo").WithProfiles("B"),
			).WithCreateOptionsHandler(func(context.Context) (NoOptions, error) {
				if test.factoryError {
					return NoOptions{}, sentinel
				}
				return NoOptions{}, nil
			}).WithHandler(func(context.Context, NoOptions) error {
				if test.executeError {
					return sentinel
				}
				return nil
			}).WithHelpHandler(func(ctx context.Context, invocation InvocationLike) error {
				check(ctx, invocation)
				return nil
			}).WithErrorHandler(func(ctx context.Context, invocation InvocationLike, _ error) error {
				check(ctx, invocation)
				return nil
			}).MustCreate()
			defer func() { assert.NoError(t, app.Close()) }()
			err := app.ParseAndExecute(nil, &ParseOpts{Profiles: test.profiles}, test.args...)
			assert.Equal(t, test.wantError, err != nil)
			assert.True(t, called)
		})
	}
}

func TestProfilesContextNestedParsesOverrideOuterSelection(t *testing.T) {
	innerCalls := 0
	innerExecutions := 0
	inner := NewSimpleApplication().WithName("inner").WithFlag(NewFlag(codec.NewBool()).WithName("enabled").WithProfiles("B")).
		WithCreateOptionsHandler(func(ctx context.Context) (NoOptions, error) {
			profiles, found := ProfilesFromContext(ctx)
			require.True(t, found)
			if innerCalls == 0 {
				assert.Empty(t, slices.Collect(profiles))
				assert.False(t, ContextContainsProfile(ctx, "A"))
			} else {
				assert.Equal(t, []string{"B"}, slices.Collect(profiles))
			}
			innerCalls++
			return NoOptions{}, nil
		}).WithHandler(func(ctx context.Context, _ NoOptions) error {
		profiles, found := ProfilesFromContext(ctx)
		require.True(t, found)
		if innerExecutions == 0 {
			assert.Empty(t, slices.Collect(profiles))
		} else {
			assert.Equal(t, []string{"B"}, slices.Collect(profiles))
		}
		assert.False(t, ContextContainsProfile(ctx, "A"))
		innerExecutions++
		return nil
	}).MustCreate()
	defer func() { assert.NoError(t, inner.Close()) }()
	var outerContext context.Context
	outer := NewSimpleApplication().WithName("outer").WithFlag(NewFlag(codec.NewBool()).WithName("enabled").WithProfiles("A")).
		WithHandler(func(ctx context.Context, _ NoOptions) error {
			outerContext = ctx
			first, err := inner.Parse(ctx, nil)
			require.NoError(t, err)
			assert.Empty(t, slices.Collect(first.Profiles()))
			second, err := inner.Parse(ctx, &ParseOpts{Profiles: []string{"B"}})
			require.NoError(t, err)
			assert.True(t, second.ContainsProfile("B"))
			assert.False(t, second.ContainsProfile("A"))
			// An Execute context from another invocation must not replace the
			// original selection, even when that original selection was empty.
			assert.NoError(t, first.Execute(ctx))
			assert.NoError(t, first.Close())
			assert.NoError(t, second.Execute(ctx))
			assert.NoError(t, second.Close())
			assert.True(t, ContextContainsProfile(ctx, "A"))
			assert.False(t, ContextContainsProfile(ctx, "B"))
			return nil
		}).MustCreate()
	defer func() { assert.NoError(t, outer.Close()) }()
	assert.NoError(t, outer.ParseAndExecute(nil, &ParseOpts{Profiles: []string{"A"}}))
	assert.True(t, ContextContainsProfile(outerContext, "A"))
	assert.Equal(t, 2, innerCalls)
	assert.Equal(t, 2, innerExecutions)
}

func TestProfilesContextMissingAndConcurrentSelections(t *testing.T) {
	for _, ctx := range []context.Context{nil, context.Background()} {
		profiles, found := ProfilesFromContext(ctx)
		assert.False(t, found)
		assert.Empty(t, slices.Collect(profiles))
		assert.False(t, ContextContainsProfile(ctx, "A"))
	}
	var empty Invocation[NoOptions]
	assert.Empty(t, slices.Collect(empty.Profiles()))
	assert.False(t, empty.ContainsProfile("A"))
	type markerKey struct{}
	app := NewSimpleApplication().WithName("app").WithFlag(NewFlag(codec.NewBool()).WithName("enabled").WithProfiles("A", "B")).
		WithCreateOptionsHandler(func(ctx context.Context) (NoOptions, error) {
			profile := ctx.Value(markerKey{}).(string)
			profiles, found := ProfilesFromContext(ctx)
			assert.True(t, found)
			assert.Equal(t, []string{profile}, slices.Collect(profiles))
			return NoOptions{}, nil
		}).WithHandler(func(ctx context.Context, _ NoOptions) error {
		invocation, found := InvocationFromContext(ctx)
		require.True(t, found)
		profiles, found := ProfilesFromContext(ctx)
		assert.True(t, found)
		assert.Equal(t, slices.Collect(invocation.Profiles()), slices.Collect(profiles))
		return nil
	}).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	var workers sync.WaitGroup
	for index := range 16 {
		workers.Go(func() {
			profile := "A"
			if index%2 == 0 {
				profile = "B"
			}
			ctx := context.WithValue(context.Background(), markerKey{}, profile)
			invocation, err := app.Parse(ctx, &ParseOpts{Profiles: []string{profile}})
			if !assert.NoError(t, err) {
				return
			}
			assert.NoError(t, invocation.Execute(nil))
			assert.NoError(t, invocation.Close())
		})
	}
	workers.Wait()
}
