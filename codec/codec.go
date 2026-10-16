package codec

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

var (
	ErrArgumentRequired    = NewValidationError("argument is required")
	ErrNoArgumentExpected  = NewValidationError("no argument expected")
	ErrTooManyArguments    = NewValidationError("too many arguments provided")
	ErrTooFewArguments     = NewValidationError("too few arguments provided")
	ErrIllegalArgument     = NewValidationError("illegal argument")
	ErrInvalid             = errors.New("invalid codec")
	ErrResourceRequirement = errors.New("requirements must not hold resources")
)

// Codec turns input configuration R into runtime values V.
// Typed defaults use R, which must contain no resources. Implementations must:
//
//   - Be immutable and concurrent-safe; honor operation contexts.
//   - Parse and format requirements without acquiring resources or mutating inputs.
//   - Preserve all configuration through FormatRequirement -> ParseRequirement(UsageDefault),
//     including nil/empty distinctions and independent mutable data.
//   - Register acquired resources with [Scope.OnClose] immediately, even on failure.
//   - Mark user input errors with [NewValidationError]; leave technical errors unmarked.
//   - Describe requirements in error messages, never supplied input values.
//     ValidationError messages are trusted author-controlled output. Unmarked
//     runtime errors receive neutral messages at codec/CLI boundaries, retaining causes.
type Codec[V, R any] interface {
	// CanHandleType reports supported runtime types; support must not vary by context.
	CanHandleType(context.Context, reflect.Type) (bool, error)
	// ExpectMinMax reports token counts; -1 as maximum means unbounded.
	// Counts are stable grammar properties and must not vary by context.
	ExpectMinMax(context.Context, Usage) (int, int)
	// ParseRequirement validates input tokens and returns resource-free configuration.
	ParseRequirement(context.Context, Usage, ...string) (R, error)
	// FormatRequirement serializes R losslessly. Its output is not necessarily help-safe.
	FormatRequirement(context.Context, R) ([]string, error)
	// DisplayRequirement returns help-safe text, redacting secrets when needed.
	// It need not be parseable; wrappers must preserve it.
	DisplayRequirement(context.Context, R) ([]string, error)
	// Resolve creates V using invocation-owned configuration and the resource scope.
	Resolve(context.Context, Scope, R) (V, error)
	// ToRaw adapts typed operations for registration; use [Raw] for custom codecs.
	ToRaw(context.Context) Codec[any, any]
}

// RequirementCloner is an optional data-only fast path. Its result must be
// equivalent to the required FormatRequirement -> ParseRequirement roundtrip.
type RequirementCloner[R any] interface {
	CloneRequirement(context.Context, R) (R, error)
}

// TypeBinder optionally specializes a registry codec for one concrete value type
// during compilation. Unsupported types return nil, false, nil. A successful
// binding must return an immutable, independently usable codec; it must not
// change the resolver's behavior for other bindings or subsequent calls.
type TypeBinder interface {
	BindType(context.Context, reflect.Type) (Codec[any, any], bool, error)
}

// EnsureArgumentsMatching checks token counts against [Codec.ExpectMinMax].
// Count errors match [ErrValidation]; cancellation and invalid metadata do not.
func EnsureArgumentsMatching[V, R any](ctx context.Context, usage Usage, source Codec[V, R], args ...string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	min, max := source.ExpectMinMax(ctx, usage)
	if err := ctx.Err(); err != nil {
		return err
	}
	if min < 0 || max < -1 || max != -1 && min > max {
		return fmt.Errorf("%w: invalid arity %d,%d", ErrInvalid, min, max)
	}
	if max == 0 && len(args) > 0 {
		return ErrNoArgumentExpected
	}
	if max >= 0 && len(args) > max {
		return fmt.Errorf("%w: maximum is %d; but got %d arguments", ErrTooManyArguments, max, len(args))
	}
	if len(args) < min {
		if len(args) == 0 {
			return ErrArgumentRequired
		}
		return fmt.Errorf("%w: minimum is %d; but got %d arguments", ErrTooFewArguments, min, len(args))
	}
	return nil
}
