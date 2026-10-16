package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/echocat/cli4g/codec"
)

// Input construction and Create are untimed, including for large values. The
// scanner only needs to decode flag names; value length should not require a
// second Unicode representation of the complete token.
func BenchmarkApplicationParseShortValue(b *testing.B) {
	type options struct {
		Text  string
		Alpha bool `flag:"alpha" short:"a"`
		Beta  bool `flag:"beta" short:"b"`
	}
	for _, size := range []int{16, 1024, 65536} {
		value := strings.Repeat("x", size)
		for _, form := range []struct{ name, prefix string }{
			{"Long", "--text="}, {"Short", "-s="}, {"Unicode", "-界="}, {"Cluster", "-abs="},
		} {
			b.Run(fmt.Sprintf("%s/Bytes=%d", form.name, size), func(b *testing.B) {
				app, err := NewApplication[*options]("app").
					WithFlag(NewFlag(codec.NewString(), "text").
						WithShort('s', '界').
						WithField("Text"),
					).
					Create()
				require.NoError(b, err)
				defer func() { assert.NoError(b, app.Close()) }()
				benchmarkParse(b, app, context.Background(), nil, form.prefix+value)
			})
		}
	}
}

func TestVariadicFlagStopsAtNextOption(t *testing.T) {
	a, err := NewSimpleApplication("variadic").
		WithFlag(
			NewFlag(customVariadic1Codec{}, "items"),
			NewFlag(codec.NewBool(), "verbose"),
		).
		Create()
	require.NoError(t, err)
	i, err := a.Parse(context.Background(), nil, "--items", "one", "two", "--verbose")
	require.NoError(t, err)
	defer func() { assert.NoError(t, i.Close()) }()
	_, value := i.Flag("items")
	assert.Equal(t, customVariadic1{[]string{"one", "two"}}, value)
	_, value = i.Flag("verbose")
	assert.Equal(t, true, value)
}

func TestConcurrentImmutableCodecParsesIndependentValues(t *testing.T) {
	a, err := NewSimpleApplication("immutable").
		WithFlag(
			NewFlag(codec.NewInt(), "count"),
		).
		Create()
	require.NoError(t, err)
	var wg sync.WaitGroup
	for number := range 50 {
		wg.Go(func() {
			i, err := a.Parse(context.Background(), nil, "--count", strconv.Itoa(number))
			if !assert.NoError(t, err) {
				return
			}
			defer func() { assert.NoError(t, i.Close()) }()
			_, value := i.Flag("count")
			assert.Equal(t, number, value)
		})
	}
	wg.Wait()
}

func TestSliceCodecInOptionsAndDefaults(t *testing.T) {
	type options struct {
		Strings []string `flag:"strings"`
		Bools   []bool   `flag:"bools"`
		Numbers []uint8  `argument:"numbers"`
	}
	a, err := NewApplication[*options]("slices").
		Create()
	require.NoError(t, err)
	inv, err := a.Parse(context.Background(), nil, "--strings=one", "--strings=two", "--bools=true", "--bools=false", "3", "4")
	require.NoError(t, err)
	defer func() { assert.NoError(t, inv.Close()) }()
	_, parsed := inv.Application()
	got := parsed
	assert.Equal(t, []string{"one", "two"}, got.Strings)
	assert.Equal(t, []bool{true, false}, got.Bools)
	assert.Equal(t, []uint8{3, 4}, got.Numbers)
}

func TestRepeatedSliceFlags(t *testing.T) {
	type options struct {
		Uint16s []uint16 `flag:"uint16s|nums" short:"u" env:"NUMBERS" required:"true"`
		Single  int      `flag:"single"`
	}
	a := NewApplication[*options]("repeat").
		WithHandler(func(context.Context, *options) error { return nil }).
		WithCommand(NewSimpleCommand("child")).
		MustCreate()
	ctx := context.Background()
	inv := a.MustParse(ctx, nil, "--uint16s=1", "--nums", "2", "-u3")
	defer func() { assert.NoError(t, inv.Close()) }()
	_, parsed := inv.Application()
	assert.Equal(t, []uint16{1, 2, 3}, parsed.Uint16s)
	flag, value := inv.Flag("nums")
	assert.NotNil(t, flag)
	assert.Equal(t, []uint16{1, 2, 3}, value)
	flag, value = inv.ShortFlag('u')
	assert.NotNil(t, flag)
	assert.Equal(t, []uint16{1, 2, 3}, value)
	count := 0
	for range inv.Flags() {
		count++
	}
	assert.Equal(t, 1, count, "expected one aggregated flag entry")

	inherited, err := a.Parse(ctx, nil, "--uint16s=4", "child", "--nums=5")
	require.NoError(t, err)
	defer func() { assert.NoError(t, inherited.Close()) }()
	_, parsed = inherited.Application()
	assert.Equal(t, []uint16{4, 5}, parsed.Uint16s)

	_, err = a.Parse(ctx, nil)
	require.ErrorContains(t, err, "required flag")
	_, err = a.Parse(ctx, nil, "--uint16s")
	require.ErrorIs(t, err, codec.ErrArgumentRequired)
	_, err = a.Parse(ctx, nil, "--uint16s=1", "2", "3")
	require.ErrorIs(t, err, codec.ErrTooManyArguments)
	_, err = a.Parse(ctx, nil, "--uint16s=1", "--single=1", "--single=2")
	require.ErrorContains(t, err, "more than once")

	env := &ParseOpts{LookupEnv: func(_ context.Context, key string) (string, bool) { return "6", key == "NUMBERS" }}
	fromEnv, err := a.Parse(ctx, env)
	require.NoError(t, err)
	defer func() { assert.NoError(t, fromEnv.Close()) }()
	_, parsed = fromEnv.Application()
	assert.Equal(t, []uint16{6}, parsed.Uint16s)
	var wg sync.WaitGroup
	for n := range 30 {
		wg.Go(func() {
			result, err := a.Parse(ctx, nil, "--uint16s="+strconv.Itoa(n), "--nums="+strconv.Itoa(n+1))
			if !assert.NoError(t, err) {
				return
			}
			defer func() { assert.NoError(t, result.Close()) }()
			_, parsed := result.Application()
			assert.Equal(t, []uint16{uint16(n), uint16(n + 1)}, parsed.Uint16s)
		})
	}
	wg.Wait()
}

func TestShortFlagEqualsValuesAndExistingAttachedSeparatedAndClusterForms(t *testing.T) {
	type options struct {
		Text  string `flag:"string" short:"s"`
		Alpha bool   `flag:"alpha" short:"a"`
		Beta  bool   `flag:"beta" short:"b"`
		Gamma bool   `flag:"gamma" short:"c"`
	}
	app, err := NewApplication[options]("app").Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, app.Close()) }()
	for _, test := range []struct {
		name  string
		args  []string
		text  string
		flags string
	}{
		{name: "equals", args: []string{"-s=foo"}, text: "foo"},
		{name: "explicit empty", args: []string{"-s="}, text: ""},
		{name: "only first separator", args: []string{"-s==foo"}, text: "=foo"},
		{name: "attached", args: []string{"-sfoo"}, text: "foo"},
		{name: "separated", args: []string{"-s", "foo"}, text: "foo"},
		{name: "equals inside attached value", args: []string{"-sfoo=bar"}, text: "foo=bar"},
		{name: "cluster", args: []string{"-abc"}, flags: "abc"},
		{name: "cluster with equals value", args: []string{"-abs=foo"}, text: "foo", flags: "ab"},
		{name: "cluster with attached value", args: []string{"-absfoo"}, text: "foo", flags: "ab"},
		{name: "cluster with separated value", args: []string{"-abs", "foo"}, text: "foo", flags: "ab"},
	} {
		t.Run(test.name, func(t *testing.T) {
			invocation, err := app.Parse(nil, nil, test.args...)
			require.NoError(t, err)
			defer func() { assert.NoError(t, invocation.Close()) }()
			_, parsed := invocation.Application()
			assert.Equal(t, test.text, parsed.Text)
			assert.Equal(t, strings.ContainsRune(test.flags, 'a'), parsed.Alpha)
			assert.Equal(t, strings.ContainsRune(test.flags, 'b'), parsed.Beta)
			assert.Equal(t, strings.ContainsRune(test.flags, 'c'), parsed.Gamma)
			if test.flags != "abc" {
				flag, value := invocation.ShortFlag('s')
				assert.NotNil(t, flag, "empty explicit values must still be provided")
				assert.Equal(t, test.text, value)
			}
		})
	}
}

func TestShortOptionalBoolEqualsDoesNotConsumeFollowingPositions(t *testing.T) {
	type options struct {
		Verbose bool     `flag:"verbose" short:"v" default:"true"`
		Alpha   bool     `flag:"alpha" short:"a"`
		Values  []string `argument:"values"`
	}
	app, err := NewApplication[options]("app").
		WithErrorHandler(func(_ context.Context, _ InvocationLike, err error) error { return err }).Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, app.Close()) }()
	for _, args := range [][]string{{"-v=false", "tail"}, {"-av=false", "tail"}} {
		invocation, err := app.Parse(nil, nil, args...)
		require.NoError(t, err)
		require.NoError(t, invocation.Close())
		_, parsed := invocation.Application()
		assert.False(t, parsed.Verbose, "explicit false replaces the true default")
		assert.Equal(t, []string{"tail"}, parsed.Values)
		_, value := invocation.ShortFlag('v')
		assert.Equal(t, false, value)
	}
	_, err = app.Parse(nil, nil, "-v==false")
	require.ErrorIs(t, err, codec.ErrIllegalArgument, "only the first = is a separator")
}

func TestShortEqualsSupportsUnicodeAndOptionalStringAndRejectsZeroArity(t *testing.T) {
	app, err := NewSimpleApplication("app").WithFlag(
		NewFlag(codec.NewString(), "unicode").WithShort('ü'),
		NewFlag(grammarString{min: 0, max: 1}, "optional").WithShort('o'),
		NewFlag(grammarString{min: 0, max: 0}, "zero").WithShort('z'),
	).WithErrorHandler(func(_ context.Context, _ InvocationLike, err error) error { return err }).Create()
	require.NoError(t, err)
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(nil, nil, "-ü=🙂=value", "-o=")
	require.NoError(t, err)
	require.NoError(t, invocation.Close())
	_, value := invocation.ShortFlag('ü')
	assert.Equal(t, "🙂=value", value)
	flag, value := invocation.ShortFlag('o')
	assert.NotNil(t, flag)
	assert.Equal(t, "", value)
	_, err = app.Parse(nil, nil, "-z=value")
	require.ErrorIs(t, err, codec.ErrNoArgumentExpected)
}

func TestShortHelpEqualsFalseContinuesWhileTrueStops(t *testing.T) {
	app := NewSimpleApplication("app").
		WithCommand(NewSimpleCommand("run")).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	invocation := app.MustParse(nil, nil, "-h=false", "run")
	require.NoError(t, invocation.Close())
	assert.False(t, invocation.HelpRequested())
	assert.Equal(t, []string{"run"}, invocation.NameStack())
	_, value := invocation.ShortFlag('h')
	assert.Equal(t, false, value)
	invocation = app.MustParse(nil, nil, "-h=true", "--unknown")
	require.NoError(t, invocation.Close())
	assert.True(t, invocation.HelpRequested())
	assert.Empty(t, invocation.NameStack())
}
