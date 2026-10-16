package cli

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultCodecFactoryOverrideIsCapturedDuringCreate(t *testing.T) {
	original := codec.Defaults
	t.Cleanup(func() { codec.Defaults = original })
	type options struct {
		Value string `flag:"value"`
	}
	calls := 0
	codec.Defaults = func(ctx context.Context) []codec.Codec[any, any] {
		calls++
		return []codec.Codec[any, any]{grammarString{min: 1, max: 1, prefix: "first:"}.ToRaw(ctx)}
	}
	first := NewApplication[options]("first").MustCreate()
	t.Cleanup(func() { assert.NoError(t, first.Close()) })
	assert.Equal(t, 1, calls)
	codec.Defaults = func(ctx context.Context) []codec.Codec[any, any] {
		calls++
		return []codec.Codec[any, any]{grammarString{min: 1, max: 1, prefix: "second:"}.ToRaw(ctx)}
	}
	second := NewApplication[options]("second").MustCreate()
	t.Cleanup(func() { assert.NoError(t, second.Close()) })
	for _, test := range []struct {
		app      *Application[options]
		expected string
	}{{first, "first:value"}, {second, "second:value"}} {
		invocation, err := test.app.Parse(nil, nil, "--value", "value")
		require.NoError(t, err)
		_, actual := invocation.Application()
		assert.Equal(t, test.expected, actual.Value)
		assert.NoError(t, invocation.Close())
	}
	assert.Equal(t, 2, calls, "parse must not consult Defaults again")
}

func TestDefaultsCanBeDisabledWithoutDisablingExplicitCodecs(t *testing.T) {
	original := codec.Defaults
	t.Cleanup(func() { codec.Defaults = original })
	codec.Defaults = nil
	type options struct {
		Value string `flag:"value"`
	}
	_, err := NewApplication[options]("missing").Create()
	assert.ErrorContains(t, err, "no codec for type string")
	app := NewApplication[options]("explicit").WithStringFlag("value", func(f *FlagRequirement[string, string]) { f.WithField("Value") }).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(nil, nil, "--value", "explicit")
	require.NoError(t, err)
	_, actual := invocation.Application()
	assert.Equal(t, "explicit", actual.Value)
	assert.NoError(t, invocation.Close())
}

func TestCombinedDefaultsPreserveUserPriorityNamesAndLocations(t *testing.T) {
	type options struct {
		Date     time.Time     `flag:"date"`
		Named    time.Time     `flag:"named" type:"date-only"`
		Duration time.Duration `flag:"duration"`
		Number   int64         `flag:"number"`
	}
	zone := time.FixedZone("custom", 2*60*60)
	rootCodec := codec.NewDate().WithFormats("02.01.2006").WithLocation(zone)
	childCodec := codec.NewDate().WithFormats("20060102").WithLocation(zone)
	app := NewApplication[options]("app").WithCodec(rootCodec).
		WithCodec(rootCodec, "date-only").WithCommand(NewSimpleCommand("child").WithCodec(childCodec)).MustCreate()
	defer func() { assert.NoError(t, app.Close()) }()
	invocation, err := app.Parse(nil, nil, "child", "--date", "20261014", "--named", "15.10.2006", "--duration", "42s", "--number", "42")
	require.NoError(t, err)
	defer func() { assert.NoError(t, invocation.Close()) }()
	_, actual := invocation.Application()
	assert.Equal(t, time.Date(2026, 10, 14, 0, 0, 0, 0, zone), actual.Date)
	assert.Equal(t, time.Date(2006, 10, 15, 0, 0, 0, 0, zone), actual.Named)
	assert.Equal(t, 42*time.Second, actual.Duration)
	assert.Equal(t, int64(42), actual.Number)
	assert.NotEqual(t, reflect.TypeOf(actual.Duration), reflect.TypeOf(actual.Number))
}
