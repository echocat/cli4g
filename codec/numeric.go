package codec

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// Shared immutable metadata for a primitive numeric type.
type numericConfig struct {
	typ  reflect.Type
	bits uint8
}

func signedParseError(input string, bits int, cause error) error {
	message := "an integer expected"
	if errors.Is(cause, strconv.ErrRange) {
		maximum := int64(uint64(1)<<(bits-1) - 1)
		if strings.HasPrefix(input, "-") {
			message = fmt.Sprintf("value should not be smaller than %d", -maximum-1)
		} else {
			message = fmt.Sprintf("value should not be larger than %d", maximum)
		}
	}
	return NewValidationError(message, ErrIllegalArgument, cause)
}

func unsignedParseError(bits int, cause error) error {
	message := "0 or a positive integer expected"
	if errors.Is(cause, strconv.ErrRange) {
		maximum := ^uint64(0) >> (64 - bits)
		message = fmt.Sprintf("value should not be larger than %d", maximum)
	}
	return NewValidationError(message, ErrIllegalArgument, cause)
}

func floatMaximum(bits int) string {
	maximum := math.MaxFloat64
	if bits == 32 {
		maximum = math.MaxFloat32
	}
	return strconv.FormatFloat(maximum, 'g', -1, 64)
}

func floatParseError(input string, bits int, cause error) error {
	message := "a number expected"
	if errors.Is(cause, strconv.ErrRange) {
		if strings.HasPrefix(input, "-") {
			message = "value should not be smaller than -" + floatMaximum(bits)
		} else {
			message = "value should not be larger than " + floatMaximum(bits)
		}
	}
	return NewValidationError(message, ErrIllegalArgument, cause)
}

func complexParseError(bits int, cause error) error {
	message := "a complex number expected (for example 1+2i)"
	if errors.Is(cause, strconv.ErrRange) {
		limit := floatMaximum(bits / 2)
		message = "real and imaginary parts should be between -" + limit + " and " + limit
	}
	return NewValidationError(message, ErrIllegalArgument, cause)
}
