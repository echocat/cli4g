package codec_test

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapKeyTypes(t *testing.T) {
	type label string
	type enabled bool
	type identifier uint32
	for _, test := range []struct {
		name string
		run  func(*testing.T)
	}{
		{"string", mapKeyCase("name", "other", "name", "", "", "")},
		{"bool", mapKeyCase("true", "false", true, "1", "invalid", "")},
		{"int", mapKeyCase("-1", "2", int(-1), "-0x1", "invalid", "")},
		{"int8", mapKeyCase("-128", "127", int8(-128), "-0x80", "128", "-129")},
		{"int16", mapKeyCase("-32768", "32767", int16(-32768), "-0x8000", "32768", "-32769")},
		{"int32", mapKeyCase("-2147483648", "2147483647", int32(-2147483648), "-0x80000000", "2147483648", "-2147483649")},
		{"int64", mapKeyCase("-9223372036854775808", "9223372036854775807", int64(-9223372036854775808), "-0x8000000000000000", "9223372036854775808", "-9223372036854775809")},
		{"uint", mapKeyCase("1", "2", uint(1), "0x1", "-1", "")},
		{"uint8", mapKeyCase("255", "0", uint8(255), "0xff", "256", "-1")},
		{"uint16", mapKeyCase("65535", "0", uint16(65535), "0xffff", "65536", "-1")},
		{"uint32", mapKeyCase("4294967295", "0", uint32(4294967295), "0xffffffff", "4294967296", "-1")},
		{"uint64", mapKeyCase("18446744073709551615", "0", uint64(18446744073709551615), "0xffffffffffffffff", "18446744073709551616", "-1")},
		{"uintptr", mapKeyCase("1", "2", uintptr(1), "0x1", "-1", "")},
		{"named-string", mapKeyCase("name", "other", label("name"), "", "", "")},
		{"named-bool", mapKeyCase("true", "false", enabled(true), "TRUE", "invalid", "")},
		{"named-uint32", mapKeyCase("42", "0", identifier(42), "0x2a", "4294967296", "-1")},
	} {
		t.Run(test.name, test.run)
	}
}

func mapKeyCase[K codec.MapKey](input, other string, expected K, alias, invalid, overflow string) func(*testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()
		source := codec.NewMap[K](codec.NewString())
		values, err := source.ParseRequirement(ctx, codec.UsageArgument, input+"=one", other+"=two")
		require.NoError(t, err)
		assert.Equal(t, "one", values[expected])
		assert.Len(t, values, 2)
		plain, err := source.FormatRequirement(ctx, values)
		require.NoError(t, err)
		restored, err := source.ParseRequirement(ctx, codec.UsageDefault, plain...)
		require.NoError(t, err)
		assert.Equal(t, values, restored)
		for _, duplicate := range []string{input, alias} {
			if duplicate == "" {
				continue
			}
			_, err := source.ParseRequirement(ctx, codec.UsageArgument, input+"=one", duplicate+"=two")
			require.ErrorIs(t, err, codec.ErrIllegalArgument)
			assert.ErrorContains(t, err, "map keys must be unique")
		}
		for _, bad := range []string{invalid, overflow} {
			if bad == "" {
				continue
			}
			_, err := source.ParseRequirement(ctx, codec.UsageFlag, bad+"=value")
			require.ErrorIs(t, err, codec.ErrIllegalArgument)
			assert.ErrorContains(t, err, "map key")
		}
	}
}

func TestMapParsingArityAndAccumulation(t *testing.T) {
	ctx := context.Background()
	source := codec.NewStringMap()
	min, max := source.ExpectMinMax(ctx, codec.UsageFlag)
	assert.Equal(t, 1, min)
	assert.Equal(t, 1, max)
	min, max = source.ExpectMinMax(ctx, codec.UsageArgument)
	assert.Zero(t, min)
	assert.Equal(t, -1, max)
	_, err := source.ParseRequirement(ctx, codec.UsageFlag)
	require.ErrorIs(t, err, codec.ErrArgumentRequired)
	_, err = source.ParseRequirement(ctx, codec.UsageFlag, "a=1", "b=2")
	require.ErrorIs(t, err, codec.ErrTooManyArguments)
	_, err = source.ParseRequirement(ctx, codec.UsageArgument, "missing-separator")
	require.ErrorIs(t, err, codec.ErrIllegalArgument)
	values, err := source.ParseRequirement(ctx, codec.UsageArgument, "=empty-key", "empty=", "url=a=b,c")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"": "empty-key", "empty": "", "url": "a=b,c"}, values)
	handles, err := source.CanHandleType(ctx, reflect.TypeFor[map[string]string]())
	require.NoError(t, err)
	assert.True(t, handles)
	handles, err = source.CanHandleType(ctx, reflect.TypeFor[map[uint32]string]())
	require.NoError(t, err)
	assert.False(t, handles)
	assert.Panics(t, func() { codec.NewMap[string, string, string](nil) })
	cumulative := source.(codec.Cumulative[map[string]string])
	previous := map[string]string{"a": "first"}
	merged, err := cumulative.ParseRequirementInto(ctx, codec.UsageFlag, previous, "b=second")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"a": "first", "b": "second"}, merged)
	assert.Equal(t, map[string]string{"a": "first"}, previous)
	failed, err := cumulative.ParseRequirementInto(ctx, codec.UsageArgument, previous, "b=second", "a=duplicate")
	require.ErrorIs(t, err, codec.ErrIllegalArgument)
	assert.Equal(t, previous, failed)
	assert.Equal(t, map[string]string{"a": "first"}, previous)
	_, err = codec.NewInt32Map().ParseRequirement(ctx, codec.UsageFlag, "port=invalid")
	require.ErrorIs(t, err, codec.ErrIllegalArgument)
	assert.ErrorContains(t, err, "map value: an integer expected")
	assert.NotContains(t, err.Error(), "port")
}

func TestMapLosslessDefaultsAndIndependentValues(t *testing.T) {
	ctx := context.Background()
	source := codec.NewStringMap()
	for _, original := range []map[string]string{
		nil, {}, {"a": "1", "b": "2"}, {"a=b": "c=d"}, {"map+json:reserved": "value"}, {"\xff=\x00": "\xfe"},
	} {
		plain, err := source.FormatRequirement(ctx, original)
		require.NoError(t, err)
		restored, err := source.ParseRequirement(ctx, codec.UsageDefault, plain...)
		require.NoError(t, err)
		assert.Equal(t, original, restored)
		cloned, err := codec.CloneRequirement(ctx, source, original)
		require.NoError(t, err)
		assert.Equal(t, original, cloned)
	}
	plain, err := source.FormatRequirement(ctx, map[string]string{"z": "last", "a": "first"})
	require.NoError(t, err)
	assert.Equal(t, []string{"a=first", "z=last"}, plain)
	nested := codec.NewMap[string](codec.NewStringSlice())
	for _, original := range []map[string][]string{
		nil, {}, {"nil": nil, "empty": {}, "multiple": {"one", "two"}, "bytes=\xff": {"\xfe", "a=b"}},
	} {
		plain, err := nested.FormatRequirement(ctx, original)
		require.NoError(t, err)
		restored, err := nested.ParseRequirement(ctx, codec.UsageDefault, plain...)
		require.NoError(t, err)
		assert.Equal(t, original, restored)
		cloned, err := codec.CloneRequirement(ctx, nested, original)
		require.NoError(t, err)
		resolved, err := nested.Resolve(ctx, new(codec.ResourceScope), original)
		require.NoError(t, err)
		assert.Equal(t, original, cloned)
		assert.Equal(t, original, resolved)
		if len(original) > 0 {
			cloned["multiple"][0] = "changed clone"
			resolved["multiple"][0] = "changed runtime"
			assert.Equal(t, "one", original["multiple"][0])
		}
	}
	_, err = source.ParseRequirement(ctx, codec.UsageDefault, "map+json:invalid")
	require.ErrorIs(t, err, codec.ErrIllegalArgument)
}

func TestMapFileFailureKeepsCleanupAndDisplaySafeDefaults(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "input")
	require.NoError(t, os.WriteFile(path, []byte("data"), 0600))
	var opened []*os.File
	source := codec.NewMap[string](observedFileCodec{codec.NewFile(), &opened})
	requirements, err := source.ParseRequirement(ctx, codec.UsageDefault, "a="+path, "b="+path+"-missing")
	require.NoError(t, err)
	assert.Empty(t, opened)
	scope := new(codec.ResourceScope)
	defer func() { assert.NoError(t, scope.Close()) }()
	_, err = source.Resolve(ctx, scope, requirements)
	require.Error(t, err)
	require.Len(t, opened, 1)
	require.NoError(t, scope.Close())
	_, err = opened[0].Read(make([]byte, 1))
	assert.ErrorIs(t, err, os.ErrClosed)
	fileMap := codec.NewMap[string](codec.NewFile())
	display, err := fileMap.DisplayRequirement(ctx, map[string]codec.FileRequirement{
		"input": {Path: path, Mode: codec.FileRead, Permission: 0600},
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"input=" + path}, display)
}
