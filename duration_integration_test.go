package cli

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type taggedDurationOptions struct {
	Sleep   time.Duration   `flag:"sleep" env:"CLI4G_TEST_SLEEP" default:"1m30s"`
	Delays  []time.Duration `flag:"delay"`
	Periods []time.Duration `argument:"periods"`
}

func TestTaggedDurationsUseStandardCodecsAcrossInputSources(t *testing.T) {
	t.Run("application", func(t *testing.T) {
		app := NewApplication[taggedDurationOptions]().WithName("app").WithErrorHandlerEnabled(false).MustCreate()
		defer func() { assert.NoError(t, app.Close()) }()
		testTaggedDurationSources(t, app, app, nil, func(invocation *Invocation[taggedDurationOptions]) taggedDurationOptions {
			_, parsed := invocation.Application()
			return parsed
		})
	})
	t.Run("command", func(t *testing.T) {
		app := NewSimpleApplication().WithName("app").WithErrorHandlerEnabled(false).
			WithCommand(NewCommand[taggedDurationOptions]().WithName("run")).MustCreate()
		defer func() { assert.NoError(t, app.Close()) }()
		command, found := FindCommand[taggedDurationOptions](app, "run")
		require.True(t, found)
		testTaggedDurationSources(t, app, command, []string{"run"}, func(invocation *Invocation[NoOptions]) taggedDurationOptions {
			parsed, found := command.OptionsFrom(invocation)
			require.True(t, found)
			return parsed
		})
	})
}

func testTaggedDurationSources[O any](t *testing.T, app *Application[O], container Container, prefix []string, optionsFrom func(*Invocation[O]) taggedDurationOptions) {
	t.Helper()
	for _, test := range []struct {
		name string
		env  string
		args []string
		want taggedDurationOptions
	}{
		{"default", "", nil, taggedDurationOptions{Sleep: 90 * time.Second}},
		{"environment", "250ms", nil, taggedDurationOptions{Sleep: 250 * time.Millisecond}},
		{"CLI", "250ms", []string{"--sleep=-2s", "--delay=100ms", "--delay=1s", "--", "5m", "-750ms"}, taggedDurationOptions{
			Sleep: -2 * time.Second, Delays: []time.Duration{100 * time.Millisecond, time.Second},
			Periods: []time.Duration{5 * time.Minute, -750 * time.Millisecond},
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			parseOpts := &ParseOpts{LookupEnv: func(context.Context, string) (string, bool) {
				return test.env, test.env != ""
			}}
			invocation, err := app.Parse(context.Background(), parseOpts, append(append([]string(nil), prefix...), test.args...)...)
			require.NoError(t, err)
			defer func() { assert.NoError(t, invocation.Close()) }()
			assert.Equal(t, test.want, optionsFrom(invocation))
			definition, found := container.Flag("sleep")
			require.True(t, found)
			formatted, hasDefault, err := definition.FormatDefault(context.Background())
			require.NoError(t, err)
			assert.True(t, hasDefault)
			assert.Equal(t, []string{"1m30s"}, formatted)
		})
	}
	_, err := app.Parse(context.Background(), nil, append(append([]string(nil), prefix...), "--sleep=invalid")...)
	assert.ErrorIs(t, err, ErrValidation)
}

func TestDurationBuilderTypedDefaultsAndFieldBindings(t *testing.T) {
	type options struct {
		Sleep  time.Duration
		Delays []time.Duration
	}
	app := NewApplication[options]().WithName("app").WithErrorHandlerEnabled(false).
		WithDurationFlag("sleep", func(f *FlagRequirement[time.Duration, time.Duration]) {
			f.WithField("Sleep").WithDefault(1500 * time.Millisecond)
		}).
		WithDurationSliceArgument("delays", func(a *ArgumentRequirement[[]time.Duration, []time.Duration]) {
			a.WithField("Delays").WithDefault([]time.Duration{time.Second, 2 * time.Second})
		}).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	for _, test := range []struct {
		args []string
		want options
	}{
		{nil, options{Sleep: 1500 * time.Millisecond, Delays: []time.Duration{time.Second, 2 * time.Second}}},
		{[]string{"--sleep=250ms", "3s"}, options{Sleep: 250 * time.Millisecond, Delays: []time.Duration{3 * time.Second}}},
	} {
		invocation, err := app.Parse(context.Background(), nil, test.args...)
		require.NoError(t, err)
		_, parsed := invocation.Application()
		assert.Equal(t, test.want, parsed)
		require.NoError(t, invocation.Close())
	}
}
