package cli

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTaggedWholeEmbeddedBindingsAreOpaque(t *testing.T) {
	for _, kind := range []string{"flag", "argument", "arg", "recursive"} {
		t.Run(kind, func(t *testing.T) {
			root := NewApplication[*struct {
				*ExplicitEmbeddedValue `flag:"whole"`
			}]().WithName("app").WithHelp(false).WithErrorHandlerEnabled(false)
			if kind == "flag" {
				app, err := root.Create()
				require.NoError(t, err)
				defer func() { require.NoError(t, app.Close()) }()
				_, found := app.Flag("internal")
				assert.False(t, found)
				invocation, err := app.Parse(nil, nil, "--whole=cli")
				require.NoError(t, err)
				defer func() { require.NoError(t, invocation.Close()) }()
				_, options := invocation.Application()
				assert.Equal(t, "cli", options.Text)
				return
			}
			if kind == "recursive" {
				app, err := NewApplication[*struct {
					*RecursiveBoundValue `flag:"whole"`
				}]().WithName("app").WithHelp(false).WithErrorHandlerEnabled(false).Create()
				require.NoError(t, err)
				defer func() { require.NoError(t, app.Close()) }()
				invocation, err := app.Parse(nil, nil, "--whole=cli")
				require.NoError(t, err)
				defer func() { require.NoError(t, invocation.Close()) }()
				_, options := invocation.Application()
				assert.Equal(t, "cli", options.Text)
				return
			}
			if kind == "arg" {
				app, err := NewApplication[*struct {
					*ExplicitEmbeddedValue `arg:"whole"`
				}]().WithName("app").WithHelp(false).WithErrorHandlerEnabled(false).Create()
				require.NoError(t, err)
				defer func() { require.NoError(t, app.Close()) }()
				_, found := app.Flag("internal")
				assert.False(t, found)
				invocation, err := app.Parse(nil, nil, "cli")
				require.NoError(t, err)
				defer func() { require.NoError(t, invocation.Close()) }()
				_, options := invocation.Application()
				assert.Equal(t, "cli", options.Text)
				return
			}
			app, err := NewApplication[*struct {
				*ExplicitEmbeddedValue `argument:"whole"`
			}]().WithName("app").WithHelp(false).WithErrorHandlerEnabled(false).Create()
			require.NoError(t, err)
			defer func() { require.NoError(t, app.Close()) }()
			_, found := app.Flag("internal")
			assert.False(t, found)
			invocation, err := app.Parse(nil, nil, "cli")
			require.NoError(t, err)
			defer func() { require.NoError(t, invocation.Close()) }()
			_, options := invocation.Application()
			assert.Equal(t, "cli", options.Text)
		})
	}
}
