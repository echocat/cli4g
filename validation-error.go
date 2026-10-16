package cli

import "github.com/echocat/cli4g/codec"

// ErrValidation is the codec package's shared invalid-input sentinel.
var ErrValidation = codec.ErrValidation

// ValidationError is the codec package's input-error wrapper.
type ValidationError = codec.ValidationError

// NewValidationError marks invalid input with a message and diagnostic causes.
// Error displays only message; errors.Is and errors.As can inspect the causes.
// Warning: message is displayed verbatim. Authors must not include passwords,
// supplied input values, or other confidential data.
func NewValidationError(message string, causes ...error) error {
	return codec.NewValidationError(message, causes...)
}

// NewValidationErrorf formats an input failure, preserving causes supplied with %w.
// Warning: %w also displays the cause text. Use NewValidationError with a separate
// message and causes to avoid exposing input or other confidential data.
func NewValidationErrorf(format string, args ...any) error {
	return codec.NewValidationErrorf(format, args...)
}

// IsValidationError recognizes the shared sentinel through wrapping and errors.Join.
func IsValidationError(err error) bool { return codec.IsValidationError(err) }
