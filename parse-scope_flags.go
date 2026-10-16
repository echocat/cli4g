package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/echocat/cli4g/codec"
)

// Bind path-specific definitions and count distinct CLI flags using the existing
// invocation-local index. Zero entries are still unprovided: only successful
// decoding assigns their index.
// Repeated flags and unselected definitions do not inflate the reservation.
func prepareFlagEntries(ctx context.Context, scopes []*parseScope, flags []flagInput) error {
	if len(flags) == 0 {
		return nil
	}
	var previous *Flag[any, any]
	var resolved *Flag[any, any]
	for i := range flags {
		if err := ctx.Err(); err != nil {
			return err
		}
		flag := &flags[i]
		if flag.definition == previous {
			flag.definition = resolved
			continue
		}
		previous = flag.definition
		owner := scopes[flag.scope]
		resolved = owner.schema.flagByName[flag.definition.Name()]
		flag.definition = resolved
		if len(flags) < 2 || len(owner.schema.flags) < 2 {
			continue
		}
		if owner.provided == nil {
			owner.provided = make(map[*Flag[any, any]]int)
		}
		owner.provided[resolved] = 0
	}
	for _, scope := range scopes {
		if count := len(scope.provided); count > 1 {
			scope.flags = make([]flagEntry[any], 0, min(count, 128))
		}
	}
	return ctx.Err()
}

func (this *parseScope) addFlag(ctx context.Context, f *Flag[any, any], decoder codec.Codec[any, any], args []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	previous := this.provided[f]
	if previous != 0 && !f.cumulative {
		return NewValidationErrorf("flag --%s supplied more than once", f.Name())
	}
	if previous != 0 && f.cumulative {
		cumulative, ok := decoder.(codec.Cumulative[any])
		if !ok {
			return fmt.Errorf("flag --%s: cumulative codec clone lost ParseInto support", f.Name())
		}
		v, parseErr := cumulative.ParseRequirementInto(ctx, codec.UsageFlag, this.flags[previous-1].requirement, args...)
		// Even a failing Set can attach new resources or replace slice/map
		// headers. Bind the partial aggregate before invocation rollback.
		bindErr := this.storeFlag(ctx, f, v, previous)
		if parseErr != nil {
			return errors.Join(fmt.Errorf("flag --%s: %w", f.Name(), codec.SanitizeError(parseErr, "value could not be decoded")), bindErr)
		}
		return bindErr
	}
	v, err := decoder.ParseRequirement(ctx, codec.UsageFlag, args...)
	if err != nil {
		return fmt.Errorf("flag --%s: %w", f.Name(), codec.SanitizeError(err, "value could not be decoded"))
	}
	return this.storeFlag(ctx, f, v, previous)
}

func (this *parseScope) storeFlag(ctx context.Context, f *Flag[any, any], v any, previous int) error {
	if err := codec.ValidateRequirement(v); err != nil {
		return fmt.Errorf("flag --%s: %w", f.Name(), err)
	}
	if previous != 0 {
		this.flags[previous-1].requirement = v
	} else {
		this.flags = append(this.flags, flagEntry[any]{flag: f, requirement: v})
		if this.provided == nil {
			this.provided = make(map[*Flag[any, any]]int)
		}
		this.provided[f] = len(this.flags)
	}
	// Retain decoded values before reporting cancellation so owned resources
	// remain reachable by the invocation's cleanup.
	return ctx.Err()
}

func (this *parseScope) finishFlags(ctx context.Context, opts *ParseOpts) error {
	for _, f := range this.schema.flags {
		if err := ctx.Err(); err != nil {
			return err
		}
		if this.provided[f] == 0 {
			for _, name := range f.envVars {
				value, ok := opts.lookupEnv(ctx, name)
				if err := ctx.Err(); err != nil {
					return err
				}
				if ok {
					if err := this.addFlag(ctx, f, f.codec, []string{value}); err != nil {
						return err
					}
					break
				}
			}
		}
		if f.HasDefault() && this.provided[f] == 0 {
			requirement, _, err := f.Default(ctx)
			if err != nil {
				return fmt.Errorf("flag --%s default: %w", f.Name(), err)
			}
			if err := this.storeFlag(ctx, f, requirement, 0); err != nil {
				return err
			}
		}
		if f.required && this.provided[f] == 0 {
			return NewValidationErrorf("required flag --%s is missing", f.Name())
		}
	}
	return ctx.Err()
}
