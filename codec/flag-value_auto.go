package codec

import (
	"context"
	"reflect"
)

func AutomaticFlagValue(ctx context.Context, typ reflect.Type) (Codec[any, any], bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	if typ.Kind() == reflect.Interface {
		return nil, false, nil
	}
	pointer := typ
	if typ.Kind() != reflect.Pointer {
		pointer = reflect.PointerTo(typ)
	}
	if !pointer.Implements(reflect.TypeFor[Value]()) {
		return nil, false, nil
	}
	probe := reflect.New(pointer.Elem()).Interface().(Value)
	traits, err := configuredValueTraits[any](probe, nil)
	if err != nil {
		return nil, true, err
	}
	decoder := tokenValue[any]{typ: typ, traits: traits, resolve: func(ctx context.Context, scope Scope, args []string) (any, error) {
		target := reflect.New(pointer.Elem())
		setter := target.Interface().(Value)
		ownValue(scope, setter)
		for _, input := range args {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if err := setter.Set(input); err != nil {
				return nil, err
			}
		}
		if typ.Kind() == reflect.Pointer {
			return target.Interface(), ctx.Err()
		}
		return target.Elem().Interface(), ctx.Err()
	}}
	if err := ctx.Err(); err != nil {
		return nil, true, err
	}
	return Raw[any, []string](ctx, decoder), true, nil
}
