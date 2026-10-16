package codec

import (
	"context"
	"slices"
)

// MessageError separates a user-facing message from its diagnostic causes.
// Error never appends cause messages; errors.Is and errors.As traverse Unwrap.
type MessageError struct {
	message string
	causes  []error
}

// NewMessageError creates an error whose displayed text is message only.
// Nil causes are ignored; the causes are copied and remain available to errors.Is
// and errors.As. Warning: message is displayed verbatim and must not contain
// passwords, input values, or other confidential data.
func NewMessageError(message string, causes ...error) *MessageError {
	result := &MessageError{message: message}
	for _, cause := range causes {
		if cause != nil {
			result.causes = append(result.causes, cause)
		}
	}
	return result
}

func (this *MessageError) Error() string {
	if this == nil {
		return "operation failed"
	}
	return this.message
}

func (this *MessageError) Unwrap() []error {
	if this == nil {
		return nil
	}
	return slices.Clone(this.causes)
}

// SanitizeError retains explicitly user-facing errors and otherwise replaces
// their displayed text with fallback while preserving the original cause and
// classification. Validation errors are trusted author-supplied messages.
// Bare context cancellation/deadline errors are already safe. Nil stays nil.
func SanitizeError(err error, fallback string) error {
	if err == nil || IsValidationError(err) || err == context.Canceled || err == context.DeadlineExceeded {
		return err
	}
	if _, ok := err.(*MessageError); ok {
		return err
	}
	return NewMessageError(fallback, err)
}

func prefixError(prefix string, err error) error {
	visible := SanitizeError(err, "value could not be processed")
	return NewMessageError(prefix+": "+visible.Error(), visible)
}
