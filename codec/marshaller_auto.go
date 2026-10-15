package codec

import (
	"context"
	"encoding"
	"reflect"
)

// AutomaticMarshaller adapts a concrete T or *T whose pointer implements both
// encoding.TextMarshaler and encoding.TextUnmarshaler. The zero value must be
// ready for UnmarshalText. Detection does not call either text method; Resolve
// unmarshals into a fresh value and registers any owned resources with the scope.
// Requirements and defaults are input strings, as with [NewMarshaller].
func AutomaticMarshaller(ctx context.Context, typ reflect.Type) (Codec[any, any], bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	if typ == nil || typ.Kind() == reflect.Interface {
		return nil, false, nil
	}
	pointer := typ
	if typ.Kind() != reflect.Pointer {
		pointer = reflect.PointerTo(typ)
	}
	if !pointer.Implements(reflect.TypeFor[encoding.TextMarshaler]()) ||
		!pointer.Implements(reflect.TypeFor[encoding.TextUnmarshaler]()) {
		return nil, false, nil
	}
	decoder := textCodec[any]{typ: typ, resolve: func(ctx context.Context, scope Scope, input string) (any, error) {
		target := reflect.New(pointer.Elem())
		unmarshaler := target.Interface().(encoding.TextUnmarshaler)
		ownValue(scope, unmarshaler)
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		err := unmarshaler.UnmarshalText([]byte(input))
		if err == nil {
			err = ctx.Err()
		}
		if typ.Kind() == reflect.Pointer {
			return target.Interface(), err
		}
		return target.Elem().Interface(), err
	}}
	if err := ctx.Err(); err != nil {
		return nil, true, err
	}
	return Raw[any, string](ctx, decoder), true, nil
}
