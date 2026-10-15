package codec

import (
	"context"
	"fmt"
	"reflect"
)

type raw[V, R any] struct{ source Codec[V, R] }

// Raw adapts a typed codec for registry use without changing its behavior.
// Custom wrappers should implement ToRaw as Raw(ctx, wrapper), not delegate to
// an embedded codec. Nil codecs and canceled contexts return nil.
func Raw[V, R any](ctx context.Context, source Codec[V, R]) Codec[any, any] {
	if ctx.Err() != nil || isNil(source) {
		return nil
	}
	return raw[V, R]{source}
}

func (this raw[V, R]) ToRaw(ctx context.Context) Codec[any, any] {
	if ctx.Err() != nil {
		return nil
	}
	return this
}

func (this raw[V, R]) CanHandleType(ctx context.Context, t reflect.Type) (bool, error) {
	return this.source.CanHandleType(ctx, t)
}

func (this raw[V, R]) ExpectMinMax(ctx context.Context, usage Usage) (int, int) {
	return this.source.ExpectMinMax(ctx, usage)
}

func (this raw[V, R]) ParseRequirement(ctx context.Context, usage Usage, args ...string) (any, error) {
	return this.source.ParseRequirement(ctx, usage, args...)
}

func (this raw[V, R]) FormatRequirement(ctx context.Context, value any) ([]string, error) {
	typed, err := requirementAs[R](value)
	if err != nil {
		return nil, err
	}
	return this.source.FormatRequirement(ctx, typed)
}

func (this raw[V, R]) DisplayRequirement(ctx context.Context, value any) ([]string, error) {
	typed, err := requirementAs[R](value)
	if err != nil {
		return nil, err
	}
	return DisplayRequirement(ctx, this.source, typed)
}

func (this raw[V, R]) CloneRequirement(ctx context.Context, value any) (any, error) {
	typed, err := requirementAs[R](value)
	if err != nil {
		return nil, err
	}
	return CloneRequirement(ctx, this.source, typed)
}

func (this raw[V, R]) Resolve(ctx context.Context, scope Scope, value any) (any, error) {
	typed, err := requirementAs[R](value)
	if err != nil {
		return nil, err
	}
	return this.source.Resolve(ctx, scope, typed)
}

func (this raw[V, R]) IsCumulative() bool { return IsCumulative(this.source) }

func (this raw[V, R]) ParseRequirementInto(ctx context.Context, usage Usage, previous any, args ...string) (any, error) {
	typed, err := requirementAs[R](previous)
	if err != nil {
		return nil, err
	}
	decoder, ok := this.source.(Cumulative[R])
	if !ok {
		return nil, fmt.Errorf("codec has no cumulative requirement parser")
	}
	return decoder.ParseRequirementInto(ctx, usage, typed, args...)
}

func (this raw[V, R]) RequirementType() reflect.Type { return reflect.TypeFor[R]() }

func (this raw[V, R]) IsBoolFlag() bool {
	trait, ok := this.source.(interface{ IsBoolFlag() bool })
	return ok && trait.IsBoolFlag()
}

func requirementAs[R any](value any) (R, error) {
	typed, ok := value.(R)
	if !ok && (value != nil || reflect.TypeFor[R]().Kind() != reflect.Interface) {
		return typed, fmt.Errorf("wrong requirement type: %T, expected %s", value, reflect.TypeFor[R]())
	}
	return typed, nil
}
