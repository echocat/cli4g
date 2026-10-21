package codec

import (
	"context"
	"fmt"
	"reflect"
)

type raw[V, R any] struct{ source Codec[V, R] }

// Raw adapts a typed codec for registry use. Unmarked errors receive neutral
// messages while their causes and classification are retained.
// Custom wrappers should implement ToRaw as Raw(ctx, wrapper), not delegate to
// an embedded codec. Nil codecs and canceled contexts return nil.
// Warning: ValidationError messages are displayed verbatim; authors must keep
// supplied inputs and confidential data out of those messages.
func Raw[V, R any](ctx context.Context, source Codec[V, R]) Codec[any, any] {
	if ctx.Err() != nil || isNil(source) {
		return nil
	}
	result := raw[V, R]{source}
	if enumeration, ok := source.(Enum[R]); ok {
		enum := rawEnum[V, R]{raw: result, enumeration: enumeration}
		if binder, ok := source.(TypeBinder); ok {
			return rawEnumBinder[V, R]{rawEnum: enum, binder: binder}
		}
		return enum
	}
	if binder, ok := source.(TypeBinder); ok {
		return rawBinder[V, R]{raw: result, binder: binder}
	}
	return result
}

type rawBinder[V, R any] struct {
	raw[V, R]
	binder TypeBinder
}

func (this rawBinder[V, R]) BindType(ctx context.Context, typ reflect.Type) (Codec[any, any], bool, error) {
	return this.binder.BindType(ctx, typ)
}

func (this rawBinder[V, R]) ToRaw(ctx context.Context) Codec[any, any] {
	if ctx.Err() != nil {
		return nil
	}
	return this
}

type rawEnumBinder[V, R any] struct {
	rawEnum[V, R]
	binder TypeBinder
}

func (this rawEnumBinder[V, R]) BindType(ctx context.Context, typ reflect.Type) (Codec[any, any], bool, error) {
	return this.binder.BindType(ctx, typ)
}

func (this rawEnumBinder[V, R]) ToRaw(ctx context.Context) Codec[any, any] {
	if ctx.Err() != nil {
		return nil
	}
	return this
}

// Only actual enums expose Enum[any]. Collection choices use a separate
// projection, so adapting a collection never claims to enumerate all collections.
type rawEnum[V, R any] struct {
	raw[V, R]
	enumeration Enum[R]
}

func (this rawEnum[V, R]) ToRaw(ctx context.Context) Codec[any, any] {
	if ctx.Err() != nil {
		return nil
	}
	return this
}

func (this rawEnum[V, R]) EnumVariants(ctx context.Context) ([]EnumVariant[any], error) {
	variants, err := this.enumeration.EnumVariants(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]EnumVariant[any], len(variants))
	for i, variant := range variants {
		if err := ValidateRequirement(variant.Value); err != nil {
			return nil, err
		}
		value, err := CloneRequirement(ctx, this.source, variant.Value)
		if err != nil {
			return nil, err
		}
		result[i] = EnumVariant[any]{Value: value, Description: variant.Description}
	}
	return result, ctx.Err()
}

func (this raw[V, R]) EnumChoices(ctx context.Context) ([]EnumChoice, error) {
	return DisplayEnum(ctx, this.source)
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
	value, err := this.source.ParseRequirement(ctx, usage, args...)
	return value, SanitizeError(err, "value could not be decoded")
}

func (this raw[V, R]) FormatRequirement(ctx context.Context, value any) ([]string, error) {
	typed, err := requirementAs[R](value)
	if err != nil {
		return nil, err
	}
	result, err := this.source.FormatRequirement(ctx, typed)
	return result, SanitizeError(err, "value could not be formatted")
}

func (this raw[V, R]) DisplayRequirement(ctx context.Context, value any) ([]string, error) {
	typed, err := requirementAs[R](value)
	if err != nil {
		return nil, err
	}
	result, err := DisplayRequirement(ctx, this.source, typed)
	return result, SanitizeError(err, "value could not be displayed")
}

func (this raw[V, R]) CloneRequirement(ctx context.Context, value any) (any, error) {
	typed, err := requirementAs[R](value)
	if err != nil {
		return nil, err
	}
	result, err := CloneRequirement(ctx, this.source, typed)
	return result, SanitizeError(err, "value could not be copied")
}

func (this raw[V, R]) Resolve(ctx context.Context, scope Scope, value any) (any, error) {
	typed, err := requirementAs[R](value)
	if err != nil {
		return nil, err
	}
	result, err := this.source.Resolve(ctx, scope, typed)
	return result, SanitizeError(err, "value could not be resolved")
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
	result, err := decoder.ParseRequirementInto(ctx, usage, typed, args...)
	return result, SanitizeError(err, "value could not be decoded")
}

func (this raw[V, R]) RequirementType() reflect.Type {
	typ := reflect.TypeFor[R]()
	if typ.Kind() == reflect.Interface {
		if source, ok := this.source.(interface{ RequirementType() reflect.Type }); ok {
			return source.RequirementType()
		}
	}
	return typ
}

func (this raw[V, R]) IsBoolFlag() bool {
	if reflect.TypeFor[V]().Kind() == reflect.Bool {
		return true
	}
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
