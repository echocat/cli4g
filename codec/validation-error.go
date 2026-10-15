package codec

import (
	"errors"
	"fmt"
)

// ErrValidation identifies invalid input, independently of the failing operation.
var ErrValidation = errors.New("validation failed")

// ValidationError marks an input failure without changing its message or cause.
type ValidationError struct {
	Err error
}

func (this *ValidationError) Error() string {
	if this == nil || this.Err == nil {
		return ErrValidation.Error()
	}
	return this.Err.Error()
}

func (this *ValidationError) Unwrap() error {
	if this == nil {
		return nil
	}
	return this.Err
}

func (this *ValidationError) Is(target error) bool { return this != nil && target == ErrValidation }

// NewValidationError marks an input failure. Nil stays nil and already marked
// errors are returned unchanged. Operational failures should be returned directly.
func NewValidationError(err error) error {
	if err == nil || IsValidationError(err) {
		return err
	}
	return &ValidationError{Err: err}
}

// NewValidationErrorf formats an input failure using fmt.Errorf, including %w.
func NewValidationErrorf(format string, args ...any) error {
	return NewValidationError(fmt.Errorf(format, args...))
}

// IsValidationError recognizes marked errors through wrapping and errors.Join.
func IsValidationError(err error) bool { return errors.Is(err, ErrValidation) }
