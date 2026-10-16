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

func TestArgumentRequirementChecksDoNotCloneCodecs(t *testing.T) {
	cloneErr := errors.New("argument codec clone failed")
	for _, test := range []struct {
		name       string
		min, max   int
		required   bool
		args       []string
		cloneErr   error
		wantClones int64
		wantError  error
		errorText  string
		wantValue  any
	}{
		{
			name: "omitted optional argument", min: 1, max: 1, cloneErr: cloneErr,
		},
		{
			name: "omitted variadic argument", min: 0, max: -1, cloneErr: cloneErr,
		},
		{
			name: "missing required argument", min: 1, max: 1, required: true, cloneErr: cloneErr,
			errorText: "required argument value is missing",
		},
		{
			name: "incomplete multi-value argument", min: 2, max: 2, args: []string{"one"}, cloneErr: cloneErr,
			wantError: codec.ErrTooFewArguments,
		},
		{
			name: "provided argument", min: 1, max: 1, args: []string{"one"},
			wantValue: "one",
		},
		{
			name: "provided variadic argument", min: 0, max: -1, args: []string{"one", "two"},
			wantValue: "one|two",
		},
		{
			name: "provided argument with clone error", min: 1, max: 1, args: []string{"one"}, cloneErr: cloneErr,
			wantValue: "one",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			clones := &atomic.Int64{}
			app, err := NewSimpleApplication("app").
				WithArgument(
					NewArgument(grammarString{
						min: test.min, max: test.max, clones: clones, cloneErr: test.cloneErr,
					}, "value").WithRequired(test.required),
				).
				Create()
			require.NoError(t, err)
			invocation, err := app.Parse(context.Background(), nil, test.args...)
			assert.Equal(t, test.wantClones, clones.Load())
			if test.wantError != nil || test.errorText != "" {
				require.Error(t, err)
				assert.Nil(t, invocation)
				if test.wantError != nil {
					assert.ErrorIs(t, err, test.wantError)
				}
				if test.errorText != "" {
					assert.ErrorContains(t, err, test.errorText)
				}
				if test.wantClones == 0 {
					assert.NotErrorIs(t, err, cloneErr)
				}
				return
			}
			require.NoError(t, err)
			defer func() { assert.NoError(t, invocation.Close()) }()
			argument, value := invocation.Argument(0)
			assert.Equal(t, test.wantValue, value)
			if len(test.args) == 0 {
				assert.Nil(t, argument)
			} else {
				require.NotNil(t, argument)
				assert.Equal(t, "value", argument.Name())
			}
		})
	}
}

func TestBoolCodecUsageForFlagAndArgument(t *testing.T) {
	ctx := context.Background()
	decoder := codec.NewBool()
	//goland:noinspection GoShadowedVar,GoReservedWordUsedAsName
	min, max := decoder.ExpectMinMax(context.Background(), codec.UsageFlag)
	assert.Equal(t, 0, min)
	assert.Equal(t, 1, max)
	min, max = decoder.ExpectMinMax(context.Background(), codec.UsageArgument)
	assert.Equal(t, 1, min)
	assert.Equal(t, 1, max)
	value, err := decoder.ParseRequirement(ctx, codec.UsageFlag)
	require.NoError(t, err)
	assert.True(t, value)
	_, err = decoder.ParseRequirement(ctx, codec.UsageArgument)
	require.ErrorIs(t, err, codec.ErrArgumentRequired)

	type options struct {
		Ready   bool `flag:"ready" type:"boolean"`
		Enabled bool `argument:"enabled" required:"true" type:"boolean"`
	}
	app, err := NewApplication[*options]("bool").
		WithCodec(decoder, "boolean").Create()
	require.NoError(t, err)
	for _, test := range []struct {
		args           []string
		ready, enabled bool
	}{
		{[]string{"--ready", "false"}, true, false},
		{[]string{"--ready=false", "true"}, false, true},
	} {
		inv, err := app.Parse(ctx, nil, test.args...)
		require.NoError(t, err)
		_, value := inv.Application()
		got := value
		assert.Equal(t, test.ready, got.Ready, "args %v", test.args)
		assert.Equal(t, test.enabled, got.Enabled, "args %v", test.args)
		require.NoError(t, inv.Close())
	}
}

func TestVariadicAndCodecArity(t *testing.T) {
	require.NoError(t, codec.EnsureArgumentsMatching(context.Background(), codec.UsageFlag, codec.NewString(), "ok"))
	require.NoError(t, codec.EnsureArgumentsMatching(context.Background(), codec.UsageFlag, customVariadic1Codec{}, "a", "b"))
	a, err := NewSimpleApplication("var").
		WithArgument(
			NewArgument(codec.NewString(), "first"),
			NewArgument(customVariadic1Codec{}, "rest"),
		).
		Create()
	require.NoError(t, err)
	i, err := a.Parse(context.Background(), nil, "one", "two", "three")
	require.NoError(t, err)
	defer func() { assert.NoError(t, i.Close()) }()
	argument, value := i.Argument(1)
	assert.NotNil(t, argument)
	assert.Equal(t, customVariadic1{[]string{"two", "three"}}, value)
}

func TestSliceCodecCreatesIndependentRuntimeSlices(t *testing.T) {
	type options struct {
		Values []int `argument:"values"`
	}
	a, err := NewApplication[*options]("slices").
		WithCodec(codec.NewIntSlice()).
		Create()
	require.NoError(t, err)
	var wg sync.WaitGroup
	for range 30 {
		wg.Go(func() {
			inv, err := a.Parse(context.Background(), nil, "1", "2")
			if !assert.NoError(t, err) {
				return
			}
			defer func() { assert.NoError(t, inv.Close()) }()
			_, parsed := inv.Application()
			assert.Equal(t, []int{1, 2}, parsed.Values)
			parsed.Values[0] = 99
		})
	}
	wg.Wait()
}

func TestManyFixedArgumentsRetainDefinitionOrder(t *testing.T) {
	requirement := NewSimpleApplication("app")
	args := make([]string, 160)
	for i := range args {
		name := "value" + strconv.Itoa(i)
		requirement.WithArgument(NewArgument(codec.NewString(), name))
		args[i] = strconv.Itoa(i)
	}
	app, err := requirement.Create()
	require.NoError(t, err)
	invocation, err := app.Parse(context.Background(), nil, args...)
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	i := 0
	for argument, value := range invocation.Arguments() {
		assert.Equal(t, "value"+strconv.Itoa(i), argument.Name())
		assert.Equal(t, args[i], value)
		i++
	}
	assert.Equal(t, len(args), i)
}

func TestArgumentLookupUsesDefinitionIndexWhenEarlierOptionalArgumentIsOmitted(t *testing.T) {
	type options struct {
		First  string `argument:"first"`
		Second string `argument:"second" default:"fallback"`
	}
	app, err := NewApplication[options]("app").
		Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, app.Close()) }()
	for _, test := range []struct {
		name   string
		args   []string
		first  string
		second string
	}{
		{name: "only later default", second: "fallback"},
		{name: "earlier input and later default", args: []string{"input"}, first: "input", second: "fallback"},
		{name: "both supplied", args: []string{"one", "two"}, first: "one", second: "two"},
	} {
		t.Run(test.name, func(t *testing.T) {
			invocation, err := app.Parse(nil, nil, test.args...)
			require.NoError(t, err)
			defer func() { assert.NoError(t, invocation.Close()) }()
			for _, afterClose := range []bool{false, true} {
				if afterClose {
					require.NoError(t, invocation.Close())
				}
				firstDefinition, found := app.Argument(0)
				require.True(t, found)
				first, value := invocation.Argument(0)
				if len(test.args) == 0 {
					assert.Nil(t, first)
					assert.Nil(t, value)
				} else {
					assert.Same(t, firstDefinition, first)
					assert.Equal(t, test.first, value)
				}
				secondDefinition, found := app.Argument(1)
				require.True(t, found)
				second, value := invocation.Argument(1)
				assert.Same(t, secondDefinition, second)
				assert.Equal(t, test.second, value)
				for _, index := range []int{-1, 2, 100} {
					argument, value := invocation.Argument(index)
					assert.Nil(t, argument)
					assert.Nil(t, value)
				}
				var names []string
				var values []any
				for argument, value := range invocation.Arguments() {
					names = append(names, argument.Name())
					values = append(values, value)
				}
				if len(test.args) == 0 {
					assert.Equal(t, []string{"second"}, names)
					assert.Equal(t, []any{test.second}, values)
				} else {
					assert.Equal(t, []string{"first", "second"}, names)
					assert.Equal(t, []any{test.first, test.second}, values)
				}
			}
		})
	}
}

func TestArgumentDefinitionIndicesAreLocalToSelectedLeafAndVariadicDefaults(t *testing.T) {
	type childOptions struct {
		First string
		Rest  []string
	}
	app := NewSimpleApplication("app").
		WithArgument(NewArgument(codec.NewString(), "root").
			WithDefault("root default"),
		).
		WithCommand(NewCommand[childOptions]("child").
			WithArgument(
				NewArgument(codec.NewString(), "first").
					WithField("First"),
				NewArgument(codec.NewStrings(), "rest").
					WithField("Rest").
					WithDefaultStr("one", "two"),
			),
		).
		MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(nil, nil, "child")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	child, found := FindCommand[childOptions](app, "child")
	require.True(t, found)
	first, value := invocation.Argument(0)
	assert.Nil(t, first)
	assert.Nil(t, value)
	definition, found := child.Argument(1)
	require.True(t, found)
	second, value := invocation.Argument(1)
	assert.Same(t, definition, second)
	assert.Equal(t, []string{"one", "two"}, value)
	var names []string
	for argument := range invocation.Arguments() {
		names = append(names, argument.Name())
	}
	assert.Equal(t, []string{"rest"}, names, "parent values must not shift leaf indices")
}
