package cli

import (
	"context"
	"fmt"

	"github.com/echocat/cli4g/codec"
)

// valueArity is resolved by Create, independently of each invocation's context.
type valueArity struct {
	min int
	max int
}

func compileValueArity(ctx context.Context, decoder codec.Codec[any, any], usage codec.Usage) (valueArity, error) {
	if err := ctx.Err(); err != nil {
		return valueArity{}, err
	}
	//goland:noinspection GoShadowedVar,GoReservedWordUsedAsName
	min, max := decoder.ExpectMinMax(ctx, usage)
	if err := ctx.Err(); err != nil {
		return valueArity{}, err
	}
	if min < 0 || max < -1 || (max != -1 && min > max) {
		return valueArity{}, fmt.Errorf("invalid codec arity %d,%d", min, max)
	}
	if usage == codec.UsageArgument && max == 0 {
		return valueArity{}, fmt.Errorf("positional argument cannot have zero maximum arity")
	}
	return valueArity{min: min, max: max}, nil
}

func (this valueArity) check(count int) error {
	if this.max == 0 && count > 0 {
		return codec.ErrNoArgumentExpected
	}
	if this.max >= 0 && count > this.max {
		return fmt.Errorf("%w: maximum is %d; got %d", codec.ErrTooManyArguments, this.max, count)
	}
	if count < this.min {
		if count == 0 {
			return codec.ErrArgumentRequired
		}
		return fmt.Errorf("%w: minimum is %d; got %d", codec.ErrTooFewArguments, this.min, count)
	}
	return nil
}
