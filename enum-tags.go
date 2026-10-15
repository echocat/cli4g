package cli

import (
	"context"
	"reflect"
	"strings"

	"github.com/echocat/cli4g/codec"
)

func enumTagCodec(field reflect.StructField) (codec.Codec[any, any], error) {
	values, exists := field.Tag.Lookup("enum")
	if !exists {
		return nil, nil
	}
	if field.Type.Kind() != reflect.String {
		return nil, errSchema("field %s: enum requires a string or string-based type", field.Name)
	}
	if field.Tag.Get("type") != "" {
		return nil, errSchema("field %s: enum cannot be combined with a named type codec", field.Name)
	}
	ctx := context.Background()
	source := taggedEnumString{Codec: codec.NewString().ToRaw(ctx), typ: field.Type}
	plain := strings.Split(values, "|")
	requirements := make([]any, len(plain))
	for i, value := range plain {
		requirements[i] = value
	}
	result, err := codec.NewEnum[any, any](source, requirements...)
	if err != nil {
		return nil, errSchema("field %s: enum: %v", field.Name, err)
	}
	return result.ToRaw(ctx), nil
}

// String-based tags use their literal names. Richer typed mappings and custom
// Set/UnmarshalText policies are selected explicitly with an enum codec.
type taggedEnumString struct {
	codec.Codec[any, any]
	typ reflect.Type
}

func (this taggedEnumString) ToRaw(ctx context.Context) codec.Codec[any, any] {
	if ctx.Err() != nil {
		return nil
	}
	return this
}

func (this taggedEnumString) CanHandleType(ctx context.Context, typ reflect.Type) (bool, error) {
	return typ == this.typ, ctx.Err()
}

func (this taggedEnumString) Resolve(ctx context.Context, scope codec.Scope, requirement any) (any, error) {
	value, err := this.Codec.Resolve(ctx, scope, requirement)
	if err != nil {
		return nil, err
	}
	result := reflect.New(this.typ).Elem()
	result.SetString(value.(string))
	return result.Interface(), ctx.Err()
}
