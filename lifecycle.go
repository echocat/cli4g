package cli

import "context"

// Validatable checks decoded options before initialization.
// Mark invalid user input with [NewValidationErrorf]; return technical errors normally.
type Validatable interface {
	// Validate checks the fully decoded options before any initializer runs.
	// It must not acquire resources or mutate the options.
	// Mark invalid input with NewValidationError or NewValidationErrorf; unmarked
	// failures remain operational and retain their original causes.
	// Warning: marked messages are displayed verbatim and must not include input
	// values or confidential data. Unmarked failures receive a neutral message.
	Validate(context.Context) error
}

// Initializable prepares options after successful validation.
// Pointer-receiver methods retain assignments, including for struct options.
// See [Invocation.Close] and [ApplicationRequirement.WithInitializeHandler].
type Initializable interface {
	// Initialize prepares options. Retained resources must be released by
	// options' Close method, including after partial initialization.
	Initialize(context.Context) error
}
