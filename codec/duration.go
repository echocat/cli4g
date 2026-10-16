package codec

import (
	"context"
	"reflect"
	"time"
)

// NewDuration constructs the standard time.Duration codec using Go duration syntax.
// Zero and negative values are valid; enforce domain limits in options validation.
func NewDuration() Codec[time.Duration, time.Duration] {
	return duration{}
}

type duration struct{}

func (this duration) ToRaw(ctx context.Context) Codec[any, any] { return Raw(ctx, this) }

func (duration) CanHandleType(ctx context.Context, typ reflect.Type) (bool, error) {
	return typ == reflect.TypeFor[time.Duration](), ctx.Err()
}

func (duration) ExpectMinMax(context.Context, Usage) (int, int) { return 1, 1 }

func (this duration) ParseRequirement(ctx context.Context, usage Usage, args ...string) (time.Duration, error) {
	if err := EnsureArgumentsMatching(ctx, usage, this, args...); err != nil {
		return 0, err
	}
	value, err := time.ParseDuration(args[0])
	if err != nil {
		return 0, NewValidationError("a duration with units ns, us, µs, ms, s, m or h expected (for example 1h30m); supported range: "+
			time.Duration(-1<<63).String()+" to "+time.Duration(1<<63-1).String(), ErrIllegalArgument, err)
	}
	return value, nil
}

func (duration) FormatRequirement(ctx context.Context, value time.Duration) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []string{value.String()}, nil
}

func (duration) CloneRequirement(ctx context.Context, value time.Duration) (time.Duration, error) {
	return value, ctx.Err()
}

func (duration) Resolve(ctx context.Context, _ Scope, value time.Duration) (time.Duration, error) {
	return value, ctx.Err()
}
