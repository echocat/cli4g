package cli

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/echocat/cli4g/codec"
)

func TestDefaultsFollowSourcePrecedenceAndReplaceFactorySlices(t *testing.T) {
	type options struct {
		Name    string   `flag:"name" env:"DEFAULT_TEST_NAME" default:"tag"`
		Items   []string `flag:"items|item" short:"i"`
		Enabled bool     `flag:"enabled" default:"false"`
		Empty   string   `flag:"empty" default:""`
		Factory string   `flag:"factory"`
	}
	type childOptions struct {
		Name string `argument:"name" default:"child-default"`
	}
	var factories int
	app, err := NewApplication[*options]("app").
		WithCreateOptionsHandler(func(context.Context) (*options, error) {
			factories++
			return &options{
				Name: "factory", Items: []string{"factory"}, Enabled: true,
				Empty: "factory", Factory: "factory-only",
			}, nil
		}).
		WithFlag(NewFlag(codec.NewStrings(), "items").
			WithAliases("item").
			WithShort('i').
			WithField("Items").
			WithDefault([]string{"default-one", "default-two"}),
		).
		WithCommand(NewCommand[childOptions]("child").
			WithCreateOptionsHandler(func(context.Context) (childOptions, error) {
				return childOptions{Name: "child-factory"}, nil
			}),
		).
		Create()
	require.NoError(t, err)
	assert.Zero(t, factories, "Create must not run options factories")
	for _, test := range []struct {
		name  string
		args  []string
		env   bool
		value string
		items []string
	}{
		{
			name: "definition defaults", args: []string{"child"}, value: "tag",
			items: []string{"default-one", "default-two"},
		},
		{
			name: "environment", args: []string{"child"}, env: true, value: "environment",
			items: []string{"default-one", "default-two"},
		},
		{
			name: "CLI", args: []string{"--name=CLI", "--item=one", "child", "-itwo"},
			env: true, value: "CLI", items: []string{"one", "two"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			parseOptions := &ParseOpts{LookupEnv: func(context.Context, string) (string, bool) {
				return "environment", test.env
			}}
			invocation, err := app.Parse(nil, parseOptions, test.args...)
			require.NoError(t, err)
			defer func() { assert.NoError(t, invocation.Close()) }()
			_, parsed := invocation.Application()
			assert.Equal(t, test.value, parsed.Name)
			assert.Equal(t, test.items, parsed.Items)
			assert.False(t, parsed.Enabled)
			assert.Empty(t, parsed.Empty)
			assert.Equal(t, "factory-only", parsed.Factory)
			_, value := invocation.ShortFlag('i')
			assert.Equal(t, test.items, value)
			child, found := FindCommand[childOptions](app, "child")
			require.True(t, found)
			childValue, found := child.OptionsFrom(invocation)
			require.True(t, found)
			assert.Equal(t, "child-default", childValue.Name)
			_, value = invocation.Argument(0)
			assert.Equal(t, "child-default", value)
		})
	}
	assert.Equal(t, 3, factories)
}

func TestTypedDefaultsAreSnapshotsAndIndependentAcrossConcurrentInvocations(t *testing.T) {
	type options struct {
		Values []string
		Zero   int
		Nil    []string
	}
	prototype := []string{"original"}
	var factories atomic.Int64
	app, err := NewApplication[*options]("app").
		WithCreateOptionsHandler(func(context.Context) (*options, error) {
			factories.Add(1)
			return &options{Zero: 99, Nil: []string{"factory"}}, nil
		}).
		WithFlag(
			NewFlag(codec.NewStrings(), "values").
				WithField("Values").
				WithDefault(prototype),
			NewFlag(codec.NewInt(), "zero").
				WithField("Zero").
				WithDefault(0),
			NewFlag(codec.NewStrings(), "nil").
				WithField("Nil").
				WithDefault(nil),
		).
		Create()
	require.NoError(t, err)
	prototype[0] = "changed after Create"
	flag, found := app.Flag("values")
	require.True(t, found)
	formatted, found, err := flag.FormatDefault(nil)
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, []string{"original"}, formatted)
	assert.Zero(t, factories.Load(), "formatting a default must not create options")
	value, found, err := flag.Default(nil)
	require.NoError(t, err)
	require.True(t, found)
	value.([]string)[0] = "changed getter result"
	var workers sync.WaitGroup
	for n := range 32 {
		workers.Go(func() {
			invocation, err := app.Parse(context.Background(), nil)
			if !assert.NoError(t, err) {
				return
			}
			defer func() { assert.NoError(t, invocation.Close()) }()
			_, parsed := invocation.Application()
			assert.Equal(t, []string{"original"}, parsed.Values)
			assert.Zero(t, parsed.Zero)
			assert.Nil(t, parsed.Nil)
			parsed.Values[0] = strconv.Itoa(n)
		})
	}
	workers.Wait()
	assert.EqualValues(t, 32, factories.Load())
	formatted, _, err = flag.FormatDefault(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{"original"}, formatted)
}

func TestDefaultSettersReplaceEachOtherAndStringsDescribeCompleteValues(t *testing.T) {
	flag := NewFlag(codec.NewInt(), "value").
		WithDefault(1).
		WithDefaultStr("2")
	flag.SetDefault(3)
	argument := NewArgument(codec.NewStrings(), "values").
		WithDefault([]string{"old"})
	argument.SetDefaultStr("one two", "three,four")
	app, err := NewSimpleApplication("app").
		WithFlag(flag).
		WithArgument(argument).
		Create()
	require.NoError(t, err)
	invocation, err := app.Parse(context.Background(), nil)
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, value := invocation.Flag("value")
	assert.Equal(t, 3, value)
	_, value = invocation.Argument(0)
	assert.Equal(t, []string{"one two", "three,four"}, value)
	definition, found := app.Argument(0)
	require.True(t, found)
	formatted, found, err := definition.FormatDefault(context.Background())
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, []string{"one two", "three,four"}, formatted)
}

func TestRequiredAndDefaultAreRejectedIncludingEmptyAndZeroValues(t *testing.T) {
	for _, test := range []struct {
		name  string
		build func() error
	}{
		{
			name: "typed flag zero",
			build: func() error {
				_, err := NewSimpleApplication("app").
					WithFlag(NewFlag(codec.NewInt(), "value").
						WithDefault(0).
						WithRequired(true),
					).
					Create()
				return err
			},
		},
		{
			name: "string flag empty",
			build: func() error {
				_, err := NewSimpleApplication("app").
					WithFlag(NewFlag(codec.NewString(), "value").
						WithRequired(true).
						WithDefaultStr(""),
					).
					Create()
				return err
			},
		},
		{
			name: "typed argument nil",
			build: func() error {
				_, err := NewSimpleApplication("app").
					WithArgument(NewArgument(codec.NewStrings(), "value").WithDefault(nil).WithRequired(true)).Create()
				return err
			},
		},
		{
			name: "tagged flag empty",
			build: func() error {
				_, err := NewApplication[struct {
					Value string `flag:"value" required:"true" default:""`
				}]("app").Create()
				return err
			},
		},
		{
			name: "tagged argument",
			build: func() error {
				_, err := NewApplication[struct {
					Value string `argument:"value" required:"true" default:"default"`
				}]("app").Create()
				return err
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			require.ErrorContains(t, test.build(), "required cannot be combined with a default")
		})
	}
}

func TestStringDefaultsAreDecodedOnlyWhenNeededAndErrorsHaveDefinitionNames(t *testing.T) {
	type options struct {
		Number int `flag:"number" env:"DEFAULT_TEST_NUMBER" default:"invalid"`
	}
	app, err := NewApplication[options]("app").
		Create()
	require.NoError(t, err)
	noEnv := &ParseOpts{LookupEnv: func(context.Context, string) (string, bool) { return "", false }}
	invocation, err := app.Parse(context.Background(), noEnv)
	require.ErrorContains(t, err, "flag --number default")
	assert.Nil(t, invocation)
	invocation, err = app.Parse(context.Background(), noEnv, "--number=7")
	require.NoError(t, err)
	require.NoError(t, invocation.Close())
	_, value := invocation.Application()
	assert.Equal(t, 7, value.Number)
	env := &ParseOpts{LookupEnv: func(context.Context, string) (string, bool) { return "8", true }}
	invocation, err = app.Parse(context.Background(), env)
	require.NoError(t, err)
	require.NoError(t, invocation.Close())
	_, value = invocation.Application()
	assert.Equal(t, 8, value.Number)
}

func TestDecodedDefaultsRemainOwnedOnCancellationOrLaterRequiredFailure(t *testing.T) {
	for _, cancelDuringDefault := range []bool{false, true} {
		name := "later missing required flag"
		if cancelDuringDefault {
			name = "cancellation during default"
		}
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var events []string
			var resource *ownedTestResource
			decoder := callbackCodec[*ownedTestResource, string]{
				Codec: codec.NewMarshallerP(func() *ownedTestResource {
					resource = &ownedTestResource{events: &events}
					return resource
				}),
				onParse: func(context.Context) {
					if cancelDuringDefault {
						cancel()
					}
				},
			}
			app, err := NewApplication[*resourceOptions]("app").
				WithFlag(
					NewFlag(decoder, "resource").
						WithField("Resource").
						WithDefaultStr("default"),
					NewFlag(codec.NewInt(), "count").
						WithField("Count").
						WithRequired(true),
				).
				WithErrorHandler(func(_ context.Context, _ InvocationLike, err error) error {
					// This fixture measures decoded-value ownership, independently of
					// help rendering's additional default formatting and value creation.
					return err
				}).
				Create()
			require.NoError(t, err)
			invocation, err := app.Parse(ctx, nil)
			assert.Nil(t, invocation)
			if cancelDuringDefault {
				require.ErrorIs(t, err, context.Canceled)
			} else {
				require.ErrorContains(t, err, "required flag --count")
			}
			assert.Nil(t, resource, "required checks and cancellation finish before acquisition")
			assert.Empty(t, events)
		})
	}
}

func TestNoDefaultAndCanceledDefaultAccess(t *testing.T) {
	app, err := NewSimpleApplication("app").
		WithFlag(
			NewFlag(codec.NewString(), "absent"),
			NewFlag(codec.NewString(), "present").
				WithDefault("value"),
		).
		Create()
	require.NoError(t, err)
	absent, found := app.Flag("absent")
	require.True(t, found)
	assert.False(t, absent.HasDefault())
	_, found, err = absent.Default(nil)
	require.NoError(t, err)
	assert.False(t, found)
	present, found := app.Flag("present")
	require.True(t, found)
	assert.True(t, present.HasDefault())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, found, err = present.Default(ctx)
	assert.True(t, found)
	assert.ErrorIs(t, err, context.Canceled)
	_, _, err = present.FormatDefault(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestDefaultSnapshotCloneFailureIsACompileError(t *testing.T) {
	failure := errors.New("cannot snapshot codec")
	_, err := NewSimpleApplication("app").
		WithFlag(
			NewFlag(grammarString{min: 1, max: 1, cloneErr: failure}, "value").
				WithDefault("default"),
		).
		Create()
	require.ErrorIs(t, err, failure)
	require.ErrorContains(t, err, "flag --value default")
}

func TestTypedRequirementCancellationDoesNotAcquireResource(t *testing.T) {
	type parseKey struct{}
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), parseKey{}, true))
	defer cancel()
	var events []string
	var cloned *ownedTestResource
	decoder := callbackCodec[*ownedTestResource, string]{
		Codec: codec.NewMarshallerP(func() *ownedTestResource {
			cloned = &ownedTestResource{events: &events}
			return cloned
		}),
		onCloneValue: func(ctx context.Context) {
			if ctx.Value(parseKey{}) != nil {
				cancel()
			}
		},
	}
	prototype := "default"
	app, err := NewApplication[*resourceOptions]("app").
		WithFlag(NewFlag(decoder, "resource").
			WithField("Resource").
			WithDefault(prototype),
		).
		Create()
	require.NoError(t, err)
	assert.Nil(t, cloned, "Create snapshots the requirement, not the resource")
	invocation, err := app.Parse(ctx, nil)
	assert.Nil(t, invocation)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, cloned)
	assert.Empty(t, events)
}

type rewritingDefaultCodec struct{ grammarString }

func (c rewritingDefaultCodec) ToRaw(ctx context.Context) codec.Codec[any, any] {
	return codec.Raw(ctx, c)
}
func (c rewritingDefaultCodec) Clone(context.Context) (codec.Codec[string, string], error) {
	return c, nil
}
func (c rewritingDefaultCodec) CloneDirect() codec.Codec[string, string] { return c }
func (c rewritingDefaultCodec) ParseRequirement(ctx context.Context, usage codec.Usage, args ...string) (string, error) {
	value, err := c.grammarString.Parse(ctx, usage, args...)
	if len(args) > 0 {
		args[0] = "rewritten"
	}
	return value, err
}

func TestStringDefaultTokensRemainIndependentWhenCodecRewritesItsInput(t *testing.T) {
	app, err := NewSimpleApplication("app").
		WithFlag(NewFlag(rewritingDefaultCodec{grammarString{min: 1, max: 1}}, "value").
			WithDefaultStr("original"),
		).
		Create()
	require.NoError(t, err)
	var workers sync.WaitGroup
	for range 32 {
		workers.Go(func() {
			invocation, err := app.Parse(context.Background(), nil)
			if !assert.NoError(t, err) {
				return
			}
			defer func() { assert.NoError(t, invocation.Close()) }()
			_, value := invocation.Flag("value")
			assert.Equal(t, "original", value)
		})
	}
	workers.Wait()
}
