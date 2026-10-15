package codec

import (
	"context"
	"fmt"
	"reflect"
	"slices"
)

// ValueTraits explicitly enables optional bool flag input or repeated occurrences.
type ValueTraits struct{ BoolFlag, Cumulative bool }

// ValueConfig overrides traits when a zero value is not suitable for trait detection.
// At most one config is accepted; Traits=nil uses IsBoolFlag/IsCumulative methods.
type ValueConfig[T any] struct{ Traits *ValueTraits }

// FlagValueTraits names the traits used by [NewFlagValue].
type FlagValueTraits = ValueTraits

// FlagValueConfig configures [NewFlagValue] and [NewPointerFlagValue].
type FlagValueConfig[T any] = ValueConfig[T]

// NewFlagValue creates a Set/String adapter to keep resource-free input tokens
// as their requirements. String is a display method, not a lossless serialization
// of arbitrary T.
// Set must explicitly mark invalid input; the adapter preserves returned errors.
func NewFlagValue[T any, TP interface {
	*T
	Value
}](configs ...ValueConfig[T]) (Codec[T, []string], error) {
	traits, err := configuredValueTraits(TP(new(T)), configs)
	if err != nil {
		return nil, err
	}
	return tokenValue[T]{typ: reflect.TypeFor[T](), traits: traits, resolve: func(ctx context.Context, scope Scope, inputs []string) (T, error) {
		var target TP = new(T)
		ownValue(scope, target)
		for _, input := range inputs {
			if err := ctx.Err(); err != nil {
				return *target, err
			}
			if err := target.Set(input); err != nil {
				return *target, err
			}
		}
		return *target, ctx.Err()
	}}, nil
}

// NewPointerFlagValue adapts a fallible, concurrent-safe factory of fresh pointer values.
// Resources in non-nil partial results are registered for cleanup even on error.
// Requirements are Set input tokens; see [NewValueP] for an infallible factory.
func NewPointerFlagValue[T Value](factory func() (T, error), configs ...ValueConfig[T]) (Codec[T, []string], error) {
	if factory == nil {
		return nil, fmt.Errorf("%w: nil value factory", ErrInvalid)
	}
	typ := reflect.TypeFor[T]()
	if typ.Kind() != reflect.Pointer {
		return nil, fmt.Errorf("%w: value factory requires a concrete pointer", ErrInvalid)
	}
	probe := reflect.New(typ.Elem()).Interface().(Value)
	traits, err := configuredValueTraits(probe, configs)
	if err != nil {
		return nil, err
	}
	return tokenValue[T]{typ: typ, traits: traits, resolve: func(ctx context.Context, scope Scope, inputs []string) (T, error) {
		target, err := factory()
		if !isNil(target) {
			ownValue(scope, target)
		}
		if err != nil {
			return target, err
		}
		if isNil(target) {
			return target, fmt.Errorf("%w: value factory returned nil", ErrInvalid)
		}
		for _, input := range inputs {
			if err := ctx.Err(); err != nil {
				return target, err
			}
			if err := target.Set(input); err != nil {
				return target, err
			}
		}
		return target, ctx.Err()
	}}, nil
}

type tokenValue[T any] struct {
	typ     reflect.Type
	traits  ValueTraits
	resolve func(context.Context, Scope, []string) (T, error)
}

func (this tokenValue[T]) ToRaw(ctx context.Context) Codec[any, any] { return Raw(ctx, this) }
func (this tokenValue[T]) CanHandleType(ctx context.Context, typ reflect.Type) (bool, error) {
	return typ == this.typ, ctx.Err()
}
func (this tokenValue[T]) IsBoolFlag() bool   { return this.traits.BoolFlag }
func (this tokenValue[T]) IsCumulative() bool { return this.traits.Cumulative }
func (this tokenValue[T]) ExpectMinMax(_ context.Context, usage Usage) (int, int) {
	if usage == UsageDefault && this.traits.Cumulative {
		return 0, -1
	}
	if usage == UsageFlag && this.traits.BoolFlag {
		return 0, 1
	}
	return 1, 1
}
func (this tokenValue[T]) ParseRequirement(ctx context.Context, usage Usage, args ...string) ([]string, error) {
	if err := EnsureArgumentsMatching(ctx, usage, this, args...); err != nil {
		return nil, err
	}
	if usage == UsageFlag && len(args) == 0 {
		return []string{"true"}, nil
	}
	return slices.Clone(args), nil
}

func (this tokenValue[T]) FormatRequirement(ctx context.Context, args []string) ([]string, error) {
	if err := EnsureArgumentsMatching(ctx, UsageDefault, this, args...); err != nil {
		return nil, err
	}
	return slices.Clone(args), ctx.Err()
}

func (this tokenValue[T]) CloneRequirement(ctx context.Context, args []string) ([]string, error) {
	if err := EnsureArgumentsMatching(ctx, UsageDefault, this, args...); err != nil {
		return nil, err
	}
	return slices.Clone(args), ctx.Err()
}

func (this tokenValue[T]) ParseRequirementInto(ctx context.Context, usage Usage, previous []string, args ...string) ([]string, error) {
	additional, err := this.ParseRequirement(ctx, usage, args...)
	if err != nil {
		return previous, err
	}
	return append(previous, additional...), nil
}

func (this tokenValue[T]) Resolve(ctx context.Context, scope Scope, args []string) (T, error) {
	if isNil(scope) {
		var zero T
		return zero, fmt.Errorf("%w: value resolution requires a scope", ErrInvalid)
	}
	if err := ctx.Err(); err != nil {
		var zero T
		return zero, err
	}
	return this.resolve(ctx, scope, args)
}

func configuredValueTraits[T any](probe Value, configs []ValueConfig[T]) (traits ValueTraits, err error) {
	if len(configs) > 1 {
		return traits, fmt.Errorf("at most one value configuration is allowed")
	}
	if len(configs) == 1 && configs[0].Traits != nil {
		return *configs[0].Traits, nil
	}
	defer func() {
		if failure := recover(); failure != nil {
			err = fmt.Errorf("%w: zero-instance traits: %v; use explicit Traits", ErrInvalid, failure)
		}
	}()
	if trait, ok := probe.(interface{ IsBoolFlag() bool }); ok {
		traits.BoolFlag = trait.IsBoolFlag()
	}
	if trait, ok := probe.(interface{ IsCumulative() bool }); ok {
		traits.Cumulative = trait.IsCumulative()
	}
	return traits, nil
}
