package codec

import (
	"context"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const confidentialInput = "private-input-DO-NOT-DISPLAY-9f21"

func TestMessageErrorSeparatesDisplayFromMultipleCauses(t *testing.T) {
	native := &strconv.NumError{Func: "ParseInt", Num: confidentialInput, Err: strconv.ErrSyntax}
	causes := []error{ErrIllegalArgument, nil, native}
	err := NewValidationError("an integer expected", causes...)
	causes[0] = io.EOF
	for _, rendered := range []string{err.Error(), fmt.Sprintf("%v", err), fmt.Sprintf("%+v", err), fmt.Errorf("flag --password: %w", err).Error()} {
		assert.NotContains(t, rendered, confidentialInput)
		assert.NotContains(t, rendered, "strconv")
	}
	assert.Equal(t, "an integer expected", err.Error())
	assert.ErrorIs(t, err, ErrIllegalArgument)
	assert.ErrorIs(t, err, ErrValidation)
	assert.ErrorIs(t, err, strconv.ErrSyntax)
	assert.ErrorIs(t, err, native)
	var original *strconv.NumError
	require.ErrorAs(t, err, &original)
	assert.Same(t, native, original)
	assert.Equal(t, confidentialInput, original.Num, "diagnostic data remains deliberately accessible")
	var multiple interface{ Unwrap() []error }
	require.ErrorAs(t, err, &multiple)
	unwrapped := multiple.Unwrap()
	require.Len(t, unwrapped, 2)
	unwrapped[0] = nil
	assert.ErrorIs(t, err, ErrIllegalArgument, "Unwrap must not expose mutable internal storage")
	assert.Same(t, err, SanitizeError(err, "fallback"))
	operational := SanitizeError(native, "value could not be decoded")
	assert.Equal(t, "value could not be decoded", operational.Error())
	assert.ErrorIs(t, operational, native)
	assert.NotErrorIs(t, operational, ErrValidation)
	assert.Same(t, operational, SanitizeError(operational, "another fallback"))
	assert.Nil(t, SanitizeError(nil, "fallback"))
	for _, canceled := range []error{context.Canceled, context.DeadlineExceeded} {
		assert.Equal(t, canceled, SanitizeError(canceled, "fallback"))
		wrapped := fmt.Errorf("%s: %w", confidentialInput, canceled)
		safe := SanitizeError(wrapped, "operation canceled")
		assert.NotContains(t, safe.Error(), confidentialInput)
		assert.ErrorIs(t, safe, canceled)
	}
}

func TestIntegerErrorsDescribeExactBoundsAndRetainNativeCauses(t *testing.T) {
	for _, test := range []struct {
		name     string
		source   any
		bits     int
		unsigned bool
	}{
		{"int", NewInt(), strconv.IntSize, false},
		{"int8", NewInt8(), 8, false},
		{"int16", NewInt16(), 16, false},
		{"int32", NewInt32(), 32, false},
		{"int64", NewInt64(), 64, false},
		{"uint", NewUint(), strconv.IntSize, true},
		{"uint8", NewUint8(), 8, true},
		{"uint16", NewUint16(), 16, true},
		{"uint32", NewUint32(), 32, true},
		{"uint64", NewUint64(), 64, true},
		{"uintptr", NewUintptr(), strconv.IntSize, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			source, err := Snapshot(context.Background(), test.source)
			require.NoError(t, err)
			bits := test.bits
			if !test.unsigned {
				bits--
			}
			power := new(big.Int).Lsh(big.NewInt(1), uint(bits))
			maximum := new(big.Int).Sub(new(big.Int).Set(power), big.NewInt(1))
			minimum := new(big.Int)
			if !test.unsigned {
				minimum.Neg(power)
			}
			for _, boundary := range []string{minimum.String(), maximum.String()} {
				_, err := source.ParseRequirement(context.Background(), UsageFlag, boundary)
				require.NoError(t, err, boundary)
			}
			tooLarge := new(big.Int).Add(maximum, big.NewInt(1)).String()
			_, err = source.ParseRequirement(context.Background(), UsageFlag, tooLarge)
			assert.EqualError(t, err, "value should not be larger than "+maximum.String())
			assert.ErrorIs(t, err, strconv.ErrRange)
			tooSmall := new(big.Int).Sub(minimum, big.NewInt(1)).String()
			_, err = source.ParseRequirement(context.Background(), UsageFlag, tooSmall)
			if test.unsigned {
				assert.EqualError(t, err, "0 or a positive integer expected")
			} else {
				assert.EqualError(t, err, "value should not be smaller than "+minimum.String())
				assert.ErrorIs(t, err, strconv.ErrRange)
			}
			assert.ErrorIs(t, err, ErrIllegalArgument)
			var native *strconv.NumError
			require.ErrorAs(t, err, &native)
			assert.Equal(t, tooSmall, native.Num)
			_, err = source.ParseRequirement(context.Background(), UsageFlag, confidentialInput)
			assert.ErrorIs(t, err, strconv.ErrSyntax)
			assert.NotContains(t, err.Error(), confidentialInput)
			assert.NotContains(t, err.Error(), "strconv")
		})
	}
}

func TestFloatingAndComplexErrorsDescribeRequirements(t *testing.T) {
	for _, test := range []struct {
		name           string
		source         any
		input, message string
	}{
		{"float32 upper", NewFloat32(), "1e100", "value should not be larger than 3.4028234663852886e+38"},
		{"float32 lower", NewFloat32(), "-1e100", "value should not be smaller than -3.4028234663852886e+38"},
		{"float64 upper", NewFloat64(), "1e10000", "value should not be larger than 1.7976931348623157e+308"},
		{"float64 lower", NewFloat64(), "-1e10000", "value should not be smaller than -1.7976931348623157e+308"},
		{"float syntax", NewFloat64(), confidentialInput, "a number expected"},
		{"complex syntax", NewComplex128(), confidentialInput, "a complex number expected (for example 1+2i)"},
		{"complex64 range", NewComplex64(), "1+1e100i", "real and imaginary parts should be between -3.4028234663852886e+38 and 3.4028234663852886e+38"},
		{"complex128 range", NewComplex128(), "1e10000+2i", "real and imaginary parts should be between -1.7976931348623157e+308 and 1.7976931348623157e+308"},
	} {
		t.Run(test.name, func(t *testing.T) {
			source, err := Snapshot(context.Background(), test.source)
			require.NoError(t, err)
			_, err = source.ParseRequirement(context.Background(), UsageFlag, test.input)
			assert.EqualError(t, err, test.message)
			assert.ErrorIs(t, err, ErrIllegalArgument)
			var native *strconv.NumError
			require.ErrorAs(t, err, &native)
			assert.Equal(t, test.input, native.Num)
		})
	}
}

func TestBuiltinAndCompositeErrorsDoNotEchoInputs(t *testing.T) {
	enumeration, err := NewEnum(NewString(), "public-one", "public-two")
	require.NoError(t, err)
	for _, test := range []struct {
		name   string
		source any
		args   []string
		want   string
	}{
		{"bool", NewBool(), []string{confidentialInput}, "one of true, false"},
		{"date", NewDate(), []string{confidentialInput}, "a valid date expected"},
		{"datetime", NewDateTime(), []string{confidentialInput}, "a valid date and time expected"},
		{"duration", NewDuration(), []string{confidentialInput}, "a duration with units"},
		{"enum", enumeration, []string{confidentialInput}, `expected one of: "public-one", "public-two"`},
		{"slice", NewSlice(NewInt32()), []string{confidentialInput}, "slice element 0: an integer expected"},
		{"map key", NewMap[int16](NewString()), []string{confidentialInput + "=value"}, "map key: an integer expected"},
		{"map value", NewMap[string](NewInt8()), []string{confidentialInput + "=" + confidentialInput}, "map value: an integer expected"},
		{"nested map", NewMap[string](NewSlice(NewUint8())), []string{confidentialInput + "=" + confidentialInput}, "map value: slice element 0: 0 or a positive integer expected"},
		{"duplicate key", NewMap[string](NewString()), []string{confidentialInput + "=one", confidentialInput + "=two"}, "map keys must be unique"},
	} {
		t.Run(test.name, func(t *testing.T) {
			source, err := Snapshot(context.Background(), test.source)
			require.NoError(t, err)
			usage := UsageFlag
			if len(test.args) > 1 {
				usage = UsageArgument
			}
			_, err = source.ParseRequirement(context.Background(), usage, test.args...)
			require.ErrorIs(t, err, ErrIllegalArgument)
			assert.Contains(t, err.Error(), test.want)
			assert.NotContains(t, err.Error(), confidentialInput)
			assert.NotContains(t, err.Error(), "strconv")
		})
	}
}

func TestCalendarErrorsShowFormatsWithoutEchoingInput(t *testing.T) {
	_, err := NewDate().WithFormats("2006-01-02").ParseRequirement(context.Background(), UsageFlag, "2026-02-30")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "day must exist in the selected month")
	assert.Contains(t, err.Error(), "2026-07-15")
	assert.NotContains(t, err.Error(), "2026-02-30")
	var native *time.ParseError
	require.ErrorAs(t, err, &native)
	assert.Equal(t, "2026-02-30", native.Value)
	for _, source := range []any{NewDate(), NewDateTime()} {
		decoder, err := Snapshot(context.Background(), source)
		require.NoError(t, err)
		_, err = decoder.ParseRequirement(context.Background(), UsageFlag, confidentialInput)
		var detail interface{ Unwrap() []error }
		require.ErrorAs(t, err, &detail)
		assert.Greater(t, len(detail.Unwrap()), 1, "retain the original parse attempts")
	}
}

func TestFileErrorsHidePathsButPreserveFilesystemCauses(t *testing.T) {
	path := filepath.Join(t.TempDir(), confidentialInput)
	source := NewFile()
	requirement, err := source.ParseRequirement(context.Background(), UsageFlag, path)
	require.NoError(t, err)
	scope := new(ResourceScope)
	defer func() { assert.NoError(t, scope.Close()) }()
	_, err = source.Resolve(context.Background(), scope, requirement)
	assert.EqualError(t, err, "file or parent directory must exist")
	assert.NotErrorIs(t, err, ErrValidation)
	assert.ErrorIs(t, err, os.ErrNotExist)
	var native *os.PathError
	require.ErrorAs(t, err, &native)
	assert.Equal(t, path, native.Path)
	assert.NotContains(t, err.Error(), confidentialInput)
	assert.NotContains(t, err.Error(), path)
}

func TestResourceCleanupErrorHidesInputAndPreservesCause(t *testing.T) {
	native := &os.PathError{Op: "close", Path: confidentialInput, Err: os.ErrPermission}
	scope := new(ResourceScope)
	scope.OnClose(func() error { return native })
	err := scope.Close()
	assert.EqualError(t, err, "resource could not be closed")
	assert.ErrorIs(t, err, native)
	assert.ErrorIs(t, err, os.ErrPermission)
	assert.NotContains(t, err.Error(), confidentialInput)
	var original *os.PathError
	require.ErrorAs(t, err, &original)
	assert.Same(t, native, original)
	assert.Same(t, err, scope.Close())
}

func TestStructuredConfigurationErrorsDoNotEchoData(t *testing.T) {
	for _, test := range []struct {
		source any
		prefix string
	}{
		{NewFile(), "file+json:"},
		{NewDate(), "date+json:"},
		{NewDateTime(), "datetime+json:"},
		{NewSlice(NewString()), "slice+json:"},
		{NewMap[string](NewString()), "map+json:"},
	} {
		source, err := Snapshot(context.Background(), test.source)
		require.NoError(t, err)
		input := test.prefix + `{"` + confidentialInput + `":"` + confidentialInput + `"}`
		_, err = source.ParseRequirement(context.Background(), UsageDefault, input)
		require.ErrorIs(t, err, ErrIllegalArgument)
		assert.False(t, strings.Contains(err.Error(), confidentialInput))
	}
}
