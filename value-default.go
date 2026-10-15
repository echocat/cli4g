package cli

import (
	"context"
	"slices"

	"github.com/echocat/cli4g/codec"
)

// Defaults hold resource-free requirements. Presence is independent of zero,
// nil and empty. String defaults remain lazy; typed defaults snapshot at Create.
type valueDefault[R any] struct {
	value    R
	strings  []string
	isString bool
	owner    *definitionLifecycle
}

func rawDefault[R any](source *valueDefault[R]) *valueDefault[any] {
	if source == nil {
		return nil
	}
	return &valueDefault[any]{value: source.value, strings: slices.Clone(source.strings), isString: source.isString}
}
func snapshotDefault(source *valueDefault[any], selected codec.Codec[any, any], _ interface{}, owner *definitionLifecycle) (*valueDefault[any], error) {
	if source == nil {
		return nil, nil
	}
	result := rawDefault(source)
	result.owner = owner
	if source.isString {
		return result, nil
	}
	if err := codec.ValidateRequirement(source.value); err != nil {
		return nil, err
	}
	value, err := codec.CloneRequirement(context.Background(), selected, source.value)
	if err != nil {
		return nil, err
	}
	if err := codec.ValidateRequirement(value); err != nil {
		return nil, err
	}
	result.value = value
	return result, nil
}
func resolveDefault[V, R any](ctx context.Context, source *valueDefault[R], selected codec.Codec[V, R], _ interface{}) (R, bool, error) {
	var zero R
	if source == nil {
		return zero, false, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := source.owner.beginRead(); err != nil {
		return zero, true, err
	}
	defer source.owner.endRead()
	if err := ctx.Err(); err != nil {
		return zero, true, err
	}
	var result R
	var err error
	if source.isString {
		result, err = selected.ParseRequirement(ctx, codec.UsageDefault, slices.Clone(source.strings)...)
	} else {
		result, err = codec.CloneRequirement(ctx, selected, source.value)
	}
	if err == nil {
		err = ctx.Err()
	}
	if err == nil {
		err = codec.ValidateRequirement(result)
	}
	return result, true, err
}
func formatDefault[V, R any](ctx context.Context, source *valueDefault[R], selected codec.Codec[V, R], typ interface{}) ([]string, bool, error) {
	requirement, found, err := resolveDefault(ctx, source, selected, typ)
	if err != nil || !found {
		return nil, found, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	plain, err := codec.DisplayRequirement(ctx, selected, requirement)
	if err == nil {
		err = ctx.Err()
	}
	return slices.Clone(plain), true, err
}
