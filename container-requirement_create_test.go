package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/echocat/cli4g/codec"
)

func TestPrimitiveDefaultsAndCommandOverride(t *testing.T) {
	type options struct {
		Int     int        `flag:"int"`
		Int8    int8       `flag:"int8"`
		Uint8   uint8      `flag:"uint8"`
		Uintptr uintptr    `flag:"uintptr"`
		Float   float32    `flag:"float"`
		Complex complex128 `flag:"complex"`
		Values  []string   `argument:"values"`
	}
	a := NewApplication[*options]().WithName("numeric").
		WithCommand(NewSimpleCommand().
			WithName("override").
			WithCodec(&constantOneCodec{}),
		).
		MustCreate()
	ctx := context.Background()
	root := a.MustParse(ctx, nil, "--int=0x2a", "--int8=-8", "--uint8=255", "--uintptr=42", "--float=1.25", "--complex=1+2i", "one", "two")
	_, v := root.Application()
	got := v
	assert.Equal(t, 42, got.Int)
	assert.Equal(t, int8(-8), got.Int8)
	assert.Equal(t, uint8(255), got.Uint8)
	assert.Equal(t, uintptr(42), got.Uintptr)
	assert.Equal(t, float32(1.25), got.Float)
	assert.Equal(t, complex128(1+2i), got.Complex)
	assert.Equal(t, []string{"one", "two"}, got.Values)
	child, err := a.Parse(ctx, nil, "--int=ignored", "override")
	require.NoError(t, err)
	_, v = child.Application()
	assert.Equal(t, 1, v.Int, "command codec did not override default int codec")
}

func TestCreateSnapshotAndDuplicates(t *testing.T) {
	r := NewSimpleApplication().
		WithName("snapshot").
		WithFlag(
			NewFlag(codec.NewString()).
				WithName("before"),
		)
	a, err := r.Create()
	require.NoError(t, err)
	r.Flags[0].(*FlagRequirement[string, string]).Names[0] = "after"
	_, found := a.Flag("after")
	assert.False(t, found, "requirement mutation changed compiled schema")
	_, found = a.Flag("before")
	assert.True(t, found, "compiled flag disappeared")
	bad := NewApplication[*rootOpts]().
		WithName("bad").
		WithFlag(
			NewFlag(codec.NewBool()).
				WithName("verbose"),
		)
	_, err = bad.Create()
	require.ErrorContains(t, err, "duplicate")
}

func TestAllPrimitiveSliceDefaults(t *testing.T) {
	type options struct {
		Bools       []bool       `flag:"bools"`
		Strings     []string     `flag:"strings"`
		Ints        []int        `flag:"ints"`
		Int8s       []int8       `flag:"int8s"`
		Int16s      []int16      `flag:"int16s"`
		Int32s      []int32      `flag:"int32s"`
		Int64s      []int64      `flag:"int64s"`
		Uints       []uint       `flag:"uints"`
		Uint8s      []uint8      `flag:"uint8s"`
		Uint16s     []uint16     `flag:"uint16s"`
		Uint32s     []uint32     `flag:"uint32s"`
		Uint64s     []uint64     `flag:"uint64s"`
		Uintptrs    []uintptr    `flag:"uintptrs"`
		Float32s    []float32    `flag:"float32s"`
		Float64s    []float64    `flag:"float64s"`
		Complex64s  []complex64  `flag:"complex64s"`
		Complex128s []complex128 `flag:"complex128s"`
	}
	a, err := NewApplication[*options]().
		WithName("all-slices").
		Create()
	require.NoError(t, err)
	inv, err := a.Parse(context.Background(), nil, "--uint16s=65535", "--int8s=-8", "--float32s=1.25", "--complex128s=1+2i")
	require.NoError(t, err)
	_, parsed := inv.Application()
	got := parsed
	assert.Equal(t, []uint16{65535}, got.Uint16s)
	assert.Equal(t, []int8{-8}, got.Int8s)
	assert.Equal(t, []float32{1.25}, got.Float32s)
	assert.Equal(t, []complex128{1 + 2i}, got.Complex128s)
}
