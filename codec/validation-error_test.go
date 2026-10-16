package codec

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidationErrorFormattingWrappingAndIdentity(t *testing.T) {
	first, second := errors.New("first cause"), errors.New("second cause")
	err := NewValidationErrorf("invalid value %d: %w; %w", 7, first, second)
	assert.Equal(t, "invalid value 7: first cause; second cause", err.Error())
	assert.ErrorIs(t, err, first)
	assert.ErrorIs(t, err, second)
	assert.ErrorIs(t, err, ErrValidation)
	var typed *ValidationError
	require.ErrorAs(t, err, &typed)
	require.NotNil(t, typed.Unwrap())
	for _, marked := range []error{err, fmt.Errorf("outer: %w", err), errors.Join(io.ErrUnexpectedEOF, err)} {
		assert.True(t, IsValidationError(marked))
		assert.Same(t, marked, SanitizeError(marked, "fallback"), "authors control explicitly marked messages")
	}
	assert.Nil(t, SanitizeError(nil, "fallback"))
	assert.False(t, IsValidationError(nil))
	assert.False(t, IsValidationError(io.ErrUnexpectedEOF))
	var nilError *ValidationError
	assert.Equal(t, ErrValidation.Error(), nilError.Error())
	assert.Nil(t, nilError.Unwrap())
	assert.False(t, nilError.Is(ErrValidation))
}

func TestSpecificCodecErrorsRetainValidationCategory(t *testing.T) {
	for _, sentinel := range []error{ErrArgumentRequired, ErrNoArgumentExpected, ErrTooManyArguments, ErrTooFewArguments, ErrIllegalArgument, ErrSpecialFile} {
		wrapped := fmt.Errorf("input: %w", sentinel)
		assert.True(t, IsValidationError(wrapped))
		assert.ErrorIs(t, wrapped, sentinel)
	}
	assert.False(t, IsValidationError(ErrInvalid))
	assert.False(t, IsValidationError(ErrResourceRequirement))
}

type unmarkedIOValue struct{}

func (*unmarkedIOValue) Set(string) error { return io.ErrUnexpectedEOF }
func (*unmarkedIOValue) String() string   { return "unused" }

type unmarkedIOText struct{}

func (unmarkedIOText) MarshalText() ([]byte, error) { return []byte("unused"), nil }
func (*unmarkedIOText) UnmarshalText([]byte) error  { return io.ErrUnexpectedEOF }

func TestValueAndTextAdaptersPreserveUnmarkedReadFailures(t *testing.T) {
	ctx := context.Background()
	value, err := NewValue[unmarkedIOValue]()
	require.NoError(t, err)
	pointer, err := NewValueP(func() *unmarkedIOValue { return new(unmarkedIOValue) })
	require.NoError(t, err)
	automatic, found, err := AutomaticFlagValue(ctx, reflect.TypeFor[unmarkedIOValue]())
	require.NoError(t, err)
	require.True(t, found)
	for _, test := range []struct {
		name   string
		source Codec[any, any]
	}{
		{"value", value.ToRaw(ctx)},
		{"pointer", pointer.ToRaw(ctx)},
		{"automatic", automatic},
		{"text", NewMarshaller[unmarkedIOText]().ToRaw(ctx)},
		{"text-pointer", NewMarshallerP(func() *unmarkedIOText { return new(unmarkedIOText) }).ToRaw(ctx)},
	} {
		t.Run(test.name, func(t *testing.T) {
			requirement, err := test.source.ParseRequirement(ctx, UsageFlag, "input")
			require.NoError(t, err)
			scope := new(ResourceScope)
			defer func() { assert.NoError(t, scope.Close()) }()
			_, err = test.source.Resolve(ctx, scope, requirement)
			assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
			assert.NotContains(t, err.Error(), io.ErrUnexpectedEOF.Error())
			assert.False(t, IsValidationError(err))
			assert.NotErrorIs(t, err, ErrIllegalArgument)
		})
	}
}
