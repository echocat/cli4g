package cli

import (
	"context"
	"errors"
	"io"
)

// ErrorHandler observes input and operational failures after invocation cleanup.
// A non-nil return replaces the error; nil preserves it. The context exposes the
// invocation and the container whose execution failed. Direct container calls
// can have a nil invocation and retain caller ownership of their supplied options.
type ErrorHandler func(context.Context, InvocationLike, error) error

// DefaultErrorHandler is captured by Create when no error handler is set.
// Its built-in value writes to os.Stderr, shows help for ErrValidation, and exits with
// status 2 for input errors, 130 for context.Canceled, or 1 for other operational
// errors. Configure this variable before Create, without concurrent modification.
// Nil disables the global fallback; individual applications can disable error handling.
var DefaultErrorHandler ErrorHandler = (&RespondErrorHandlerRequirement{
	ValidationExitCode: 2, CanceledExitCode: 130, ExecutionExitCode: 1,
}).MustCreate()

type errorBoundaryKey struct{}

func ctxValue(ctx context.Context, key any) any {
	if ctx == nil {
		return nil
	}
	return ctx.Value(key)
}

func errorContext(ctx context.Context, handler ErrorHandler) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if handler != nil {
		ctx = context.WithValue(ctx, errorBoundaryKey{}, true)
	}
	return ctx
}

func joinErrors(primary, cleanup error) error {
	if cleanup == nil || errors.Is(primary, cleanup) {
		return primary
	}
	if primary == nil {
		return cleanup
	}
	return errors.Join(primary, cleanup)
}

func respondError(ctx context.Context, invocation InvocationLike, container Container, handler ErrorHandler, err error) error {
	if err == nil || handler == nil {
		return err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if invocation != nil {
		ctx = withInvocationContext(ctx, invocation)
	}
	if container != nil {
		ctx = withContainerContext(ctx, container)
	}
	if replacement := handler(ctx, invocation, err); replacement != nil {
		return replacement
	}
	return err
}

// executeDirect dispatches supplied options without acquiring their ownership.
// During parsing or invocation execution, the outer boundary reports errors.
func (this *containerBase) executeDirect(ctx context.Context, container Container, options any) error {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = withContainerContext(ctx, container)
	var handler ErrorHandler
	if this.plan != nil && this.plan.definition != nil {
		handler = this.plan.definition.errorHandler
	}
	err := this.requireOpen()
	if err == nil {
		err = ctx.Err()
	}
	if err == nil {
		err = this.execute(errorContext(ctx, handler), options)
	}
	if err == nil {
		err = ctx.Err()
	}
	if boundary, _ := ctx.Value(errorBoundaryKey{}).(bool); boundary {
		return err
	}
	if err == nil || this.plan == nil || this.plan.definition == nil {
		return err
	}
	if handler == nil {
		return err
	}
	invocation, _ := InvocationFromContext(ctx)
	if closer, ok := invocation.(io.Closer); ok {
		err = joinErrors(err, closer.Close())
	}
	return respondError(ctx, invocation, container, handler, err)
}
