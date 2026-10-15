package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type directMethodOptions struct {
	events *[]string
	ready  bool
}

func (opts *directMethodOptions) Validate(context.Context) error {
	*opts.events = append(*opts.events, "method validate")
	return nil
}
func (opts *directMethodOptions) Initialize(context.Context) error {
	opts.ready = true
	*opts.events = append(*opts.events, "method initialize")
	return nil
}
func (opts *directMethodOptions) Close() error {
	*opts.events = append(*opts.events, "method close")
	return nil
}

func TestDirectLifecycleCallsOptionsMethodsAndCallbacks(t *testing.T) {
	for _, command := range []bool{false, true} {
		var events []string
		validate := func(context.Context, *directMethodOptions) error {
			events = append(events, "callback validate")
			return nil
		}
		initialize := func(_ context.Context, opts *directMethodOptions) error {
			assert.True(t, opts.ready)
			events = append(events, "callback initialize")
			return nil
		}
		closeOptions := func(*directMethodOptions) error { events = append(events, "callback close"); return nil }
		root := NewApplication[*directMethodOptions]().WithName("app").WithValidateHandler(validate).
			WithInitializeHandler(initialize).WithCloseHandler(closeOptions)
		root.WithCommand(NewCommand[*directMethodOptions]().WithName("run").WithValidateHandler(validate).
			WithInitializeHandler(initialize).WithCloseHandler(closeOptions))
		app := root.MustCreate()
		opts := &directMethodOptions{events: &events}
		if command {
			selected, found := FindCommand[*directMethodOptions](app, "run")
			require.True(t, found)
			require.NoError(t, selected.Validate(context.Background(), opts))
			require.NoError(t, selected.Initialize(context.Background(), opts))
			require.NoError(t, selected.Close(opts))
		} else {
			require.NoError(t, app.Validate(context.Background(), opts))
			require.NoError(t, app.Initialize(context.Background(), opts))
			require.NoError(t, app.CloseOptions(opts))
		}
		assert.Equal(t, []string{"method validate", "callback validate", "method initialize", "callback initialize", "callback close", "method close"}, events)
		assert.True(t, opts.ready)
		require.NoError(t, app.Close())
	}
}

func TestCloseCallbackReleasesPartiallyCreatedOrInitializedOptions(t *testing.T) {
	type options struct{ File *os.File }
	path := filepath.Join(t.TempDir(), "input")
	require.NoError(t, os.WriteFile(path, []byte("contents"), 0600))
	for _, stage := range []string{"factory", "validate", "initialize"} {
		t.Run(stage, func(t *testing.T) {
			failure := errors.New("failed " + stage)
			var file *os.File
			closed := 0
			root := NewApplication[*options]().WithName("app").WithErrorHandlerEnabled(false).
				WithCreateOptionsHandler(func(context.Context) (*options, error) {
					var err error
					file, err = os.Open(path)
					if err != nil {
						return nil, err
					}
					if stage == "factory" {
						return &options{file}, failure
					}
					return &options{file}, nil
				}).WithCloseHandler(func(opts *options) error { closed++; return opts.File.Close() })
			if stage == "validate" {
				root.WithValidateHandler(func(context.Context, *options) error { return failure })
			} else if stage == "initialize" {
				root.WithInitializeHandler(func(context.Context, *options) error { return failure })
			}
			app := root.MustCreate()
			defer func() { assert.NoError(t, app.Close()) }()
			invocation, err := app.Parse(context.Background(), nil)
			require.ErrorIs(t, err, failure)
			assert.Nil(t, invocation)
			assert.Equal(t, 1, closed)
			_, err = file.Read(make([]byte, 1))
			assert.ErrorIs(t, err, os.ErrClosed)
		})
	}
}
