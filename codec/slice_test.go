package codec

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSliceParsingAndFormatting(t *testing.T) {
	t.Run("generic", func(t *testing.T) {
		testSliceParsingAndFormatting(t, NewSlice(NewInt8()))
	})
	t.Run("shortcut", func(t *testing.T) {
		testSliceParsingAndFormatting(t, NewInt8Slice())
	})
}

func testSliceParsingAndFormatting(t *testing.T, c Codec[[]int8, []int8]) {
	t.Helper()
	ctx := context.Background()
	//goland:noinspection GoShadowedVar,GoReservedWordUsedAsName
	min, max := c.ExpectMinMax(ctx, UsageArgument)
	assert.Equal(t, 0, min)
	assert.Equal(t, -1, max)
	min, max = c.ExpectMinMax(ctx, UsageFlag)
	assert.Equal(t, 1, min)
	assert.Equal(t, 1, max)
	_, err := c.ParseRequirement(ctx, UsageFlag)
	require.ErrorIs(t, err, ErrArgumentRequired)
	_, err = c.ParseRequirement(ctx, UsageFlag, "1", "2")
	require.ErrorIs(t, err, ErrTooManyArguments)
	handles, err := c.CanHandleType(ctx, reflect.TypeFor[[]int8]())
	require.NoError(t, err)
	assert.True(t, handles)
	handles, err = c.CanHandleType(ctx, reflect.TypeFor[[]uint8]())
	require.NoError(t, err)
	assert.False(t, handles)
	empty, err := c.ParseRequirement(ctx, UsageArgument)
	require.NoError(t, err)
	assert.Nil(t, empty)
	values, err := c.ParseRequirement(ctx, UsageArgument, "-8", "7")
	require.NoError(t, err)
	assert.Equal(t, []int8{-8, 7}, values)
	formatted, err := c.FormatRequirement(ctx, values)
	require.NoError(t, err)
	assert.Equal(t, []string{"-8", "7"}, formatted)
	_, err = c.ParseRequirement(ctx, UsageArgument, "1", "128")
	require.ErrorIs(t, err, ErrIllegalArgument)
	assert.ErrorContains(t, err, "slice element 1")
	formatted, err = c.FormatRequirement(ctx, nil)
	require.NoError(t, err)
	assert.Nil(t, formatted)
	nested := NewSlice(NewSlice(NewString()))
	structured, err := nested.FormatRequirement(ctx, [][]string{{"a", "b"}})
	require.NoError(t, err)
	restored, err := nested.ParseRequirement(ctx, UsageDefault, structured...)
	require.NoError(t, err)
	assert.Equal(t, [][]string{{"a", "b"}}, restored)
}
