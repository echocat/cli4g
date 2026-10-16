package cli

import (
	"context"
	"testing"
	"time"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithNameReplacesAndWithAliasesAppends(t *testing.T) {
	root := NewSimpleApplication("old").
		WithName("app")
	command := NewSimpleCommand("old").
		WithAliases("old-alias").
		WithName("run").
		WithAliases("r")
	flag := NewFlag(codec.NewString(), "old").
		WithAliases("old-alias").
		WithName("value").
		WithAliases("v")
	app := root.WithCommand(command.WithFlag(flag)).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	_, found := app.Command("old")
	assert.False(t, found)
	selected, found := app.Command("r")
	require.True(t, found)
	assert.Equal(t, []string{"run", "r"}, selected.Names())
	_, found = selected.Flag("old")
	assert.False(t, found)
	definition, found := selected.Flag("v")
	require.True(t, found)
	assert.Equal(t, []string{"value", "v"}, definition.Names())
	invocation, err := app.Parse(context.Background(), nil, "r", "--v=input")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, value := invocation.Flag("value")
	assert.Equal(t, "input", value)
}

func TestSetCodecsReplacesCompleteRegistrationList(t *testing.T) {
	type options struct {
		Text string        `flag:"text" type:"text"`
		Wait time.Duration `flag:"wait" type:"duration"`
	}
	registrations := []codec.Registration{
		codec.NewRegistration(codec.NewString(), "text"),
		codec.NewRegistration(codec.NewDuration(), "duration"),
	}
	for _, command := range []bool{false, true} {
		root := NewSimpleApplication("app")
		var args []string
		if command {
			child := NewCommand[options]("run").
				WithCodec(codec.NewInt(), "removed")
			child.SetCodecs(registrations)
			assert.Len(t, child.Codecs, 2)
			root.WithCommand(child)
			args = []string{"run"}
		} else {
			typed := NewApplication[options]("app").
				WithCodec(codec.NewInt(), "removed")
			typed.SetCodecs(registrations)
			assert.Len(t, typed.Codecs, 2)
			app := typed.MustCreate()
			invocation, err := app.Parse(context.Background(), nil, "--text=input", "--wait=250ms")
			require.NoError(t, err)
			_, opts := invocation.Application()
			assert.Equal(t, options{"input", 250 * time.Millisecond}, opts)
			require.NoError(t, invocation.Close())
			require.NoError(t, app.Close())
			continue
		}
		app := root.MustCreate()
		selected, found := FindCommand[options](app, "run")
		require.True(t, found)
		invocation, err := app.Parse(context.Background(), nil, append(args, "--text=input", "--wait=250ms")...)
		require.NoError(t, err)
		opts, found := selected.OptionsFrom(invocation)
		require.True(t, found)
		assert.Equal(t, options{"input", 250 * time.Millisecond}, opts)
		require.NoError(t, invocation.Close())
		require.NoError(t, app.Close())
	}
}
