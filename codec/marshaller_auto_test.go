package codec_test

import (
	"context"
	"encoding"
	"io"
	"os"
	"reflect"
	"testing"

	"github.com/echocat/cli4g/codec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type automaticTextResource struct{ file *os.File }

func (*automaticTextResource) MarshalText() ([]byte, error) {
	panic("automatic text detection and requirement inspection must not marshal live values")
}

func (value *automaticTextResource) UnmarshalText(input []byte) error {
	file, err := os.Open(os.DevNull)
	value.file = file
	if err != nil {
		return err
	}
	switch string(input) {
	case "invalid":
		return codec.NewValidationError(io.ErrUnexpectedEOF)
	case "failure":
		return io.ErrUnexpectedEOF
	}
	return nil
}

func (value *automaticTextResource) Close() error { return value.file.Close() }

func TestAutomaticMarshallerOwnsFreshTargetsIncludingPartialFailures(t *testing.T) {
	ctx := context.Background()
	for _, typ := range []reflect.Type{reflect.TypeFor[automaticTextResource](), reflect.TypeFor[*automaticTextResource]()} {
		source, found, err := codec.AutomaticMarshaller(ctx, typ)
		require.NoError(t, err)
		require.True(t, found)
		matches, err := source.CanHandleType(ctx, typ)
		require.NoError(t, err)
		assert.True(t, matches)
		for _, input := range []string{"success", "invalid", "failure"} {
			requirement, err := source.ParseRequirement(ctx, codec.UsageFlag, input)
			require.NoError(t, err)
			plain, err := source.FormatRequirement(ctx, requirement)
			require.NoError(t, err)
			assert.Equal(t, []string{input}, plain)
			display, err := source.DisplayRequirement(ctx, requirement)
			require.NoError(t, err)
			assert.Equal(t, plain, display)
			scope := new(codec.ResourceScope)
			value, err := source.Resolve(ctx, scope, requirement)
			if input == "success" {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, io.ErrUnexpectedEOF)
				assert.Equal(t, input == "invalid", codec.IsValidationError(err))
			}
			assert.Equal(t, typ, reflect.TypeOf(value))
			var file *os.File
			if typ.Kind() == reflect.Pointer {
				file = value.(*automaticTextResource).file
			} else {
				file = value.(automaticTextResource).file
			}
			_, err = file.Stat()
			require.NoError(t, err)
			require.NoError(t, scope.Close())
			require.NoError(t, scope.Close())
			_, err = file.Read(make([]byte, 1))
			assert.ErrorIs(t, err, os.ErrClosed)
		}
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		_, err = source.Resolve(canceled, new(codec.ResourceScope), "success")
		assert.ErrorIs(t, err, context.Canceled)
		_, err = source.Resolve(ctx, nil, "success")
		assert.ErrorIs(t, err, codec.ErrInvalid)
	}
}

type automaticMarshalOnly struct{}

func (automaticMarshalOnly) MarshalText() ([]byte, error) { return nil, nil }

type automaticUnmarshalOnly struct{}

func (*automaticUnmarshalOnly) UnmarshalText([]byte) error { return nil }

func TestAutomaticMarshallerRejectsUnsupportedTypesAndCanceledDetection(t *testing.T) {
	for _, typ := range []reflect.Type{
		nil,
		reflect.TypeFor[string](),
		reflect.TypeFor[encoding.TextUnmarshaler](),
		reflect.TypeFor[automaticMarshalOnly](),
		reflect.TypeFor[automaticUnmarshalOnly](),
		reflect.TypeFor[[]automaticTextResource](),
	} {
		source, found, err := codec.AutomaticMarshaller(context.Background(), typ)
		require.NoError(t, err)
		assert.False(t, found)
		assert.Nil(t, source)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	source, _, err := codec.AutomaticMarshaller(ctx, reflect.TypeFor[automaticTextResource]())
	assert.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, source)
}
