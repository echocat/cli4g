package codec

import (
	"context"
	"fmt"
	"reflect"
)

// Snapshot erases the types of an immutable codec without reflective calls.
func Snapshot(ctx context.Context, value any) (Codec[any, any], error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if isNil(value) {
		return nil, fmt.Errorf("%w: nil codec", ErrInvalid)
	}
	if raw, ok := value.(Codec[any, any]); ok {
		return checkedSnapshot(raw)
	}
	adapter, ok := value.(interface {
		ToRaw(context.Context) Codec[any, any]
	})
	if !ok {
		return nil, fmt.Errorf("%w: %T is not a codec", ErrInvalid, value)
	}
	result := adapter.ToRaw(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if isNil(result) {
		return nil, fmt.Errorf("%w: %T returned a nil raw codec", ErrInvalid, value)
	}
	return checkedSnapshot(result)
}

func checkedSnapshot(source Codec[any, any]) (Codec[any, any], error) {
	if schema, ok := source.(interface{ RequirementType() reflect.Type }); ok {
		if !policyForRequirement(schema.RequirementType()).valid {
			return nil, fmt.Errorf("%w: %s", ErrResourceRequirement, schema.RequirementType())
		}
	}
	return source, nil
}

func isNil(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Interface, reflect.Pointer, reflect.Func, reflect.Map, reflect.Slice, reflect.Chan:
		return v.IsNil()
	default:
		return false
	}
}
