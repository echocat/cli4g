package cli

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/echocat/cli4g/codec"
)

type taggedCalendarOptions struct {
	At     time.Time   `flag:"at" env:"CLI4G_TEST_AT" default:"14 Oct 26 12:30 UTC"`
	Dates  []time.Time `flag:"date"`
	Future []time.Time `argument:"future"`
}

func TestTaggedCalendarDefaultsEnvironmentAndCLI(t *testing.T) {
	t.Run("application", func(t *testing.T) {
		app := NewApplication[taggedCalendarOptions]().WithName("app").WithErrorHandlerEnabled(false).MustCreate()
		defer func() { assert.NoError(t, app.Close()) }()
		testTaggedCalendarSources(t, app, nil, func(invocation *Invocation[taggedCalendarOptions]) taggedCalendarOptions {
			_, values := invocation.Application()
			return values
		})
	})
	t.Run("command", func(t *testing.T) {
		app := NewSimpleApplication().WithName("app").WithErrorHandlerEnabled(false).
			WithCommand(NewCommand[taggedCalendarOptions]().WithName("run")).MustCreate()
		defer func() { assert.NoError(t, app.Close()) }()
		command, found := FindCommand[taggedCalendarOptions](app, "run")
		require.True(t, found)
		testTaggedCalendarSources(t, app, []string{"run"}, func(invocation *Invocation[NoOptions]) taggedCalendarOptions {
			values, found := command.OptionsFrom(invocation)
			require.True(t, found)
			return values
		})
	})
}

func testTaggedCalendarSources[O any](t *testing.T, app *Application[O], prefix []string, valuesFrom func(*Invocation[O]) taggedCalendarOptions) {
	t.Helper()
	day := time.Date(2026, time.October, 14, 0, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name string
		env  string
		args []string
		want taggedCalendarOptions
	}{
		{"default", "", nil, taggedCalendarOptions{At: day.Add(12*time.Hour + 30*time.Minute)}},
		{"environment", "2026-10-14 12:30:45", nil, taggedCalendarOptions{At: day.Add(12*time.Hour + 30*time.Minute + 45*time.Second)}},
		{"CLI", "invalid", []string{"--at=2026-10-14T12:30:45.123456789Z", "--date=14.10.2026", "--date=15 Oct 26", "--", "20261016", "2026-10-17T01:02:03Z"}, taggedCalendarOptions{
			At:     day.Add(12*time.Hour + 30*time.Minute + 45*time.Second + 123456789*time.Nanosecond),
			Dates:  []time.Time{day, day.AddDate(0, 0, 1)},
			Future: []time.Time{day.AddDate(0, 0, 2), day.AddDate(0, 0, 3).Add(time.Hour + 2*time.Minute + 3*time.Second)},
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			opts := &ParseOpts{LookupEnv: func(context.Context, string) (string, bool) { return test.env, test.env != "" }}
			invocation, err := app.Parse(t.Context(), opts, append(append([]string(nil), prefix...), test.args...)...)
			require.NoError(t, err)
			assert.Equal(t, test.want, valuesFrom(invocation))
			require.NoError(t, invocation.Close())
		})
	}
	_, err := app.Parse(t.Context(), nil, append(append([]string(nil), prefix...), "--at=invalid")...)
	assert.ErrorIs(t, err, ErrValidation)
}

func TestCalendarBuildersTypedDefaultsCustomLayoutsAndHelp(t *testing.T) {
	type options struct {
		Day   time.Time
		At    time.Time
		Days  []time.Time
		Times []time.Time
	}
	day := time.Date(2026, time.October, 14, 0, 0, 0, 0, time.UTC)
	precise := time.Date(2026, time.October, 14, 12, 30, 45, 987654321, time.FixedZone("custom", 1234))
	app := NewApplication[options]().WithName("app").WithErrorHandlerEnabled(false).
		WithDateFlag("day", func(f *FlagRequirement[time.Time, time.Time]) {
			f.WithField("Day").WithCodec(codec.NewDate().WithFormats("02/01/2006")).WithDefault(day.Add(12 * time.Hour))
		}).
		WithDateTimeFlag("at", func(f *FlagRequirement[time.Time, time.Time]) {
			f.WithField("At").WithCodec(codec.NewDateTime().WithFormats("02.01.2006 15:04")).WithDefault(precise)
		}).
		WithDateSliceFlag("days", func(f *FlagRequirement[[]time.Time, []time.Time]) {
			f.WithField("Days").WithDefault([]time.Time{day})
		}).
		WithDateTimeSliceArgument("times", func(a *ArgumentRequirement[[]time.Time, []time.Time]) {
			a.WithField("Times").WithDefault([]time.Time{precise})
		}).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	for _, test := range []struct {
		args []string
		want options
	}{
		{nil, options{Day: day, At: precise, Days: []time.Time{day}, Times: []time.Time{precise}}},
		{[]string{"--day=15/10/2026", "--at=15.10.2026 12:30", "--days=15 Oct 26", "--days=20261016", "2026-10-17"}, options{
			Day: day.AddDate(0, 0, 1), At: day.AddDate(0, 0, 1).Add(12*time.Hour + 30*time.Minute),
			Days: []time.Time{day.AddDate(0, 0, 1), day.AddDate(0, 0, 2)}, Times: []time.Time{day.AddDate(0, 0, 3)},
		}},
	} {
		invocation, err := app.Parse(t.Context(), nil, test.args...)
		require.NoError(t, err)
		_, values := invocation.Application()
		assert.Equal(t, test.want, values)
		values.Days[0] = time.Time{}
		require.NoError(t, invocation.Close())
	}
	for name, want := range map[string][]string{"day": {"14/10/2026"}, "at": {"14.10.2026 12:30"}, "days": {"2026-10-14"}} {
		flag, found := app.Flag(name)
		require.True(t, found)
		formatted, present, err := flag.FormatDefault(t.Context())
		require.NoError(t, err)
		assert.True(t, present)
		assert.Equal(t, want, formatted, "help must not expose internal serialization")
	}
}

func TestNamedDateCodecAndInvalidConfiguration(t *testing.T) {
	type options struct {
		Birthday time.Time `flag:"birthday" type:"birthday" default:"Oct 14, 2026"`
	}
	app := NewApplication[options]().WithName("app").WithErrorHandlerEnabled(false).
		WithCodec(codec.NewDate().WithFormats("Jan 2, 2006"), "birthday").MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(t.Context(), nil, "--birthday=Oct 15, 2026")
	require.NoError(t, err)
	_, values := invocation.Application()
	assert.Equal(t, time.Date(2026, time.October, 15, 0, 0, 0, 0, time.UTC), values.Birthday)
	require.NoError(t, invocation.Close())
	_, err = NewSimpleApplication().WithName("app").WithDateFlag("day", func(f *FlagRequirement[time.Time, time.Time]) {
		f.WithCodec(codec.NewDate().WithFormats(""))
	}).Create()
	assert.ErrorIs(t, err, codec.ErrInvalid)
}
