package codec

import (
	"context"
	"reflect"
)

func NewString() Codec[string, string] {
	return stringValue{}
}

type stringValue struct{}

func (this stringValue) ToRaw(ctx context.Context) Codec[any, any] { return Raw(ctx, this) }

func (this stringValue) CanHandleType(ctx context.Context, t reflect.Type) (bool, error) {
	return t == reflect.TypeFor[string](), ctx.Err()
}

func (this stringValue) ExpectMinMax(context.Context, Usage) (int, int) {
	return 1, 1
}

func (this stringValue) ParseRequirement(ctx context.Context, usg Usage, args ...string) (string, error) {
	if err := EnsureArgumentsMatching(ctx, usg, this, args...); err != nil {
		return "", err
	}
	return args[0], nil
}

func (this stringValue) FormatRequirement(_ context.Context, v string) ([]string, error) {
	return []string{v}, nil
}

func (stringValue) CloneRequirement(ctx context.Context, value string) (string, error) {
	return value, ctx.Err()
}

func (stringValue) Resolve(ctx context.Context, _ Scope, value string) (string, error) {
	return value, ctx.Err()
}
