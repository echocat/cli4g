package codec

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDateDefaultLayouts(t *testing.T) {
	source := NewDate()
	want := time.Date(2026, time.October, 14, 0, 0, 0, 0, time.UTC)
	for _, input := range []string{"2026-10-14", "14 Oct 26", "14.10.2026", "20261014"} {
		t.Run(input, func(t *testing.T) {
			value, err := source.ParseRequirement(t.Context(), UsageFlag, input)
			require.NoError(t, err)
			assert.Equal(t, want, value)
			testCalendarRoundtrip(t, source, value)
			display, err := source.DisplayRequirement(t.Context(), value)
			require.NoError(t, err)
			assert.Equal(t, []string{"2026-10-14"}, display)
		})
	}
	for _, input := range []string{"2026-10-14T00:00:00Z", "2026-10-14 12:30:00", "14 Oct 26 12:30 UTC", "03/04/2026", "2025-02-29", "2100-02-29", "2026-13-01", "", "not-a-date"} {
		_, err := source.ParseRequirement(t.Context(), UsageArgument, input)
		assert.ErrorIs(t, err, ErrIllegalArgument, input)
	}
	value, err := source.ParseRequirement(t.Context(), UsageArgument, "2024-02-29")
	require.NoError(t, err)
	assert.Equal(t, 29, value.Day())
}

func TestDateTimeDefaultLayouts(t *testing.T) {
	base := time.Date(2026, time.October, 14, 12, 30, 45, 0, time.UTC)
	for _, test := range []struct {
		input string
		want  time.Time
	}{
		{"2026-10-14T12:30:45Z", base},
		{"2026-10-14T12:30:45.123456789Z", base.Add(123456789 * time.Nanosecond)},
		{"2026-10-14T14:30:45+02:00", base.In(time.FixedZone("", 2*60*60))},
		{"2026-10-14 12:30:45", base},
		{"14 Oct 26 12:30 UTC", base.Add(-45 * time.Second)},
		{"14 Oct 26 12:30 GMT", base.Add(-45 * time.Second).In(time.FixedZone("GMT", 0))},
		{"14 Oct 26 14:30 +0200", base.Add(-45 * time.Second).In(time.FixedZone("", 2*60*60))},
		{"2026-10-14", time.Date(2026, time.October, 14, 0, 0, 0, 0, time.UTC)},
		{"14 Oct 26", time.Date(2026, time.October, 14, 0, 0, 0, 0, time.UTC)},
		{"14.10.2026", time.Date(2026, time.October, 14, 0, 0, 0, 0, time.UTC)},
		{"20261014", time.Date(2026, time.October, 14, 0, 0, 0, 0, time.UTC)},
	} {
		t.Run(test.input, func(t *testing.T) {
			source := NewDateTime()
			value, err := source.ParseRequirement(t.Context(), UsageFlag, test.input)
			require.NoError(t, err)
			assert.True(t, test.want.Equal(value), "got %s, want %s", value, test.want)
			wantZone, wantOffset := test.want.Zone()
			zone, offset := value.Zone()
			assert.Equal(t, wantZone, zone)
			assert.Equal(t, wantOffset, offset)
			testCalendarRoundtrip(t, source, value)
		})
	}
	for _, input := range []string{"2026-10-14T25:00:00Z", "2026-10-14T12:30:60Z", "2025-02-29T12:30:00Z", "14 Oct 26 12:30 XYZ", "14 Oct 26 12:30 CET", "", "not-a-time"} {
		_, err := NewDateTime().ParseRequirement(t.Context(), UsageFlag, input)
		assert.ErrorIs(t, err, ErrIllegalArgument, input)
	}
}

func testCalendarRoundtrip(t *testing.T, source Codec[time.Time, time.Time], value time.Time) {
	t.Helper()
	encoded, err := source.FormatRequirement(t.Context(), value)
	require.NoError(t, err)
	parsed, err := source.ParseRequirement(t.Context(), UsageDefault, encoded...)
	require.NoError(t, err)
	cloned, err := CloneRequirement(t.Context(), source, value)
	require.NoError(t, err)
	assert.Equal(t, parsed, cloned, "fast cloning must match serialization")
	assert.True(t, value.Equal(parsed))
	zone, offset := value.Zone()
	parsedZone, parsedOffset := parsed.Zone()
	assert.Equal(t, zone, parsedZone)
	assert.Equal(t, offset, parsedOffset)
	resolved, err := source.Resolve(t.Context(), nil, parsed)
	require.NoError(t, err)
	assert.Equal(t, parsed, resolved)
	_, err = source.ParseRequirement(t.Context(), UsageFlag, encoded...)
	assert.ErrorIs(t, err, ErrIllegalArgument, "internal serialization is not a CLI format")
	_, err = Snapshot(t.Context(), source)
	require.NoError(t, err, "time.Time requirements must remain resource-free")
}

func TestCalendarCustomFormatsAndLocations(t *testing.T) {
	additional := []string{"2006/01/02"}
	source := NewDate().WithFormats("02/01/2006", additional...)
	additional[0] = "broken"
	value, err := source.ParseRequirement(t.Context(), UsageFlag, "02/03/2026")
	require.NoError(t, err)
	assert.Equal(t, time.March, value.Month())
	assert.Equal(t, 2, value.Day())
	value, err = source.ParseRequirement(t.Context(), UsageFlag, "2026/10/14")
	require.NoError(t, err, "configured layouts must own their slices")
	assert.Equal(t, 14, value.Day())
	_, err = source.ParseRequirement(t.Context(), UsageFlag, "2026-10-14")
	assert.ErrorIs(t, err, ErrIllegalArgument, "custom layouts replace defaults")
	other := source.WithFormats("01/02/2006")
	value, err = other.ParseRequirement(t.Context(), UsageFlag, "02/03/2026")
	require.NoError(t, err)
	assert.Equal(t, time.February, value.Month())
	assert.Equal(t, 3, value.Day())
	value, err = source.WithFormats("02/01/2006", "01/02/2006").ParseRequirement(t.Context(), UsageFlag, "02/03/2026")
	require.NoError(t, err)
	assert.Equal(t, time.March, value.Month(), "the first matching layout wins")

	location, err := time.LoadLocation("Europe/Berlin")
	require.NoError(t, err)
	dateTime := NewDateTime().WithLocation(location)
	value, err = dateTime.ParseRequirement(t.Context(), UsageFlag, "14 Jan 26 12:30 CET")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, time.January, 14, 12, 30, 0, 0, location), value)
	testCalendarRoundtrip(t, dateTime, value)
	assert.Same(t, location, value.Location())
	cloned, err := dateTime.CloneRequirement(t.Context(), value)
	require.NoError(t, err)
	assert.Same(t, location, cloned.Location())
	assert.Equal(t, time.Date(2026, time.July, 14, 12, 30, 0, 0, location), cloned.AddDate(0, 6, 0), "cloning must retain DST rules")
	value, err = dateTime.ParseRequirement(t.Context(), UsageFlag, "2026-07-14 12:30:45")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, time.July, 14, 12, 30, 45, 0, location), value)
	value, err = NewDate().WithLocation(location).ParseRequirement(t.Context(), UsageFlag, "2026-07-14")
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, time.July, 14, 0, 0, 0, 0, location), value)
	testCalendarRoundtrip(t, NewDate().WithLocation(location), value)
	value, err = NewDateTime().WithFormats("02 Jan 06 15:04 MST -0700").ParseRequirement(t.Context(), UsageFlag, "14 Oct 26 12:30 XYZ +0000")
	require.NoError(t, err, "an explicit numeric offset resolves an otherwise unknown abbreviation")
	zone, offset := value.Zone()
	assert.Equal(t, "XYZ", zone)
	assert.Zero(t, offset)
}

func TestCalendarDefaultsPreservePrecisionAndDisplay(t *testing.T) {
	value := time.Date(2026, time.October, 14, 12, 30, 45, 123456789, time.FixedZone("odd\xff", 1234))
	source := NewDateTime().WithFormats("02.01.2006 15:04")
	testCalendarRoundtrip(t, source, value)
	display, err := source.DisplayRequirement(t.Context(), value)
	require.NoError(t, err)
	assert.Equal(t, []string{"14.10.2026 12:30"}, display)
	display, err = NewDateTime().DisplayRequirement(t.Context(), value.UTC())
	require.NoError(t, err)
	assert.Equal(t, []string{value.UTC().Format(time.RFC3339Nano)}, display)
	for _, value := range []time.Time{{}, time.Date(10000, time.January, 1, 0, 0, 0, 0, time.UTC), time.Now()} {
		testCalendarRoundtrip(t, NewDateTime(), value)
	}
	cloned, err := NewDate().CloneRequirement(t.Context(), value)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2026, time.October, 14, 0, 0, 0, 0, value.Location()), cloned)
	location, err := time.LoadLocation("Europe/Berlin")
	require.NoError(t, err)
	_, err = NewDateTime().FormatRequirement(t.Context(), value.In(location))
	assert.ErrorIs(t, err, ErrInvalid, "named zone rules require a configured location")
	fixedNamedLocation, err := time.LoadLocation("Etc/GMT-2")
	require.NoError(t, err)
	_, err = NewDateTime().FormatRequirement(t.Context(), value.In(fixedNamedLocation))
	assert.ErrorIs(t, err, ErrInvalid, "constant IANA locations also retain their configured name")
	testCalendarRoundtrip(t, NewDateTime().WithLocation(fixedNamedLocation), value.In(fixedNamedLocation))
}

func TestCalendarValidationAndCancellation(t *testing.T) {
	for _, source := range []Codec[time.Time, time.Time]{NewDate(), NewDateTime()} {
		matched, err := source.CanHandleType(t.Context(), reflect.TypeFor[time.Time]())
		require.NoError(t, err)
		assert.True(t, matched)
		matched, err = source.CanHandleType(t.Context(), reflect.TypeFor[time.Duration]())
		require.NoError(t, err)
		assert.False(t, matched)
		for _, usage := range []Usage{UsageFlag, UsageArgument, UsageDefault} {
			_, err := source.ParseRequirement(t.Context(), usage)
			assert.ErrorIs(t, err, ErrArgumentRequired)
			_, err = source.ParseRequirement(t.Context(), usage, "2026-10-14", "2026-10-15")
			assert.ErrorIs(t, err, ErrTooManyArguments)
		}
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, err = source.CanHandleType(ctx, reflect.TypeFor[time.Time]())
		assert.ErrorIs(t, err, context.Canceled)
		_, err = source.ParseRequirement(ctx, UsageFlag, "2026-10-14")
		assert.ErrorIs(t, err, context.Canceled)
		_, err = source.FormatRequirement(ctx, time.Time{})
		assert.ErrorIs(t, err, context.Canceled)
		_, err = source.DisplayRequirement(ctx, time.Time{})
		assert.ErrorIs(t, err, context.Canceled)
		_, err = CloneRequirement(ctx, source, time.Time{})
		assert.ErrorIs(t, err, context.Canceled)
		_, err = source.Resolve(ctx, nil, time.Time{})
		assert.ErrorIs(t, err, context.Canceled)
		assert.Nil(t, source.ToRaw(ctx))
	}
	for _, source := range []Codec[time.Time, time.Time]{NewDate().WithFormats(""), NewDateTime().WithFormats(time.RFC3339, "")} {
		_, err := source.ParseRequirement(t.Context(), UsageFlag, "2026-10-14")
		assert.ErrorIs(t, err, ErrInvalid)
		assert.NotErrorIs(t, err, ErrValidation)
		_, err = Snapshot(t.Context(), source)
		assert.ErrorIs(t, err, ErrInvalid)
	}
	for _, input := range []string{"datetime+json:{}", `datetime+json:{"zone":"utc","time":"bad"}`, `datetime+json:{"extra":true}`, "datetime+json:{} {}"} {
		_, err := NewDateTime().ParseRequirement(t.Context(), UsageDefault, input)
		assert.ErrorIs(t, err, ErrIllegalArgument, input)
	}
}
