package codec

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// NewSlice composes an element codec into a repeatable []V codec with []R defaults.
// A flag occurrence takes one element; arguments/defaults take zero or more.
// Slice arguments must be last. Nil element codecs panic; nil/empty defaults stay distinct.
func NewSlice[V, R any](element Codec[V, R]) Codec[[]V, []R] {
	if isNil(element) {
		panic("slice codec requires an element codec")
	}
	return slice[V, R]{element}
}

// NewStrings creates a string slice codec; see [NewStringSlice].
func NewStrings() Codec[[]string, []string] { return NewStringSlice() }

type slice[V, R any] struct{ element Codec[V, R] }

func (this slice[V, R]) EnumChoices(ctx context.Context) ([]EnumChoice, error) {
	return DisplayEnum(ctx, this.element)
}

func (slice[V, R]) IsCumulative() bool { return true }

func (this slice[V, R]) ParseRequirementInto(ctx context.Context, usage Usage, previous []R, args ...string) ([]R, error) {
	additional, err := this.ParseRequirement(ctx, usage, args...)
	if err != nil {
		return previous, err
	}
	return append(previous, additional...), ctx.Err()
}

func (this slice[V, R]) ToRaw(ctx context.Context) Codec[any, any] { return Raw(ctx, this) }

func (this slice[V, R]) CanHandleType(ctx context.Context, typ reflect.Type) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if typ != reflect.TypeFor[[]V]() {
		return false, nil
	}
	return this.element.CanHandleType(ctx, reflect.TypeFor[V]())
}

func (slice[V, R]) ExpectMinMax(_ context.Context, usage Usage) (int, int) {
	if usage == UsageFlag {
		return 1, 1
	}
	return 0, -1
}

func (this slice[V, R]) ParseRequirement(ctx context.Context, usage Usage, args ...string) ([]R, error) {
	if err := EnsureArgumentsMatching(ctx, usage, this, args...); err != nil {
		return nil, err
	}
	if args == nil {
		return nil, nil
	}
	if usage == UsageDefault && len(args) == 1 && strings.HasPrefix(args[0], "slice+json:") {
		var groups [][][]byte
		if err := json.Unmarshal([]byte(strings.TrimPrefix(args[0], "slice+json:")), &groups); err != nil {
			return nil, fmt.Errorf("%w: slice requirement: %w", ErrIllegalArgument, err)
		}
		if groups == nil {
			return nil, ctx.Err()
		}
		result := make([]R, len(groups))
		for i, group := range groups {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			var inputs []string
			if group != nil {
				inputs = make([]string, len(group))
				for j, input := range group {
					inputs[j] = string(input)
				}
			}
			value, err := this.element.ParseRequirement(ctx, UsageDefault, inputs...)
			if err != nil {
				return nil, fmt.Errorf("slice element %d: %w", i, err)
			}
			result[i] = value
		}
		return result, ctx.Err()
	}
	result := make([]R, len(args))
	for i, arg := range args {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		value, err := this.element.ParseRequirement(ctx, usage, arg)
		if err != nil {
			return nil, fmt.Errorf("slice element %d: %w", i, err)
		}
		result[i] = value
	}
	return result, ctx.Err()
}

func (this slice[V, R]) FormatRequirement(ctx context.Context, values []R) ([]string, error) {
	if values == nil {
		return nil, ctx.Err()
	}
	result := make([]string, len(values))
	groups := make([][][]byte, len(values))
	structured := false
	for i, value := range values {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		plain, err := this.element.FormatRequirement(ctx, value)
		if err != nil {
			return nil, fmt.Errorf("slice element %d: %w", i, err)
		}
		if plain != nil {
			groups[i] = make([][]byte, len(plain))
			for j, input := range plain {
				groups[i][j] = []byte(input)
			}
		}
		if len(plain) != 1 {
			structured = true
		} else {
			result[i] = plain[0]
		}
	}
	if len(result) == 1 && strings.HasPrefix(result[0], "slice+json:") {
		structured = true
	}
	if structured {
		encoded, err := json.Marshal(groups)
		if err != nil {
			return nil, err
		}
		return []string{"slice+json:" + string(encoded)}, ctx.Err()
	}
	return result, ctx.Err()
}

func (this slice[V, R]) DisplayRequirement(ctx context.Context, values []R) ([]string, error) {
	if values == nil {
		return nil, ctx.Err()
	}
	var result []string
	for _, value := range values {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		plain, err := DisplayRequirement(ctx, this.element, value)
		if err != nil {
			return nil, err
		}
		result = append(result, plain...)
	}
	if result == nil {
		return []string{}, nil
	}
	return result, ctx.Err()
}

func (this slice[V, R]) CloneRequirement(ctx context.Context, values []R) ([]R, error) {
	if values == nil {
		return nil, ctx.Err()
	}
	result := make([]R, len(values))
	for i, value := range values {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		cloned, err := CloneRequirement(ctx, this.element, value)
		if err != nil {
			return nil, fmt.Errorf("slice element %d: %w", i, err)
		}
		result[i] = cloned
	}
	return result, ctx.Err()
}

func (this slice[V, R]) Resolve(ctx context.Context, scope Scope, requirements []R) ([]V, error) {
	if requirements == nil {
		return nil, ctx.Err()
	}
	values := make([]V, len(requirements))
	for i, requirement := range requirements {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		value, err := this.element.Resolve(ctx, scope, requirement)
		if err != nil {
			return nil, fmt.Errorf("slice element %d: %w", i, err)
		}
		values[i] = value
	}
	return values, ctx.Err()
}
