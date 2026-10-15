package cli

import "io"

// NewRespondErrorHandler starts a returning responder with all exit codes zero.
// Attach the created handler with [ApplicationRequirement.WithErrorHandler].
func NewRespondErrorHandler() *RespondErrorHandlerRequirement {
	return &RespondErrorHandlerRequirement{}
}

// RespondErrorHandlerRequirement configures output responses and exit codes.
// Validation errors show help; cancellations and other errors do not.
// Zero codes return the original error joined with any response errors.
// Nonzero codes call os.Exit, which does not run caller defers.
type RespondErrorHandlerRequirement struct {
	Output             io.Writer   // nil selects context stderr, or io.Discard when unavailable
	HelpHandler        HelpHandler // nil selects the invocation's application handler
	ValidationExitCode int         // zero returns validation errors normally
	CanceledExitCode   int         // zero returns context.Canceled errors normally
	ExecutionExitCode  int         // zero returns operational errors normally
}

// WithOutput selects a caller-owned writer. Nil uses context stderr at each
// response, or io.Discard when no output is supplied by the context.
func (this *RespondErrorHandlerRequirement) WithOutput(v io.Writer) *RespondErrorHandlerRequirement {
	this.SetOutput(v)
	return this
}

func (this *RespondErrorHandlerRequirement) SetOutput(v io.Writer) { this.Output = v }

func (this *RespondErrorHandlerRequirement) WithHelpHandler(v HelpHandler) *RespondErrorHandlerRequirement {
	this.SetHelpHandler(v)
	return this
}

func (this *RespondErrorHandlerRequirement) SetHelpHandler(v HelpHandler) { this.HelpHandler = v }

func (this *RespondErrorHandlerRequirement) WithValidationExitCode(v int) *RespondErrorHandlerRequirement {
	this.SetValidationExitCode(v)
	return this
}

func (this *RespondErrorHandlerRequirement) SetValidationExitCode(v int) { this.ValidationExitCode = v }

func (this *RespondErrorHandlerRequirement) WithCanceledExitCode(v int) *RespondErrorHandlerRequirement {
	this.SetCanceledExitCode(v)
	return this
}

func (this *RespondErrorHandlerRequirement) SetCanceledExitCode(v int) { this.CanceledExitCode = v }

func (this *RespondErrorHandlerRequirement) WithExecutionExitCode(v int) *RespondErrorHandlerRequirement {
	this.SetExecutionExitCode(v)
	return this
}

func (this *RespondErrorHandlerRequirement) SetExecutionExitCode(v int) { this.ExecutionExitCode = v }
