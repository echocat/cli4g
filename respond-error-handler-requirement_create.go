package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"sync"
)

func (this *RespondErrorHandlerRequirement) Create() (ErrorHandler, error) {
	if this == nil {
		return nil, fmt.Errorf("nil respond error handler requirement")
	}
	output := this.Output
	help := this.HelpHandler
	validationExitCode, executionExitCode := this.ValidationExitCode, this.ExecutionExitCode
	canceledExitCode := this.CanceledExitCode
	var responseMutex sync.Mutex
	return func(ctx context.Context, invocation InvocationLike, original error) error {
		if original == nil {
			return nil
		}
		if ctx == nil {
			ctx = context.Background()
		}
		// Reporting a cancellation must still work. Preserve tracing/context data.
		ctx = context.WithoutCancel(ctx)
		if invocation == nil || (reflect.ValueOf(invocation).Kind() == reflect.Pointer && reflect.ValueOf(invocation).IsNil()) {
			invocation, _ = InvocationFromContext(ctx)
		}
		validation := IsValidationError(original)
		exitCode := executionExitCode
		message := fmt.Sprintf("Error: %v\n", original)
		if validation {
			exitCode = validationExitCode
			message += "\n"
		} else if errors.Is(original, context.Canceled) {
			exitCode = canceledExitCode
		}
		responseMutex.Lock()
		defer responseMutex.Unlock()
		n, err := func() (int, error) {
			handlerOutputMutex.Lock()
			defer handlerOutputMutex.Unlock()
			return io.WriteString(handlerOutput(ctx, output), message)
		}()
		result := original
		if err != nil {
			result = errors.Join(result, fmt.Errorf("write error response: %w", err))
		} else if n != len(message) {
			result = errors.Join(result, fmt.Errorf("write error response: %w", io.ErrShortWrite))
		} else if validation {
			handler := help
			if handler == nil && invocation != nil {
				handler = invocation.HelpHandler()
			}
			if handler == nil {
				result = errors.Join(result, fmt.Errorf("input error response has no help handler"))
			} else if err := handler(ctx, invocation); err != nil {
				result = errors.Join(result, fmt.Errorf("respond with help: %w", err))
			}
		}
		if exitCode != 0 {
			os.Exit(exitCode)
		}
		return result
	}, nil
}

// MustCreate creates the handler and panics on invalid configuration.
func (this *RespondErrorHandlerRequirement) MustCreate() ErrorHandler {
	handler, err := this.Create()
	if err != nil {
		panic(err)
	}
	return handler
}
