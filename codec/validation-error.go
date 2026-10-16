package codec

import (
	"errors"
	"fmt"
)

// ErrValidation identifies invalid input, independently of the failing operation.
var ErrValidation = errors.New("validation failed")

// ValidationError marks invalid input with a user-facing message and diagnostic
// causes. Its Error method never appends the cause messages.
type ValidationError struct {
	*MessageError
}

func (this *ValidationError) Error() string {
	if this == nil || this.MessageError == nil {
		return ErrValidation.Error()
	}
	return this.MessageError.Error()
}

func (this *ValidationError) Unwrap() []error {
	if this == nil {
		return nil
	}
	return this.MessageError.Unwrap()
}

func (this *ValidationError) Is(target error) bool { return this != nil && target == ErrValidation }

// NewValidationError marks invalid input, displaying message without appending
// the causes. Causes remain available through errors.Is and errors.As.
// Warning: message is displayed verbatim. Never include passwords, supplied input
// values, or other confidential data in it; authors are responsible for its safety.
func NewValidationError(message string, causes ...error) error {
	return &ValidationError{MessageError: NewMessageError(message, causes...)}
}

// NewValidationErrorf formats an input failure using fmt.Errorf, including %w.
// Warning: formatting %v, %s, or %w includes the argument's text in the displayed
// message. To retain a cause without displaying it, use NewValidationError with
// a separate message and causes. Authors are responsible for message safety.
func NewValidationErrorf(format string, args ...any) error {
	err := fmt.Errorf(format, args...)
	return NewValidationError(err.Error(), err)
}

// IsValidationError recognizes marked errors through wrapping and errors.Join.
func IsValidationError(err error) bool { return errors.Is(err, ErrValidation) }
