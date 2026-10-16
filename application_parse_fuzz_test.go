package cli

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/echocat/cli4g/codec"
)

// Inline values are opaque input bytes, regardless of whether the flag name is
// ASCII or Unicode. Short syntax must deliver the same value as long syntax.
func FuzzShortFlagInlineStringPreservesBytes(f *testing.F) {
	for _, value := range []string{"", "plain", "=value", "日本語", "\xff", "\xc0\xaf", "\x00\xfe=value"} {
		for prefix := range uint8(6) {
			f.Add(value, prefix)
		}
	}
	app, err := NewSimpleApplication("app").
		WithFlag(
			NewFlag(codec.NewString(), "value").
				WithShort('s', '界'),
			NewFlag(codec.NewBool(), "bool").
				WithShort('b'),
		).
		WithErrorHandler(func(_ context.Context, _ InvocationLike, err error) error { return err }).
		Create()
	require.NoError(f, err)
	f.Cleanup(func() { assert.NoError(f, app.Close()) })
	f.Fuzz(func(t *testing.T, value string, variant uint8) {
		if len(value) > 64*1024 {
			t.Skip()
		}
		variant %= 6
		if variant >= 3 && (value == "" || strings.HasPrefix(value, "=")) {
			t.Skip() // attached syntax needs a nonempty value without a separator
		}
		prefix := []string{"-s=", "-界=", "-bs=", "-s", "-界", "-bs"}[variant]
		for _, token := range []string{"--value=" + value, prefix + value} {
			invocation, err := app.Parse(nil, nil, token)
			require.NoError(t, err)
			defer func() { assert.NoError(t, invocation.Close()) }()
			_, parsed := invocation.Flag("value")
			assert.Equal(t, value, parsed, "inline input must be preserved byte-for-byte")
		}
	})
}

// Generate a known grammar and independently assert options, arguments and the
// selected canonical command, rather than comparing two parser accessors.
func FuzzStructuredInvocationValues(f *testing.F) {
	f.Add("hello", "World", int32(7), true)
	f.Add("\xff=opaque", "--help", int32(-3), false)
	type rootOptions struct {
		Text   string `flag:"text" short:"t"`
		Number int32  `flag:"number"`
	}
	type commandOptions struct {
		Name string `argument:"name" required:"true"`
	}
	app := NewApplication[rootOptions]("app").
		WithErrorHandlerEnabled(false).
		WithCommand(NewCommand[commandOptions]("run").
			WithAliases("r"),
		).
		MustCreate()
	f.Cleanup(func() { assert.NoError(f, app.Close()) })
	command, found := FindCommand[commandOptions](app, "run")
	require.True(f, found)
	f.Fuzz(func(t *testing.T, text, name string, number int32, alias bool) {
		if len(text)+len(name) > 64*1024 {
			t.Skip()
		}
		selected := "run"
		if alias {
			selected = "r"
		}
		invocation, err := app.Parse(context.Background(), nil, "--text="+text,
			"--number="+strconv.FormatInt(int64(number), 10), selected, "--", name)
		require.NoError(t, err)
		defer func() { assert.NoError(t, invocation.Close()) }()
		_, root := invocation.Application()
		assert.Equal(t, rootOptions{Text: text, Number: number}, root)
		assert.Equal(t, []string{"run"}, invocation.NameStack())
		options, found := command.OptionsFrom(invocation)
		require.True(t, found)
		assert.Equal(t, name, options.Name)
		_, argument := invocation.Argument(0)
		assert.Equal(t, name, argument)
	})
}

// Exercise scanner/decoder/error/help paths with arbitrary token sequences on
// one immutable application. A failed parse must never return a live invocation.
func FuzzApplicationParseTokens(f *testing.F) {
	for _, input := range []string{
		"", "--help\x00--unknown", "--enabled=false\x00position", "-ee", "--\x00--help",
		"--count=1\x00--count=2", "--map=a=b\x00--map=c=d", "-s=\xff", "--text\x00--help",
	} {
		f.Add(input)
	}
	type options struct {
		Text      string              `flag:"text" short:"s"`
		Enabled   flagReceiverBool    `flag:"enabled" short:"e"`
		Count     flagReceiverCounter `flag:"count"`
		Map       flagReceiverMap     `flag:"map"`
		Arguments []string            `argument:"arguments"`
	}
	app := NewApplication[*options]("app").
		WithCommand(NewSimpleCommand("child")).
		WithErrorHandler(func(_ context.Context, _ InvocationLike, err error) error { return err }).
		MustCreate()
	f.Cleanup(func() { assert.NoError(f, app.Close()) })
	f.Fuzz(func(t *testing.T, input string) {
		if len(input) > 64*1024 {
			t.Skip()
		}
		tokens := strings.Split(input, "\x00")
		if len(tokens) > 128 {
			t.Skip()
		}
		invocation, err := app.Parse(nil, nil, tokens...)
		if err != nil {
			assert.Nil(t, invocation)
			return
		}
		require.NotNil(t, invocation)
		require.NoError(t, invocation.Close())
		seen := make(map[*Flag[any, any]]bool)
		for definition, value := range invocation.Flags() {
			assert.False(t, seen[definition], "one aggregate entry per flag definition")
			seen[definition] = true
			_, indexed := invocation.Flag(definition.Name())
			assert.Equal(t, value, indexed, "iteration and indexed access agree after Close")
		}
	})
}
