package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVariadicFlagCommandBoundaryTracksSelectionState(t *testing.T) {
	app, err := NewSimpleApplication("app").
		WithHandler(func(context.Context, NoOptions) error { return nil }).
		WithHelp(false).
		WithErrorHandlerEnabled(false).
		WithFlag(NewFlag(grammarString{min: 1, max: -1}, "items").
			WithShort('i'),
		).
		WithStringSliceArgument("positions").
		WithCommand(NewSimpleCommand("run").
			WithAliases("r"),
		).
		Create()
	require.NoError(t, err)
	defer app.Close()
	for _, test := range []struct {
		name      string
		args      []string
		items     string
		positions []string
		commands  int
	}{
		{"long-after-position", []string{"pos", "--items", "one", "run", "two"}, "one|run|two", []string{"pos"}, 0},
		{"short-after-position", []string{"pos", "-i", "one", "run", "two"}, "one|run|two", []string{"pos"}, 0},
		{"alias-after-position", []string{"pos", "--items", "one", "r", "two"}, "one|r|two", []string{"pos"}, 0},
		{"long-before-position", []string{"--items", "one", "run"}, "one", nil, 1},
		{"short-before-position", []string{"-i", "one", "r"}, "one", nil, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			invocation, err := app.Parse(nil, nil, test.args...)
			require.NoError(t, err)
			defer invocation.Close()
			_, items := invocation.Flag("items")
			assert.Equal(t, test.items, items)
			_, positions := invocation.Argument(0)
			if test.positions == nil {
				assert.Nil(t, positions)
			} else {
				assert.Equal(t, test.positions, positions)
			}
			assert.Len(t, invocation.NameStack(), test.commands)
		})
	}
}
