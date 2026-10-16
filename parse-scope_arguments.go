package cli

import (
	"context"
	"fmt"

	"github.com/echocat/cli4g/codec"
)

func (this *parseScope) parseArguments(ctx context.Context) error {
	position := 0
	args := this.schema.arguments
	for i, a := range args {
		if err := ctx.Err(); err != nil {
			return err
		}
		//goland:noinspection GoShadowedVar,GoReservedWordUsedAsName
		max := a.arity.max
		count := 1
		if i == len(args)-1 && max != 1 {
			count = len(this.positions) - position
			if max >= 0 && count > max {
				count = max
			}
		}
		if position+count > len(this.positions) {
			count = len(this.positions) - position
		}
		if count == 0 {
			if a.required {
				return NewValidationErrorf("required argument %s is missing", a.name)
			}
			if a.HasDefault() {
				requirement, _, err := a.Default(ctx)
				if err != nil {
					return fmt.Errorf("argument %s default: %w", a.name, err)
				}
				this.arguments = append(this.arguments, argumentEntry[any]{argument: a, requirement: requirement})
				if err := ctx.Err(); err != nil {
					return err
				}
			}
			continue
		}
		if err := a.arity.check(count); err != nil {
			return NewValidationErrorf("argument %s: %w", a.name, err)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		v, err := a.codec.ParseRequirement(ctx, codec.UsageArgument, this.positions[position:position+count]...)
		if err != nil {
			return fmt.Errorf("argument %s: %w", a.name, codec.SanitizeError(err, "value could not be decoded"))
		}
		if err := codec.ValidateRequirement(v); err != nil {
			return fmt.Errorf("argument %s: %w", a.name, err)
		}
		if this.arguments == nil {
			// One entry per argument definition, not per variadic value. Reserve
			// only once decoding succeeded; omitted arguments need no buffer.
			capacity := min(len(args)-i, len(this.positions)-position, 128)
			if capacity > 1 {
				this.arguments = make([]argumentEntry[any], 0, capacity)
			}
		}
		this.arguments = append(this.arguments, argumentEntry[any]{argument: a, requirement: v})
		position += count
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	if position != len(this.positions) {
		return NewValidationErrorf("%w: %d unexpected values", codec.ErrTooManyArguments, len(this.positions)-position)
	}
	return ctx.Err()
}
