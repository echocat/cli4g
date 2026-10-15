package codec

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNumericCodecs(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		codec   any
		input   string
		want    any
		invalid string
	}{
		{"int", NewInt(), "-42", int(-42), "invalid"},
		{"int8", NewInt8(), "-42", int8(-42), "128"},
		{"int16", NewInt16(), "-42", int16(-42), "32768"},
		{"int32", NewInt32(), "-42", int32(-42), "2147483648"},
		{"int64", NewInt64(), "-42", int64(-42), "9223372036854775808"},
		{"uint", NewUint(), "42", uint(42), "-1"},
		{"uint8", NewUint8(), "42", uint8(42), "256"},
		{"uint16", NewUint16(), "42", uint16(42), "65536"},
		{"uint32", NewUint32(), "42", uint32(42), "4294967296"},
		{"uint64", NewUint64(), "42", uint64(42), "18446744073709551616"},
		{"uintptr", NewUintptr(), "42", uintptr(42), "-1"},
		{"float32", NewFloat32(), "1.25", float32(1.25), "invalid"},
		{"float64", NewFloat64(), "1.25", float64(1.25), "invalid"},
		{"complex64", NewComplex64(), "1+2i", complex64(1 + 2i), "invalid"},
		{"complex128", NewComplex128(), "1+2i", complex128(1 + 2i), "invalid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			codec, err := Snapshot(ctx, tc.codec)
			require.NoError(t, err)
			matches, err := codec.CanHandleType(ctx, reflect.TypeOf(tc.want))
			require.NoError(t, err)
			assert.True(t, matches)
			matches, err = codec.CanHandleType(ctx, reflect.TypeFor[string]())
			require.NoError(t, err)
			assert.False(t, matches)
			//goland:noinspection GoShadowedVar,GoReservedWordUsedAsName
			min, max := codec.ExpectMinMax(ctx, UsageArgument)
			assert.Equal(t, 1, min)
			assert.Equal(t, 1, max)
			got, err := codec.ParseRequirement(ctx, UsageArgument, tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got, "parsed %q", tc.input)
			formatted, err := codec.FormatRequirement(ctx, got)
			require.NoError(t, err)
			require.Len(t, formatted, 1)
			again, err := codec.ParseRequirement(ctx, UsageArgument, formatted...)
			require.NoError(t, err)
			assert.Equal(t, got, again, "round trip")
			_, err = codec.ParseRequirement(ctx, UsageArgument, tc.invalid)
			require.ErrorIs(t, err, ErrIllegalArgument)
			_, err = codec.ParseRequirement(ctx, UsageArgument)
			require.ErrorIs(t, err, ErrArgumentRequired)
		})
	}
}
