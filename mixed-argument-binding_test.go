package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExplicitArgumentOverridePreservesTaggedPosition(t *testing.T) {
	type options struct {
		First  string `argument:"first" required:"true"`
		Second string `argument:"second" required:"true"`
	}
	app := NewApplication[options]("app").
		WithStringArgument("replacement", func(a *ArgumentRequirement[string, string]) {
			a.WithField("Second").WithRequired(true)
		}).
		MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(context.Background(), nil, "one", "two")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, parsed := invocation.Application()
	assert.Equal(t, options{First: "one", Second: "two"}, parsed)
	first, _ := app.Argument(0)
	second, _ := app.Argument(1)
	assert.Equal(t, "first", first.Name())
	assert.Equal(t, "replacement", second.Name())
}

type EmbeddedArgumentGroup struct {
	Second string   `argument:"second" required:"true"`
	Rest   []string `argument:"rest"`
}

func TestExplicitVariadicOverridePreservesEmbeddedTaggedPosition(t *testing.T) {
	type options struct {
		First string `argument:"first" required:"true"`
		*EmbeddedArgumentGroup
	}
	app := NewApplication[options]("app").
		WithStringSliceArgument("remaining", func(a *ArgumentRequirement[[]string, []string]) {
			a.WithField("EmbeddedArgumentGroup", "Rest")
		}).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(context.Background(), nil, "one", "two", "three", "four")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, parsed := invocation.Application()
	assert.Equal(t, "one", parsed.First)
	assert.Equal(t, "two", parsed.Second)
	assert.Equal(t, []string{"three", "four"}, parsed.Rest)
}

type NamedArgumentGroup struct {
	Value string `argument:"value"`
}

func TestExplicitArgumentInsideNamedSubobjectIsNotLost(t *testing.T) {
	type options struct {
		Nested *NamedArgumentGroup
	}
	app := NewApplication[*options]("app").
		WithErrorHandlerEnabled(false).
		WithStringArgument("value", func(a *ArgumentRequirement[string, string]) {
			a.WithField("Nested", "Value").WithRequired(true)
		}).
		MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	_, err := app.Parse(context.Background(), nil)
	require.ErrorIs(t, err, ErrValidation)
	assert.ErrorContains(t, err, "required argument value")
	invocation, err := app.Parse(context.Background(), nil, "input")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, parsed := invocation.Application()
	require.NotNil(t, parsed.Nested)
	assert.Equal(t, "input", parsed.Nested.Value)
}

func TestFullyExplicitArgumentsKeepRegistrationOrderWithIgnoredTags(t *testing.T) {
	type options struct {
		First  string `argument:"old-name" required:"invalid"`
		Second string
		Rest   []string
	}
	app := NewApplication[options]("app").
		WithStringArgument("first", func(a *ArgumentRequirement[string, string]) { a.WithField("First") }).
		WithStringArgument("second", func(a *ArgumentRequirement[string, string]) { a.WithField("Second") }).
		WithStringSliceArgument("rest", func(a *ArgumentRequirement[[]string, []string]) { a.WithField("Rest") }).
		MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(context.Background(), nil, "one", "two", "three", "four")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, parsed := invocation.Application()
	assert.Equal(t, options{First: "one", Second: "two", Rest: []string{"three", "four"}}, parsed)
	for index, name := range []string{"first", "second", "rest"} {
		argument, found := app.Argument(index)
		require.True(t, found)
		assert.Equal(t, name, argument.Name())
	}
}

func TestMixedArgumentsKeepBuilderOnlyPrefixAndReachableTaggedSlots(t *testing.T) {
	type options struct {
		Prefix string
		First  string `arg:"first"`
		Second string `argument:"second"`
		Nested *NamedArgumentGroup
	}
	app := NewApplication[options]("app").
		WithStringArgument("prefix", func(a *ArgumentRequirement[string, string]) { a.WithField("Prefix") }).
		WithStringArgument("replacement", func(a *ArgumentRequirement[string, string]) { a.WithField("Second") }).
		WithStringArgument("nested", func(a *ArgumentRequirement[string, string]) { a.WithField("Nested", "Value") }).
		MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(context.Background(), nil, "prefix", "nested", "first", "second")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, parsed := invocation.Application()
	assert.Equal(t, "prefix", parsed.Prefix)
	assert.Equal(t, "nested", parsed.Nested.Value)
	assert.Equal(t, "first", parsed.First)
	assert.Equal(t, "second", parsed.Second)
}
