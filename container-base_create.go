package cli

import (
	"context"
	"fmt"
	"reflect"
)

// Typed callbacks are captured by value here. The raw options are produced by
// this same definition's factory, so each assertion is at a controlled boundary.
func newContainerBase[O any](
	createOptions func(context.Context) (O, error),
	validate func(context.Context, O) error,
	initialize func(context.Context, O) error,
	handler func(context.Context, O) error,
	closeHandler func(O) error,
) *containerBase {
	typ := reflect.TypeFor[O]()
	base := typ
	if typ.Kind() == reflect.Pointer {
		base = typ.Elem()
	}
	d := &containerBase{
		commandsByName: make(map[string]CommandLike),
		newOptions: func() reflect.Value {
			value := reflect.New(base)
			if typ.Kind() != reflect.Pointer {
				return value.Elem()
			}
			if value.Type() != typ {
				return value.Convert(typ)
			}
			return value
		},
	}
	if createOptions != nil {
		d.createOptionsHandler = func(ctx context.Context) (reflect.Value, error) {
			options, err := createOptions(ctx)
			value := reflect.ValueOf(options)
			if typ.Kind() == reflect.Pointer {
				if value.IsNil() {
					if err == nil {
						err = fmt.Errorf("options factory returned a nil pointer")
					}
					return reflect.Value{}, err
				}
				return value, err
			}
			owned := reflect.New(typ).Elem()
			owned.Set(value)
			return owned, err
		}
	}
	if initialize != nil {
		d.initializeHandler = func(ctx context.Context, options any) error {
			return initialize(ctx, options.(O))
		}
	}
	if handler != nil {
		d.handler = func(ctx context.Context, options any) error {
			return handler(ctx, options.(O))
		}
	}
	if closeHandler != nil {
		d.closeHandler = func(options any) error {
			return closeHandler(options.(O))
		}
	}
	if validate != nil {
		d.validateHandler = func(ctx context.Context, options any) error {
			return validate(ctx, options.(O))
		}
	}
	return d
}
