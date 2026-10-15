package codec

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

func NewBool() Codec[bool, bool] {
	return boolean{}
}

type boolean struct{}

func (this boolean) ToRaw(ctx context.Context) Codec[any, any] { return Raw(ctx, this) }

func (this boolean) CanHandleType(ctx context.Context, t reflect.Type) (bool, error) {
	return t == reflect.TypeFor[bool](), ctx.Err()
}

func (this boolean) ExpectMinMax(_ context.Context, usg Usage) (int, int) {
	switch usg {
	case UsageFlag:
		return 0, 1
	default:
		return 1, 1
	}
}

func (this boolean) ParseRequirement(ctx context.Context, usg Usage, args ...string) (bool, error) {
	if err := EnsureArgumentsMatching(ctx, usg, this, args...); err != nil {
		return false, err
	}
	switch len(args) {
	case 0:
		return true, nil
	default:
		switch strings.ToLower(args[0]) {
		case "true", "1", "on", "yes":
			return true, nil
		case "false", "0", "off", "no", "":
			return false, nil
		default:
			return false, fmt.Errorf("%w: %s", ErrIllegalArgument, args[0])
		}
	}
}

func (this boolean) FormatRequirement(_ context.Context, v bool) ([]string, error) {
	if v {
		return []string{"true"}, nil
	}
	return []string{"false"}, nil
}

func (boolean) CloneRequirement(ctx context.Context, value bool) (bool, error) {
	return value, ctx.Err()
}

func (boolean) Resolve(ctx context.Context, _ Scope, value bool) (bool, error) {
	return value, ctx.Err()
}
