package cli

import "github.com/echocat/cli4g/codec"

// ErrValidation is the codec package's shared invalid-input sentinel.
var ErrValidation = codec.ErrValidation

// ValidationError is the codec package's input-error wrapper.
type ValidationError = codec.ValidationError

// NewValidationError marks an input failure without changing its message or cause.
func NewValidationError(err error) error { return codec.NewValidationError(err) }

// NewValidationErrorf formats an input failure, preserving causes supplied with %w.
func NewValidationErrorf(format string, args ...any) error {
	return codec.NewValidationErrorf(format, args...)
}

// IsValidationError recognizes the shared sentinel through wrapping and errors.Join.
func IsValidationError(err error) bool { return codec.IsValidationError(err) }
