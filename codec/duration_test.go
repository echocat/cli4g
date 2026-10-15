package codec

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDurationCodecParsingAndRoundtrips(t *testing.T) {
	ctx := context.Background()
	source := NewDuration()
	for _, test := range []struct {
		input string
		want  time.Duration
	}{
		{"0", 0},
		{"1ns", time.Nanosecond},
		{"1us", time.Microsecond},
		{"1µs", time.Microsecond},
		{"1μs", time.Microsecond},
		{"250ms", 250 * time.Millisecond},
		{"1.5s", 1500 * time.Millisecond},
		{"2m30s", 150 * time.Second},
		{"-1.5s", -1500 * time.Millisecond},
		{"+2h", 2 * time.Hour},
		{"9223372036854775807ns", time.Duration(1<<63 - 1)},
		{"-9223372036854775808ns", time.Duration(-1 << 63)},
	} {
		t.Run(test.input, func(t *testing.T) {
			value, err := source.ParseRequirement(ctx, UsageFlag, test.input)
			require.NoError(t, err)
			assert.Equal(t, test.want, value)
			formatted, err := source.FormatRequirement(ctx, value)
			require.NoError(t, err)
			again, err := source.ParseRequirement(ctx, UsageDefault, formatted...)
			require.NoError(t, err)
			assert.Equal(t, value, again)
			display, err := source.DisplayRequirement(ctx, value)
			require.NoError(t, err)
			assert.Equal(t, formatted, display)
			cloned, err := CloneRequirement(ctx, source, value)
			require.NoError(t, err)
			assert.Equal(t, value, cloned)
			resolved, err := source.Resolve(ctx, nil, cloned)
			require.NoError(t, err)
			assert.Equal(t, value, resolved)
		})
	}
}

func TestDurationCodecRejectsInvalidSyntaxOverflowAndArity(t *testing.T) {
	source := NewDuration()
	for _, usage := range []Usage{UsageFlag, UsageArgument, UsageDefault} {
		for _, input := range []string{"", "invalid", "5", "1d", "9223372036854775808ns", "-9223372036854775809ns"} {
			_, err := source.ParseRequirement(context.Background(), usage, input)
			assert.ErrorIs(t, err, ErrIllegalArgument, "input %q", input)
		}
		_, err := source.ParseRequirement(context.Background(), usage)
		assert.ErrorIs(t, err, ErrArgumentRequired)
		_, err = source.ParseRequirement(context.Background(), usage, "1s", "2s")
		assert.ErrorIs(t, err, ErrTooManyArguments)
	}
	matched, err := source.CanHandleType(context.Background(), reflect.TypeFor[time.Duration]())
	require.NoError(t, err)
	assert.True(t, matched)
	matched, err = source.CanHandleType(context.Background(), reflect.TypeFor[int64]())
	require.NoError(t, err)
	assert.False(t, matched, "durations must not replace the int64 codec")
}

func TestDurationCodecPreservesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	source := NewDuration()
	_, err := source.CanHandleType(ctx, reflect.TypeFor[time.Duration]())
	assert.ErrorIs(t, err, context.Canceled)
	_, err = source.ParseRequirement(ctx, UsageFlag, "1s")
	assert.ErrorIs(t, err, context.Canceled)
	_, err = source.FormatRequirement(ctx, time.Second)
	assert.ErrorIs(t, err, context.Canceled)
	_, err = source.DisplayRequirement(ctx, time.Second)
	assert.ErrorIs(t, err, context.Canceled)
	_, err = CloneRequirement(ctx, source, time.Second)
	assert.ErrorIs(t, err, context.Canceled)
	_, err = source.Resolve(ctx, nil, time.Second)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, source.ToRaw(ctx))
}
