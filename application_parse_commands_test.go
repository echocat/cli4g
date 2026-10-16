package cli

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/echocat/cli4g/codec"
)

type grammarString struct {
	min      int
	max      int
	prefix   string
	clones   *atomic.Int64
	cloneErr error
}

func (c grammarString) ToRaw(ctx context.Context) codec.Codec[any, any] {
	return codec.Raw(ctx, c)
}

func (grammarString) CanHandleType(_ context.Context, t reflect.Type) (bool, error) {
	return t == reflect.TypeFor[string](), nil
}

func (c grammarString) ExpectMinMax(context.Context, codec.Usage) (int, int) {
	return c.min, c.max
}

func (c grammarString) Parse(ctx context.Context, usage codec.Usage, values ...string) (string, error) {
	if err := codec.EnsureArgumentsMatching(ctx, usage, c, values...); err != nil {
		return "", err
	}
	return c.prefix + strings.Join(values, "|"), nil
}

func (grammarString) Format(_ context.Context, value string) ([]string, error) {
	return []string{value}, nil
}

func (c grammarString) Clone(context.Context) (codec.Codec[string, string], error) {
	if c.clones != nil {
		c.clones.Add(1)
	}
	if c.cloneErr != nil {
		return nil, c.cloneErr
	}
	return c, nil
}

func (c grammarString) CloneDirect() codec.Codec[string, string] {
	return c
}

func (grammarString) CloneValue(ctx context.Context, value string) (string, error) {
	return value, ctx.Err()
}

func TestCommandNamesAfterPositionalValues(t *testing.T) {
	type options struct {
		Verbose bool     `flag:"verbose"`
		Values  []string `argument:"values"`
	}
	app, err := NewApplication[*options]("app").
		WithHandler(func(context.Context, *options) error { return nil }).
		WithCommand(NewSimpleCommand("run")).
		Create()
	require.NoError(t, err)
	for _, test := range []struct {
		name    string
		args    []string
		values  []string
		command bool
	}{
		{"positional", []string{"file", "run"}, []string{"file", "run"}, false},
		{"interspersed flag", []string{"file", "--verbose", "run"}, []string{"file", "run"}, false},
		{"terminator", []string{"--", "run"}, []string{"run"}, false},
		{"command", []string{"run"}, nil, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			invocation, err := app.Parse(context.Background(), nil, test.args...)
			require.NoError(t, err)
			defer func() { assert.NoError(t, invocation.Close()) }()
			_, parsed := invocation.Application()
			assert.Equal(t, test.values, parsed.Values)
			assert.Equal(t, test.command, len(invocation.Stack()) == 1, "selected commands: %v", invocation.NameStack())
		})
	}
}

func TestMultiValueFlagsAndCommandNames(t *testing.T) {
	type options struct {
		Pair string `flag:"pair" short:"p"`
	}
	app := NewApplication[*options]("app").
		WithHandler(func(context.Context, *options) error { return nil }).
		WithCodec(grammarString{min: 2, max: 2}).
		WithCommand(NewSimpleCommand("run")).
		MustCreate()
	for _, test := range []struct {
		args    []string
		command bool
	}{
		{[]string{"--pair", "one", "run"}, false},
		{[]string{"--pair=one", "run", "run"}, true},
		{[]string{"-p", "one", "run"}, false},
		{[]string{"-pone", "run", "run"}, true},
	} {
		invocation, err := app.Parse(context.Background(), nil, test.args...)
		require.NoError(t, err, "args %v", test.args)
		_, parsed := invocation.Application()
		assert.Equal(t, "one|run", parsed.Pair, "args %v", test.args)
		assert.Equal(t, test.command, len(invocation.Stack()) == 1, "args %v; selected commands: %v", test.args, invocation.NameStack())
		require.NoError(t, invocation.Close())
	}
	_, err := app.Parse(context.Background(), nil, "--pair=one", "--")
	require.ErrorIs(t, err, codec.ErrTooFewArguments)
}

func TestVariadicFlagBeforeCommand(t *testing.T) {
	type options struct {
		Items string `flag:"items"`
	}
	app := NewApplication[*options]("app").
		WithCodec(grammarString{min: 0, max: -1}).
		WithCommand(NewSimpleCommand("run")).
		MustCreate()
	invocation := app.MustParse(context.Background(), nil, "--items", "one", "two", "run")
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, parsed := invocation.Application()
	assert.Equal(t, "one|two", parsed.Items)
	assert.Equal(t, []string{"run"}, invocation.NameStack())
}

func TestCommandSelectionDoesNotCloneCodecs(t *testing.T) {
	type options struct {
		Name string `flag:"name" short:"n"`
	}
	rootClones := &atomic.Int64{}
	leafClones := &atomic.Int64{}
	app := NewApplication[*options]("app").
		WithCodec(grammarString{min: 1, max: 1, prefix: "root:", clones: rootClones}).
		WithCommand(NewSimpleCommand("run").
			WithCodec(grammarString{min: 1, max: 1, prefix: "leaf:", clones: leafClones}),
		).
		MustCreate()
	invocation := app.MustParse(context.Background(), nil, "--name", "value", "run")
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, parsed := invocation.Application()
	assert.Equal(t, "leaf:value", parsed.Name)
	assert.Zero(t, rootClones.Load())
	assert.Zero(t, leafClones.Load(), "immutable codecs are shared")
}

func TestParseDoesNotInvokeLegacyCodecCloning(t *testing.T) {
	type options struct {
		Name string `flag:"name"`
	}
	cloneErr := errors.New("clone failed")
	app, err := NewApplication[*options]("app").
		WithCodec(grammarString{min: 1, max: 1, cloneErr: cloneErr}).
		Create()
	require.NoError(t, err)
	invocation, err := app.Parse(context.Background(), nil, "--name", "value")
	require.NoError(t, err)
	require.NoError(t, invocation.Close())
}

func TestLongFlagPrefixKeepsAllValuesAndTerminatorSemantics(t *testing.T) {
	type options struct {
		Name   string   `flag:"name"`
		Values []uint16 `flag:"value"`
		Tail   []string `argument:"tail"`
	}
	app, err := NewApplication[*options]("app").Create()
	require.NoError(t, err)
	args := []string{"--name=server"}
	want := make([]uint16, 160)
	for i := range want {
		args = append(args, "--value="+strconv.Itoa(i))
		want[i] = uint16(i)
	}
	args = append(args, "--", "--value=literal", "-1")
	invocation, err := app.Parse(context.Background(), nil, args...)
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, value := invocation.Application()
	assert.Equal(t, "server", value.Name)
	assert.Equal(t, want, value.Values)
	assert.Equal(t, []string{"--value=literal", "-1"}, value.Tail)
}
